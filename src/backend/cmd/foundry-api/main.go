package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/alex-jienexa/foundry-marches/internal/characters"
	"github.com/alex-jienexa/foundry-marches/internal/discordapi"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/httpapi"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/relay"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	databaseURL := requiredEnv("DATABASE_URL")
	worldID := requiredEnv("FOUNDRY_WORLD_ID")
	relayURL := requiredEnv("FOUNDRY_RELAY_BASE_URL")
	clientID := requiredEnv("FOUNDRY_RELAY_CLIENT_ID")
	apiKey := requiredEnv("FOUNDRY_RELAY_API_KEY")
	syncToken := requiredEnv("FOUNDRY_SYNC_API_TOKEN")
	if len(syncToken) < 32 {
		return errors.New("FOUNDRY_SYNC_API_TOKEN must contain at least 32 characters")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(startupCtx); err != nil {
		return err
	}
	var configuredWorld string
	if err := db.QueryRowContext(startupCtx, "SELECT id::text FROM foundry_worlds WHERE id = $1", worldID).Scan(&configuredWorld); err != nil {
		return errors.New("FOUNDRY_WORLD_ID must reference an existing row in foundry_worlds")
	}

	relayClient, err := relay.NewClient(relay.Config{
		WorldID: worldID, BaseURL: relayURL, ClientID: clientID, APIKey: apiKey,
	})
	if err != nil {
		return err
	}
	repository, err := characters.NewFoundryRepository(db)
	if err != nil {
		return err
	}
	syncService, err := foundrysync.NewSyncService(worldID, relayClient, repository)
	if err != nil {
		return err
	}
	syncRunner := &coordinatedSyncRunner{db: db, inner: syncService}
	syncHandler, err := httpapi.NewHandler(syncRunner)
	if err != nil {
		return err
	}

	discordBotToken := requiredEnv("DISCORD_BOT_API_TOKEN")
	if len(discordBotToken) < 32 || subtle.ConstantTimeCompare([]byte(discordBotToken), []byte(syncToken)) == 1 {
		return errors.New("DISCORD_BOT_API_TOKEN must be 32+ characters and differ from FOUNDRY_SYNC_API_TOKEN")
	}
	discordGuildID := requiredEnv("DISCORD_GUILD_ID")
	discordAdmins := strings.Split(os.Getenv("DISCORD_ADMIN_USER_IDS"), ",")
	discordHandler, err := discordapi.NewHandler(db, repository, discordBotToken, discordGuildID, worldID, discordAdmins)
	if err != nil {
		return err
	}
	discordHandler.SetSyncService(syncRunner)

	mux := http.NewServeMux()
	mux.Handle("/api/internal/discord/integration-status", discordapi.ServiceTokenMiddleware(discordBotToken, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type componentStatus struct {
			OK     bool   `json:"ok"`
			Detail string `json:"detail,omitempty"`
		}
		status := struct {
			Database componentStatus `json:"database"`
			Relay    componentStatus `json:"relay"`
			Foundry  componentStatus `json:"foundry"`
		}{}

		dbCtx, dbCancel := context.WithTimeout(r.Context(), 2*time.Second)
		if err := db.PingContext(dbCtx); err != nil {
			status.Database = componentStatus{Detail: err.Error()}
		} else {
			status.Database = componentStatus{OK: true, Detail: "PostgreSQL доступен"}
		}
		dbCancel()

		relayCtx, relayCancel := context.WithTimeout(r.Context(), 4*time.Second)
		if err := relayClient.PingRelay(relayCtx); err != nil {
			status.Relay = componentStatus{Detail: err.Error()}
		} else {
			status.Relay = componentStatus{OK: true, Detail: "HTTP-сервис Relay отвечает"}
		}
		relayCancel()

		foundryCtx, foundryCancel := context.WithTimeout(r.Context(), 8*time.Second)
		actors, err := relayClient.ListActors(foundryCtx, worldID)
		if errors.Is(err, relay.ErrActorListLimitReached) {
			// A full result page still proves that Foundry answered the search.
			err = nil
		}
		if err != nil {
			status.Foundry = componentStatus{Detail: err.Error()}
		} else {
			status.Foundry = componentStatus{OK: true, Detail: fmt.Sprintf("Foundry отвечает; Actor найдено: %d", len(actors))}
		}
		foundryCancel()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(status)
	})))
	mux.Handle(httpapi.SyncPath, bearerTokenMiddleware(syncToken, syncHandler))

	mux.Handle("/api/internal/discord/", discordapi.ServiceTokenMiddleware(discordBotToken, discordHandler))
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\\n"))
	})
	readyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	server := &http.Server{
		Addr: address, Handler: mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	syncInterval, err := foundrySyncInterval()
	if err != nil {
		return err
	}
	syncEnabled, err := foundrySyncEnabled()
	if err != nil {
		return err
	}
	if syncEnabled {
		go runPeriodicFoundrySync(stopCtx, syncRunner, syncInterval)
	} else {
		log.Printf("Foundry periodic sync is disabled by FOUNDRY_SYNC_ENABLED=false")
	}
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Foundry integration API listening on %s", address)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stopCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func foundrySyncInterval() (time.Duration, error) {
	seconds := 300
	if raw := strings.TrimSpace(os.Getenv("FOUNDRY_SYNC_INTERVAL_SECONDS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 30 || parsed > 86400 {
			return 0, errors.New("FOUNDRY_SYNC_INTERVAL_SECONDS must be between 30 and 86400")
		}
		seconds = parsed
	}
	return time.Duration(seconds) * time.Second, nil
}

func foundrySyncEnabled() (bool, error) {
	raw := strings.TrimSpace(os.Getenv("FOUNDRY_SYNC_ENABLED"))
	if raw == "" {
		return true, nil
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return false, errors.New("FOUNDRY_SYNC_ENABLED must be true or false")
	}
	return enabled, nil
}

func runPeriodicFoundrySync(ctx context.Context, syncer interface {
	SyncAll(context.Context) (foundrysync.SyncReport, error)
}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Printf("Foundry periodic sync enabled; interval=%s", interval)

	// Run once immediately after API startup. The PostgreSQL advisory lock
	// ensures only one instance performs this initial refresh.
	run := func() {
		syncCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		report, err := syncer.SyncAll(syncCtx)
		if err != nil {
			if !errors.Is(err, foundrysync.ErrSyncInProgress) {
				log.Printf("Foundry background sync failed: %v", err)
			}
			return
		}
		log.Printf("Foundry background sync finished: discovered=%d characters=%d snapshots=%d failed=%d",
			report.Discovered, report.CharactersSynced, report.SnapshotsOnly, report.Failed)
	}
	run()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func requiredEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return value
}

func bearerTokenMiddleware(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) ||
			subtle.ConstantTimeCompare([]byte(strings.TrimSpace(strings.TrimPrefix(header, prefix))), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
