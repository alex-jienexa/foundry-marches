package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync"
)

const SyncPath = "/api/integrations/foundry/sync"

type SyncRunner interface {
	SyncAll(context.Context) (foundrysync.SyncReport, error)
}

type Handler struct {
	syncer SyncRunner
}

func NewHandler(syncer SyncRunner) (*Handler, error) {
	if syncer == nil {
		return nil, errors.New("Foundry sync runner is required")
	}
	return &Handler{syncer: syncer}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != SyncPath {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	report, err := h.syncer.SyncAll(r.Context())
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, foundrysync.ErrSyncInProgress) {
			status = http.StatusConflict
		}
		writeJSONError(w, status, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(report)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
