package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	RegisterPath           = "/api/internal/discord/register"
	ActorsPath             = "/api/internal/discord/actors"
	CharactersPath         = "/api/internal/discord/characters"
	PreviewPath            = "/api/internal/discord/link/preview"
	LinkPath               = "/api/internal/discord/link"
	UnlinkPath             = "/api/internal/discord/unlink"
	GrantRolePath          = "/api/internal/discord/roles/grant"
	MapRolePath            = "/api/internal/discord/roles/map"
	DraftCharacterPath     = "/api/internal/discord/characters/draft"
	CharacterCreatePath    = "/api/internal/discord/characters/create"
	CharacterSearchPath    = "/api/internal/discord/characters/search"
	CharacterDetailPath    = "/api/internal/discord/characters/detail"
	MyCharactersPath       = "/api/internal/discord/characters/mine"
	ConnectActorsPath      = "/api/internal/discord/characters/connect/actors"
	ConnectActorPath       = "/api/internal/discord/characters/connect"
	UndoConnectPath        = "/api/internal/discord/characters/connect/undo"
	FoundrySyncPath        = "/api/internal/discord/foundry-sync"
	IntegrationStatusPath  = "/api/internal/discord/integration-status"
	HelpStatusPath         = "/api/internal/discord/help-status"
)

type bot struct {
	apiURL, apiToken, guildID string
	foundryPublicBaseURL      string
	campaignRoleID            string
	adminChannelID            string
	healthInterval            time.Duration
	client                    *http.Client
	syncClient                *http.Client
	session                   *discordgo.Session
	mu                        sync.Mutex
	reportMu                  sync.Mutex
	pending                   map[string]pendingLink
	pendingUnlinks            map[string]pendingUnlink
	flows                     map[string]*selectionFlow
	reportedErrors            map[string]time.Time
	lastHealthState           string
}
type pendingLink struct {
	actorUUID, characterID, discordID, guildID, ownerDiscordID string
	expiresAt                                                  time.Time
}
type pendingUnlink struct {
	actorUUID, actorName, discordID, guildID, ownerDiscordID string
	expiresAt                                                time.Time
}
type selectionFlow struct {
	mode, stage, discordID, guildID, actorUUID, ownerDiscordID string
	actors                                                     []actor
	characters                                                 []character
	page                                                       int
	expiresAt                                                  time.Time
}
type identity struct {
	GuildID        string   `json:"guild_id"`
	DiscordUserID  string   `json:"discord_user_id"`
	DiscordRoleIDs []string `json:"discord_role_ids"`
}
type registerPayload struct {
	identity
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}
type linkPayload struct {
	identity
	ActorUUID          string `json:"actor_uuid"`
	CharacterID        string `json:"character_id"`
	OwnerDiscordUserID string `json:"owner_discord_user_id,omitempty"`
}
type actor struct {
	ActorUUID string `json:"actor_uuid"`
	Name      string `json:"name"`
}
type character struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	OwnerName string `json:"owner_name,omitempty"`
}

type ownedCharacter struct {
	ID          string  `json:"id"`
	ActorUUID   string  `json:"actor_uuid"`
	Name        string  `json:"name"`
	PortraitURL *string `json:"portrait_url"`
	Level       int     `json:"level"`
	XP          int     `json:"xp"`
	Ancestry    string  `json:"ancestry"`
	Class       string  `json:"class"`
	Status      string  `json:"status"`
}

type myCharactersResult struct {
	Characters      []ownedCharacter `json:"characters"`
	ActiveSlots     int               `json:"active_slots"`
	MaxActiveSlots  int               `json:"max_active_slots"`
	RemainingSlots  int               `json:"remaining_slots"`
}

type connectResult struct {
	ActionID  string `json:"action_id"`
	ActorUUID string `json:"actor_uuid"`
	OwnerID   string `json:"owner_user_id"`
	Character struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"character"`
}
type linkPreview struct {
	ActorUUID     string `json:"actor_uuid"`
	ActorName     string `json:"actor_name"`
	CharacterID   string `json:"character_id"`
	CharacterName string `json:"character_name"`
}

type characterDetails struct {
	Identity struct {
		Name  string  `json:"name"`
		UUID  string  `json:"uuid"`
		Image *string `json:"image"`
	} `json:"identity"`
	Progression struct {
		Level      *int `json:"level"`
		Experience struct {
			Value *int `json:"value"`
			Max   *int `json:"max"`
		} `json:"experience"`
	} `json:"progression"`
	Build struct {
		Ancestry           *characterReference      `json:"ancestry"`
		Heritage           *characterReference      `json:"heritage"`
		Background         *characterReference      `json:"background"`
		Class              *characterReference      `json:"class"`
		KeyAbility         *string                  `json:"keyAbility"`
		AncestryFeatures   []classFeatureReference  `json:"ancestryFeatures"`
		ClassProficiencies *classProficiencyDetails `json:"classProficiencies"`
	} `json:"build"`
	Attributes struct {
		HP *struct {
			Current *int `json:"current"`
			Temp    *int `json:"temp"`
			Max     *int `json:"max"`
		} `json:"hp"`
		Dying *characterDying `json:"dying"`
	} `json:"attributes"`
	Statistics struct {
		AbilitiesDerived bool `json:"abilitiesDerived"`
		Abilities        map[string]struct {
			Score    *int `json:"score"`
			Modifier *int `json:"modifier"`
			Derived  bool `json:"derived"`
		} `json:"abilities"`
		ReconstructedAbilities map[string]struct {
			Score    *int `json:"score"`
			Modifier *int `json:"modifier"`
			Derived  bool `json:"derived"`
		} `json:"reconstructedAbilities"`
	} `json:"statistics"`
	Resources struct {
		HeroPoints *struct {
			Value *int `json:"value"`
			Max   *int `json:"max"`
		} `json:"heroPoints"`
	} `json:"resources"`
	Skills struct {
		Standard map[string]characterSkill `json:"standard"`
		Lore     map[string]characterSkill `json:"lore"`
	} `json:"skills"`
	Items []characterItem `json:"items"`
}
type characterDying struct {
	Current *int `json:"current"`
	Max     *int `json:"max"`
}

type characterItem struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Type                string             `json:"type"`
	Category            string             `json:"category"`
	SystemCategory      *string            `json:"systemCategory"`
	ActionType          *string            `json:"actionType"`
	Quantity            *int               `json:"quantity"`
	Level               *int               `json:"level"`
	Equipped            *characterEquipped `json:"equipped"`
	ContainerID         *string            `json:"containerId"`
	Usage               *string            `json:"usage"`
	SpellcastingEntryID *string            `json:"spellcastingEntryId"`
	Tradition           *string            `json:"tradition"`
	SpellList           *string            `json:"spellList"`
	Prepared            *bool              `json:"prepared"`
	PreparedRanks       []int              `json:"preparedRanks"`
	SpellcastingType    *string            `json:"spellcastingType"`
	Skill               *string            `json:"skill"`
	Cantrip             bool               `json:"cantrip"`
}

type characterEquipped struct {
	CarryType *string `json:"carryType"`
	HandsHeld *int    `json:"handsHeld"`
	InSlot    *bool   `json:"inSlot"`
	Invested  *bool   `json:"invested"`
}
type characterReference struct {
	Name string `json:"name"`
}

type classProficiencyDetails struct {
	Features []classFeatureReference `json:"features"`
}
type classFeatureReference struct {
	Name  string `json:"name"`
	Level *int   `json:"level"`
}
type characterSkill struct {
	Rank *int `json:"rank"`
}
type healthComponent struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}
type syncReport struct {
	Discovered       int `json:"discovered"`
	CharactersSynced int `json:"charactersSynced"`
	SnapshotsOnly    int `json:"snapshotsOnly"`
	Failed           int `json:"failed"`
}

type integrationStatus struct {
	Database healthComponent `json:"database"`
	Relay    healthComponent `json:"relay"`
	Foundry  healthComponent `json:"foundry"`
}

func main() {
	token, appID := requiredEnv("DISCORD_BOT_TOKEN"), requiredEnv("DISCORD_APPLICATION_ID")
	guildID, apiToken := requiredEnv("DISCORD_GUILD_ID"), requiredEnv("DISCORD_BOT_API_TOKEN")
	adminChannelID := strings.TrimSpace(os.Getenv("DISCORD_ADMIN_CHANNEL_ID"))
	campaignRoleID := strings.TrimSpace(os.Getenv("DISCORD_CAMPAIGN_ROLE_ID"))
	foundryPublicBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("FOUNDRY_PUBLIC_BASE_URL")), "/")
	healthInterval := 60 * time.Second
	if raw := strings.TrimSpace(os.Getenv("DISCORD_HEALTH_INTERVAL_SECONDS")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < 15 || seconds > 3600 {
			log.Fatal("DISCORD_HEALTH_INTERVAL_SECONDS must be between 15 and 3600")
		}
		healthInterval = time.Duration(seconds) * time.Second
	}
	b := &bot{apiURL: strings.TrimRight(requiredEnv("BACKEND_INTERNAL_URL"), "/"), apiToken: apiToken,
		guildID: guildID, foundryPublicBaseURL: foundryPublicBaseURL, campaignRoleID: campaignRoleID, adminChannelID: adminChannelID, healthInterval: healthInterval,
		client: &http.Client{Timeout: 12 * time.Second}, syncClient: &http.Client{Timeout: 10 * time.Minute}, pending: make(map[string]pendingLink), pendingUnlinks: make(map[string]pendingUnlink),
		flows: make(map[string]*selectionFlow), reportedErrors: make(map[string]time.Time)}
	if len(apiToken) < 32 {
		log.Fatal("DISCORD_BOT_API_TOKEN must contain at least 32 characters")
	}
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}
	s.Identify.Intents = discordgo.IntentsGuilds
	b.session = s
	s.AddHandler(b.handle)
	if err := s.Open(); err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := registerCommands(s, appID, guildID); err != nil {
		log.Fatal(err)
	}
	log.Printf("FoundryMarches bot connected; guild=%s", guildID)
	var monitorDone chan struct{}
	if adminChannelID != "" {
		monitorDone = make(chan struct{})
		go func() {
			defer close(monitorDone)
			b.monitorIntegrations(ctx, s)
		}()
	} else {
		log.Printf("DISCORD_ADMIN_CHANNEL_ID is not set; Discord health alerts are disabled")
	}
	<-ctx.Done()
	log.Printf("Discord bot shutdown requested")
	if monitorDone != nil {
		select {
		case <-monitorDone:
		case <-time.After(15 * time.Second):
			log.Printf("Timed out waiting for integration monitor to stop")
		}
	}
}

func registerCommands(s *discordgo.Session, appID, guildID string) error {
	commands := []*discordgo.ApplicationCommand{
		{Name: "help", Description: "Показать справку по боту"},
		{Name: "register", Description: "Зарегистрироваться в FoundryMarches"},
		{Name: "foundry-actors", Description: "Показать несвязанных персонажей Foundry"},
		{Name: "characters", Description: "Показать список персонажей приложения"},
		{Name: "character-unlink", Description: "Отвязать Foundry Actor; можно явно переназначить владельца", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя или часть имени персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}, {Name: "owner", Description: "Новый владелец (необязательно; иначе текущий сохраняется)", Type: discordgo.ApplicationCommandOptionUser, Required: false}}},
		{Name: "character-link", Description: "Связать Foundry Actor с персонажем и назначить владельца", Options: []*discordgo.ApplicationCommandOption{{Name: "owner", Description: "Владелец персонажа (обязательно)", Type: discordgo.ApplicationCommandOptionUser, Required: true}}},
		{Name: "character-create", Description: "Создать черновик персонажа для себя или игрока", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}, {Name: "owner", Description: "Владелец персонажа (для мастера)", Type: discordgo.ApplicationCommandOptionUser, Required: false}}},
		{Name: "character-draft", Description: "Создать черновик персонажа из выбранного Foundry Actor"},
		{Name: "character", Description: "Найти персонажа по имени (GM видит всех)", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя или часть имени персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}}},
		{Name: "my-character", Description: "Открыть карточку своего персонажа"},
		{Name: "my-characters", Description: "Показать своих персонажей и свободные слоты"},
		{Name: "character-connect", Description: "Самостоятельно привязать персонажа из Foundry"},
		{Name: "foundry-sync", Description: "Обновить снимки всех Actor из Foundry"},
		{Name: "character-inventory", Description: "Показать инвентарь персонажа", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя или часть имени персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}}},
		{Name: "character-spells", Description: "Показать заклинания персонажа", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя или часть имени персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}}},
		{Name: "integration-status", Description: "Проверить доступность API, Relay и Foundry"},
		{Name: "role-grant", Description: "Назначить роль зарегистрированному участнику",
			Options: []*discordgo.ApplicationCommandOption{
				{Name: "discord_user_id", Description: "Discord ID участника", Type: discordgo.ApplicationCommandOptionString, Required: true},
				{Name: "role_key", Description: "Роль приложения", Type: discordgo.ApplicationCommandOptionString, Required: true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Игрок", Value: "player"}, {Name: "Мастер", Value: "gm"}}}}},
		{Name: "role-map", Description: "Сопоставить Discord-роль с ролью приложения",
			Options: []*discordgo.ApplicationCommandOption{
				{Name: "discord_role_id", Description: "ID Discord-роли", Type: discordgo.ApplicationCommandOptionString, Required: true},
				{Name: "role_key", Description: "Роль приложения", Type: discordgo.ApplicationCommandOptionString, Required: true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{{Name: "Игрок", Value: "player"}, {Name: "Мастер", Value: "gm"}}}}},
	}
	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
			return fmt.Errorf("register %s: %w", cmd.Name, err)
		}
	}
	return nil
}

func (b *bot) handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID != b.guildID {
		b.reply(s, i, "Бот работает только на настроенном сервере.", nil)
		return
	}
	if i.Type == discordgo.InteractionMessageComponent {
		b.handleComponent(s, i)
		return
	}
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		return
	}
	u := interactionUser(i)
	if u == nil {
		b.edit(s, i, "Не удалось определить пользователя.", nil)
		return
	}
	id := b.identity(i, u)
	switch i.ApplicationCommandData().Name {
	case "help":
		var status helpStatus
		if err := b.post(HelpStatusPath, id, &status); err != nil {
			b.edit(s, i, "Не удалось определить состояние аккаунта: "+err.Error(), nil)
			return
		}
		b.editEmbed(s, i, helpEmbed(status))
	case "register":
		var out struct {
			UserID string `json:"user_id"`
		}
		err := b.post(RegisterPath, registerPayload{identity: id, Username: u.Username, AvatarURL: u.AvatarURL("256")}, &out)
		if err != nil {
			b.edit(s, i, "Регистрация не выполнена: "+err.Error(), nil)
			return
		}
		if b.campaignRoleID != "" {
			if err := s.GuildMemberRoleAdd(i.GuildID, u.ID, b.campaignRoleID); err != nil {
				log.Printf("Discord campaign role assignment failed for user %s: %v", u.ID, err)
				b.edit(s, i, "Аккаунт зарегистрирован, но роль доступа к кампании выдать не удалось. Сообщи мастеру.", nil)
				return
			}
		}
		if b.campaignRoleID == "" {
			log.Printf("DISCORD_CAMPAIGN_ROLE_ID is not configured; campaign channel role was not assigned")
			b.edit(s, i, "Регистрация завершена. Автоматическая выдача роли доступа к каналам пока не настроена; сообщи мастеру.", nil)
		} else {
			b.edit(s, i, "Добро пожаловать в FoundryMarches! Регистрация завершена, доступ к каналам кампании выдан.", nil)
		}
	case "foundry-actors":
		var out struct {
			Actors []actor `json:"actors"`
		}
		if err := b.post(ActorsPath, id, &out); err != nil {
			b.edit(s, i, "Ошибка: "+err.Error(), nil)
			return
		}
		lines := make([]string, 0, len(out.Actors))
		for _, a := range out.Actors {
			lines = append(lines, truncateLabel(a.Name))
		}
		if len(lines) == 0 {
			b.edit(s, i, "✨ **Все персонажи Foundry уже связаны.**\n\nЕсли ожидаемый персонаж отсутствует, запусти синхронизацию мира.", nil)
		} else {
			b.editEmbed(s, i, directoryEmbed("Персонажи Foundry", "Актёры, которые ещё не связаны с персонажами приложения. Выбери нужного в команде `/character-link`.", lines, 50))
		}
	case "characters":
		var out struct {
			Characters []character `json:"characters"`
		}
		if err := b.post(CharactersPath, id, &out); err != nil {
			b.edit(s, i, "Ошибка: "+err.Error(), nil)
			return
		}
		lines := make([]string, 0, len(out.Characters))
		for _, item := range out.Characters {
			line := fmt.Sprintf("%s · %s", truncateLabel(item.Name), characterStatusLabel(item.Status))
			if item.OwnerName != "" {
				line += " · владелец: " + truncateLabel(item.OwnerName)
			} else {
				line += " · владелец не назначен"
			}
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			b.edit(s, i, "📝 **Пока нет персонажей приложения.**\n\nСоздай первого командой `/character-create`.", nil)
		} else {
			b.editEmbed(s, i, directoryEmbed("Персонажи приложения", "Черновики и зарегистрированные персонажи кампании.", lines, 40))
		}
	case "character-connect":
		if b.adminChannelID == "" {
			b.edit(s, i, "Самостоятельная привязка пока недоступна: мастер-чат не настроен. Сообщи мастеру о необходимости задать DISCORD_ADMIN_CHANNEL_ID.", nil)
			return
		}
		b.startSelection(s, i, id, u, "connect", "")
	case "character-link":
		o := commandOptions(i.ApplicationCommandData().Options)
		b.startSelection(s, i, id, u, "link", o["owner"])
	case "character-unlink":
		o := commandOptions(i.ApplicationCommandData().Options)
		b.startCharacterLookup(s, i, id, u, o["name"], "unlink:"+o["owner"])
	case "character-create":
		o := commandOptions(i.ApplicationCommandData().Options)
		var created character
		payload := struct {
			identity
			Name               string `json:"name"`
			OwnerDiscordUserID string `json:"owner_discord_user_id,omitempty"`
		}{id, o["name"], o["owner"]}
		if err := b.post(CharacterCreatePath, payload, &created); err != nil {
			b.edit(s, i, "Не удалось создать персонажа: "+err.Error(), nil)
			return
		}
		ownerText := "ты"
		if o["owner"] != "" {
			ownerText = "<@" + o["owner"] + ">"
		}
		b.edit(s, i, fmt.Sprintf("**Черновик персонажа создан**\n\nИмя: **%s**\nВладелец: %s\nСтатус: **Черновик**\n\nМастер сможет связать его с Foundry Actor командой `/character-link`.", created.Name, ownerText), nil)
	case "character-draft":
		b.startSelection(s, i, id, u, "draft", "")
	case "character":
		o := commandOptions(i.ApplicationCommandData().Options)
		b.startCharacterLookup(s, i, id, u, o["name"], "view")
	case "my-character":
		b.startMyCharacterLookup(s, i, id, u)
	case "my-characters":
		var result myCharactersResult
		if err := b.post(MyCharactersPath, id, &result); err != nil {
			b.edit(s, i, "Не удалось получить список персонажей: "+err.Error(), nil)
			return
		}
		b.editEmbeds(s, i, myCharactersEmbeds(result, b.foundryPublicBaseURL))
	case "foundry-sync":
		var report syncReport
		if err := b.post(FoundrySyncPath, id, &report); err != nil {
			b.edit(s, i, "Синхронизация Foundry не выполнена: "+err.Error(), nil)
			return
		}
		b.edit(s, i, fmt.Sprintf("**Синхронизация Foundry завершена**\n\nОбнаружено Actor: **%d**\nОбновлено связанных персонажей: **%d**\nСнимков без привязки: **%d**\nОшибок отдельных Actor: **%d**", report.Discovered, report.CharactersSynced, report.SnapshotsOnly, report.Failed), nil)
	case "character-inventory":
		o := commandOptions(i.ApplicationCommandData().Options)
		b.startCharacterLookup(s, i, id, u, o["name"], "inventory")
	case "character-spells":
		o := commandOptions(i.ApplicationCommandData().Options)
		b.startCharacterLookup(s, i, id, u, o["name"], "spells")
	case "integration-status":
		b.showIntegrationStatus(s, i)
	case "role-grant":
		o := commandOptions(i.ApplicationCommandData().Options)
		payload := struct {
			identity
			Target string `json:"target_discord_user_id"`
			Role   string `json:"role_key"`
		}{id, o["discord_user_id"], o["role_key"]}
		if err := b.post(GrantRolePath, payload, nil); err != nil {
			b.edit(s, i, "Не удалось назначить роль: "+err.Error(), nil)
		} else {
			b.edit(s, i, "✅ **Роль приложения назначена.**", nil)
		}
	case "role-map":
		o := commandOptions(i.ApplicationCommandData().Options)
		payload := struct {
			identity
			RoleID string `json:"discord_role_id"`
			Role   string `json:"role_key"`
		}{id, o["discord_role_id"], o["role_key"]}
		if err := b.post(MapRolePath, payload, nil); err != nil {
			b.edit(s, i, "Не удалось сопоставить роль: "+err.Error(), nil)
		} else {
			b.edit(s, i, "✅ **Сопоставление Discord-роли сохранено.**", nil)
		}
	default:
		b.edit(s, i, "Неизвестная команда.", nil)
	}
}

func (b *bot) startSelection(s *discordgo.Session, i *discordgo.InteractionCreate, id identity, u *discordgo.User, mode, ownerDiscordID string) {
	var actorResult struct {
		Actors []actor `json:"actors"`
	}
	actorsPath := ActorsPath
	if mode == "connect" { actorsPath = ConnectActorsPath }
	if err := b.post(actorsPath, id, &actorResult); err != nil {
		b.edit(s, i, "Не удалось получить список Foundry Actor: "+err.Error(), nil)
		return
	}
	if len(actorResult.Actors) == 0 {
		b.edit(s, i, "Нет несвязанных Actor типа character. Сначала синхронизируйте Foundry.", nil)
		return
	}
	flow := &selectionFlow{mode: mode, stage: "actor", discordID: u.ID, guildID: i.GuildID,
		actors: actorResult.Actors, ownerDiscordID: ownerDiscordID, expiresAt: time.Now().Add(10 * time.Minute)}
	if mode == "link" {
		var characterResult struct {
			Characters []character `json:"characters"`
		}
		if err := b.post(CharactersPath, id, &characterResult); err != nil {
			b.edit(s, i, "Не удалось получить список персонажей: "+err.Error(), nil)
			return
		}
		if len(characterResult.Characters) == 0 {
			b.edit(s, i, "Персонажей приложения пока нет. Создай персонажа командой /character-create или черновик из Foundry через /character-draft.", nil)
			return
		}
		flow.characters = characterResult.Characters
	}
	key, err := randomKey()
	if err != nil {
		b.edit(s, i, "Не удалось подготовить выбор.", nil)
		return
	}
	b.mu.Lock()
	b.pruneFlowsLocked()
	b.flows[key] = flow
	b.mu.Unlock()
	components := b.actorSelectionComponents(key, flow)
	label := "Выбери Foundry Actor, из которого создать черновик:"
	if mode == "link" {
		label = "Выбери Foundry Actor для привязки к персонажу. Владелец: <@" + ownerDiscordID + ">"
	} else if mode == "connect" {
		label = "Выбери своего персонажа Foundry. Он сразу займёт один из двух активных слотов; мастер сможет отменить привязку в мастер-чате."
	}
	b.edit(s, i, label, components)
}

func (b *bot) startCharacterLookup(s *discordgo.Session, i *discordgo.InteractionCreate, id identity, u *discordgo.User, query, mode string) {
	var result struct{ Actors []actor }
	payload := map[string]any{"guild_id": id.GuildID, "discord_user_id": id.DiscordUserID, "discord_role_ids": id.DiscordRoleIDs, "query": query}
	if err := b.post(CharacterSearchPath, payload, &result); err != nil {
		b.edit(s, i, "Не удалось найти персонажа: "+err.Error(), nil)
		return
	}
	if len(result.Actors) == 0 {
		b.edit(s, i, "Персонажи с таким именем не найдены в последней синхронизации Foundry.", nil)
		return
	}
	if len(result.Actors) == 1 {
		b.showLookupMatch(s, i, id, u, result.Actors[0], mode)
		return
	}
	flow := &selectionFlow{mode: mode, stage: "actor", discordID: u.ID, guildID: i.GuildID, actors: result.Actors, expiresAt: time.Now().Add(10 * time.Minute)}
	key, err := randomKey()
	if err != nil {
		b.edit(s, i, "Не удалось подготовить выбор персонажа.", nil)
		return
	}
	b.mu.Lock()
	b.pruneFlowsLocked()
	b.flows[key] = flow
	b.mu.Unlock()
	b.edit(s, i, fmt.Sprintf("Найдено персонажей: %d. Выбери нужного:", len(result.Actors)), b.actorSelectionComponents(key, flow))
}

func (b *bot) startMyCharacterLookup(s *discordgo.Session, i *discordgo.InteractionCreate, id identity, u *discordgo.User) {
	var result struct{ Actors []actor }
	if err := b.post(MyCharactersPath, id, &result); err != nil {
		b.edit(s, i, "Не удалось получить твоих персонажей: "+err.Error(), nil)
		return
	}
	if len(result.Actors) == 0 {
		b.edit(s, i, "У тебя пока нет привязанного и подтверждённого персонажа. Попроси мастера связать персонажа с твоим Discord-аккаунтом через /character-link.", nil)
		return
	}
	if len(result.Actors) == 1 {
		b.showLookupMatch(s, i, id, u, result.Actors[0], "view")
		return
	}
	flow := &selectionFlow{mode: "view", stage: "actor", discordID: u.ID, guildID: i.GuildID, actors: result.Actors, expiresAt: time.Now().Add(10 * time.Minute)}
	key, err := randomKey()
	if err != nil {
		b.edit(s, i, "Не удалось подготовить выбор персонажа.", nil)
		return
	}
	b.mu.Lock()
	b.pruneFlowsLocked()
	b.flows[key] = flow
	b.mu.Unlock()
	b.edit(s, i, "Выбери своего персонажа:", b.actorSelectionComponents(key, flow))
}

func (b *bot) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	if strings.HasPrefix(data.CustomID, "fm-link:") || strings.HasPrefix(data.CustomID, "fm-unlink-confirm:") || strings.HasPrefix(data.CustomID, "fm-connect-undo:") {
		b.handleButton(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "fm-detail:") {
		b.handleCharacterDetailButton(s, i)
		return
	}
	parts := strings.Split(data.CustomID, ":")
	if len(parts) < 2 {
		return
	}
	u := interactionUser(i)
	if u == nil {
		b.reply(s, i, "Не удалось определить пользователя.", nil)
		return
	}
	key := parts[1]
	b.mu.Lock()
	flow, ok := b.flows[key]
	if !ok || time.Now().After(flow.expiresAt) || flow.discordID != u.ID || flow.guildID != i.GuildID {
		if ok {
			delete(b.flows, key)
		}
		b.mu.Unlock()
		b.reply(s, i, "Выбор недействителен или истёк. Запусти команду заново.", nil)
		return
	}
	if flow.stage == "actor" && !strings.HasPrefix(data.CustomID, "fm-actor:") && !strings.HasPrefix(data.CustomID, "fm-page:") {
		b.mu.Unlock()
		b.reply(s, i, "Этот выбор уже недействителен. Запусти команду заново.", nil)
		return
	}
	if flow.stage == "character" && !strings.HasPrefix(data.CustomID, "fm-character:") && !strings.HasPrefix(data.CustomID, "fm-page:") {
		b.mu.Unlock()
		b.reply(s, i, "Этот выбор уже недействителен. Запусти команду заново.", nil)
		return
	}
	if strings.HasPrefix(data.CustomID, "fm-page:") {
		if len(parts) != 4 || parts[2] != flow.stage {
			b.mu.Unlock()
			return
		}
		page, err := strconv.Atoi(parts[3])
		maxPage := (len(flow.actors) - 1) / 25
		if flow.stage == "character" {
			maxPage = (len(flow.characters) - 1) / 25
		}
		if err != nil || page < 0 || page > maxPage {
			b.mu.Unlock()
			return
		}
		flow.page = page
		components := b.selectionComponents(key, flow)
		b.mu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Components: components}})
		return
	}
	if len(data.Values) != 1 {
		b.mu.Unlock()
		return
	}
	selected := data.Values[0]
	if flow.stage == "actor" {
		var chosen *actor
		for n := range flow.actors {
			if flow.actors[n].ActorUUID == selected {
				chosen = &flow.actors[n]
				break
			}
		}
		if chosen == nil {
			b.mu.Unlock()
			b.reply(s, i, "Выбранный Actor отсутствует в списке. Запусти команду заново.", nil)
			return
		}
		flow.actorUUID = chosen.ActorUUID
		if strings.HasPrefix(flow.mode, "unlink:") {
			ownerDiscordID := strings.TrimPrefix(flow.mode, "unlink:")
			delete(b.flows, key)
			pendingKey, err := randomKey()
			if err != nil {
				b.mu.Unlock()
				b.reply(s, i, "Не удалось подготовить подтверждение отвязки.", nil)
				return
			}
			b.pendingUnlinks[pendingKey] = pendingUnlink{
				actorUUID: chosen.ActorUUID, actorName: chosen.Name, discordID: u.ID,
				guildID: i.GuildID, ownerDiscordID: ownerDiscordID, expiresAt: time.Now().Add(10 * time.Minute),
			}
			b.mu.Unlock()
			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
				return
			}
			message := "Подтверди отвязку Foundry Actor **" + chosen.Name + "**. Персонаж приложения и история сохранятся."
			if ownerDiscordID != "" {
				message += "\nНовый владелец: <@" + ownerDiscordID + ">."
			} else {
				message += "\nТекущий владелец останется без изменений."
			}
			b.edit(s, i, message, unlinkConfirmationComponents(pendingKey))
			return
		}
		if flow.mode == "connect" {
			delete(b.flows, key)
			b.mu.Unlock()
			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil { return }
			var result connectResult
			payload := struct { identity; ActorUUID string `json:"actor_uuid"` }{b.identity(i, u), chosen.ActorUUID}
			if err := b.post(ConnectActorPath, payload, &result); err != nil {
				b.edit(s, i, "Не удалось привязать персонажа: "+err.Error(), nil)
				return
			}
			auditSent := b.sendConnectAudit(s, u, chosen.Name, result)
			message := fmt.Sprintf("✅ **Персонаж привязан!**\n\n**%s** теперь связан с твоим аккаунтом и сразу занимает один из двух активных слотов.\n\nПосмотреть всех своих персонажей и свободные слоты: `/my-characters`. Карточка персонажа: `/my-character`.", result.Character.Name)
			if !auditSent { message += "\n\n⚠️ Не удалось отправить уведомление в мастер-чат. Сообщи мастеру об этой привязке." }
			b.edit(s, i, message, nil)
			return
		}
		if flow.mode == "view" || flow.mode == "inventory" || flow.mode == "spells" {
			mode := flow.mode
			delete(b.flows, key)
			b.mu.Unlock()
			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
				return
			}
			var details characterDetails
			err := b.post(CharacterDetailPath, struct {
				identity
				ActorUUID string `json:"actor_uuid"`
			}{b.identity(i, u), chosen.ActorUUID}, &details)
			if err != nil {
				b.edit(s, i, "Не удалось загрузить данные персонажа: "+err.Error(), nil)
				return
			}
			details.Identity.Image = characterPortraitURL(details.Identity.Image, b.foundryPublicBaseURL)
			switch mode {
			case "inventory":
				b.editEmbeds(s, i, characterInventoryEmbeds(details))
			case "spells":
				b.editEmbeds(s, i, characterSpellsEmbeds(details))
			default:
				b.showCharacterCard(s, i, u, chosen.ActorUUID, details)
			}
			return
		}
		if flow.mode == "draft" {
			delete(b.flows, key)
			b.mu.Unlock()
			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
				return
			}
			var created character
			err := b.post(DraftCharacterPath, struct {
				identity
				ActorUUID string `json:"actor_uuid"`
			}{b.identity(i, u), chosen.ActorUUID}, &created)
			if err != nil {
				b.edit(s, i, "Не удалось создать черновик: "+err.Error(), nil)
				return
			}
			b.edit(s, i, fmt.Sprintf("**Черновик создан**\n\nПерсонаж: **%s**\nВладелец: пока не назначен\n\nПри привязке через `/character-link` мастер должен указать владельца через Discord-упоминание.", created.Name), nil)
			return
		}
		flow.stage, flow.page = "character", 0
		components := b.characterSelectionComponents(key, flow)
		b.mu.Unlock()
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Content: "Теперь выбери персонажа приложения:", Components: components}})
		return
	}
	if flow.stage == "character" {
		var chosen *character
		for n := range flow.characters {
			if flow.characters[n].ID == selected {
				chosen = &flow.characters[n]
				break
			}
		}
		if chosen == nil {
			b.mu.Unlock()
			b.reply(s, i, "Выбранный персонаж отсутствует в списке. Запусти команду заново.", nil)
			return
		}
		actorUUID := flow.actorUUID
		delete(b.flows, key)
		b.mu.Unlock()
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
			return
		}
		payload := linkPayload{identity: b.identity(i, u), ActorUUID: actorUUID, CharacterID: chosen.ID, OwnerDiscordUserID: flow.ownerDiscordID}
		var preview linkPreview
		if err := b.post(PreviewPath, payload, &preview); err != nil {
			b.edit(s, i, "Нельзя подготовить привязку: "+err.Error(), nil)
			return
		}
		confirmKey, err := randomKey()
		if err != nil {
			b.edit(s, i, "Не удалось создать подтверждение.", nil)
			return
		}
		b.mu.Lock()
		b.prunePendingLocked()
		b.pending[confirmKey] = pendingLink{actorUUID: payload.ActorUUID, characterID: payload.CharacterID, discordID: u.ID, guildID: i.GuildID, ownerDiscordID: flow.ownerDiscordID, expiresAt: time.Now().Add(10 * time.Minute)}
		b.mu.Unlock()
		components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "Подтвердить привязку", Style: discordgo.SuccessButton, CustomID: "fm-link:" + confirmKey}}}}
		ownerLabel := "владелец, уже указанный в приложении"
		if flow.ownerDiscordID != "" {
			ownerLabel = "<@" + flow.ownerDiscordID + ">"
		}
		b.edit(s, i, fmt.Sprintf("**Проверьте привязку персонажа**\n\n**Foundry Actor**\n%s\n\n**Персонаж приложения**\n%s\n\n**Владелец**\n%s\n\nПодтверждение закрепит указанного владельца и откроет ему доступ к карточке персонажа.", preview.ActorName, preview.CharacterName, ownerLabel), components)
	}
}

func (b *bot) showLookupMatch(s *discordgo.Session, i *discordgo.InteractionCreate, id identity, u *discordgo.User, selected actor, mode string) {
	if strings.HasPrefix(mode, "unlink:") {
		pendingKey, err := randomKey()
		if err != nil {
			b.edit(s, i, "Не удалось подготовить подтверждение отвязки.", nil)
			return
		}
		ownerDiscordID := strings.TrimPrefix(mode, "unlink:")
		b.mu.Lock()
		b.pruneFlowsLocked()
		b.pendingUnlinks[pendingKey] = pendingUnlink{actorUUID: selected.ActorUUID, actorName: selected.Name, discordID: u.ID, guildID: i.GuildID, ownerDiscordID: ownerDiscordID, expiresAt: time.Now().Add(10 * time.Minute)}
		b.mu.Unlock()
		message := "Подтверди отвязку Foundry Actor **" + selected.Name + "**. Персонаж приложения и история сохранятся."
		if ownerDiscordID != "" {
			message += "\nНовый владелец: <@" + ownerDiscordID + ">."
		} else {
			message += "\nТекущий владелец останется без изменений."
		}
		b.edit(s, i, message, unlinkConfirmationComponents(pendingKey))
		return
	}
	var details characterDetails
	payload := map[string]any{"guild_id": id.GuildID, "discord_user_id": id.DiscordUserID, "discord_role_ids": id.DiscordRoleIDs, "actor_uuid": selected.ActorUUID}
	if err := b.post(CharacterDetailPath, payload, &details); err != nil {
		b.edit(s, i, "Не удалось загрузить данные персонажа: "+err.Error(), nil)
		return
	}
	details.Identity.Image = characterPortraitURL(details.Identity.Image, b.foundryPublicBaseURL)
	switch mode {
	case "inventory":
		b.editEmbed(s, i, characterInventoryEmbed(details))
	case "spells":
		b.editEmbed(s, i, characterSpellsEmbed(details))
	default:
		b.showCharacterCard(s, i, u, selected.ActorUUID, details)
	}
}

func (b *bot) showCharacterCard(s *discordgo.Session, i *discordgo.InteractionCreate, u *discordgo.User, actorUUID string, details characterDetails) {
	key, err := randomKey()
	if err != nil {
		b.edit(s, i, "Не удалось подготовить карточку персонажа.", nil)
		return
	}
	flow := &selectionFlow{mode: "detail", stage: "details", discordID: u.ID, guildID: i.GuildID, actorUUID: actorUUID, expiresAt: time.Now().Add(10 * time.Minute)}
	b.mu.Lock()
	b.pruneFlowsLocked()
	b.flows[key] = flow
	b.mu.Unlock()
	b.editCharacterCard(s, i, characterEmbed(details), characterDetailComponents(key, "overview"))
}

func characterDetailComponents(key, activeSection string) []discordgo.MessageComponent {
	sections := []struct{ id, label string }{
		{"overview", "Карточка"}, {"abilities", "Способности"}, {"skills", "Навыки"},
		{"inventory", "Снаряжение"}, {"spells", "Заклинания"}, {"metrics", "Метрики кампании"},
	}
	buttons := make([]discordgo.MessageComponent, 0, len(sections))
	for _, section := range sections {
		style, disabled := discordgo.SecondaryButton, false
		if section.id == activeSection {
			style, disabled = discordgo.PrimaryButton, true
		}
		buttons = append(buttons, discordgo.Button{
			CustomID: "fm-detail:" + key + ":" + section.id,
			Label:    section.label, Style: style, Disabled: disabled,
		})
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: buttons[:3]},
		discordgo.ActionsRow{Components: buttons[3:]},
	}
}

func (b *bot) editCharacterCard(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	content := ""
	embeds := []*discordgo.MessageEmbed{embed}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Embeds: &embeds, Components: &components})
	if err != nil {
		log.Printf("Discord character card edit: %v", err)
	}
}

func (b *bot) handleCharacterDetailButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	parts := strings.Split(i.MessageComponentData().CustomID, ":")
	if len(parts) != 3 {
		return
	}
	u := interactionUser(i)
	if u == nil {
		b.reply(s, i, "Не удалось определить пользователя.", nil)
		return
	}
	key, section := parts[1], parts[2]
	b.mu.Lock()
	flow, ok := b.flows[key]
	valid := ok && flow.mode == "detail" && flow.stage == "details" && time.Now().Before(flow.expiresAt) && flow.discordID == u.ID && flow.guildID == i.GuildID
	actorUUID := ""
	if valid {
		actorUUID = flow.actorUUID
	}
	b.mu.Unlock()
	if !valid {
		b.reply(s, i, "Карточка устарела. Выполни команду поиска персонажа заново.", nil)
		return
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}
	id := b.identity(i, u)
	payload := map[string]any{"guild_id": id.GuildID, "discord_user_id": id.DiscordUserID, "discord_role_ids": id.DiscordRoleIDs, "actor_uuid": actorUUID}
	var details characterDetails
	if err := b.post(CharacterDetailPath, payload, &details); err != nil {
		b.edit(s, i, "Не удалось обновить данные персонажа: "+err.Error(), nil)
		return
	}
	details.Identity.Image = characterPortraitURL(details.Identity.Image, b.foundryPublicBaseURL)
	var embed *discordgo.MessageEmbed
	switch section {
	case "overview":
		embed = characterEmbed(details)
	case "abilities":
		embed = characterAbilitiesEmbed(details)
	case "skills":
		embed = characterSkillsEmbed(details)
	case "inventory":
		embed = characterInventoryEmbed(details)
	case "spells":
		embed = characterSpellsEmbed(details)
	case "metrics":
		embed = characterMetricsEmbed(details)
	default:
		b.edit(s, i, "Неизвестный раздел карточки.", nil)
		return
	}
	b.editCharacterCard(s, i, embed, characterDetailComponents(key, section))
}

func (b *bot) actorSelectionComponents(key string, flow *selectionFlow) []discordgo.MessageComponent {
	start, end := pageBounds(len(flow.actors), flow.page)
	minValues := 1
	options := make([]discordgo.SelectMenuOption, 0, end-start)
	for _, item := range flow.actors[start:end] {
		options = append(options, discordgo.SelectMenuOption{Label: truncateLabel(item.Name), Value: item.ActorUUID, Description: "Персонаж из Foundry"})
	}
	rows := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "fm-actor:" + key, Placeholder: "Выбери Foundry Actor", MinValues: &minValues, MaxValues: 1, Options: options}}}}
	if len(flow.actors) > 25 {
		rows = append(rows, pageButtons("fm-page:"+key+":"+flow.stage, flow.page, len(flow.actors)))
	}
	return rows
}

func (b *bot) characterSelectionComponents(key string, flow *selectionFlow) []discordgo.MessageComponent {
	start, end := pageBounds(len(flow.characters), flow.page)
	minValues := 1
	options := make([]discordgo.SelectMenuOption, 0, end-start)
	for _, item := range flow.characters[start:end] {
		options = append(options, discordgo.SelectMenuOption{Label: truncateLabel(item.Name + " [" + characterStatusLabel(item.Status) + "]"), Value: item.ID, Description: "Персонаж приложения"})
	}
	rows := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "fm-character:" + key, Placeholder: "Выбери персонажа приложения", MinValues: &minValues, MaxValues: 1, Options: options}}}}
	if len(flow.characters) > 25 {
		rows = append(rows, pageButtons("fm-page:"+key+":"+flow.stage, flow.page, len(flow.characters)))
	}
	return rows
}

func (b *bot) selectionComponents(key string, flow *selectionFlow) []discordgo.MessageComponent {
	if flow.stage == "character" {
		return b.characterSelectionComponents(key, flow)
	}
	return b.actorSelectionComponents(key, flow)
}

func pageButtons(prefix string, page, total int) discordgo.ActionsRow {
	buttons := make([]discordgo.MessageComponent, 0, 2)
	if page > 0 {
		buttons = append(buttons, discordgo.Button{Label: "Назад", Style: discordgo.SecondaryButton, CustomID: prefix + ":" + strconv.Itoa(page-1)})
	}
	if (page+1)*25 < total {
		buttons = append(buttons, discordgo.Button{Label: "Далее", Style: discordgo.SecondaryButton, CustomID: prefix + ":" + strconv.Itoa(page+1)})
	}
	return discordgo.ActionsRow{Components: buttons}
}

func pageBounds(total, page int) (int, int) {
	maxPage := (total - 1) / 25
	if page < 0 {
		page = 0
	}
	if page > maxPage {
		page = maxPage
	}
	start := page * 25
	end := start + 25
	if end > total {
		end = total
	}
	return start, end
}
func truncateLabel(value string) string {
	r := []rune(value)
	if len(r) > 100 {
		return string(r[:97]) + "..."
	}
	return value
}
func (b *bot) prunePendingLocked() {
	now := time.Now()
	for key, pending := range b.pending {
		if now.After(pending.expiresAt) {
			delete(b.pending, key)
		}
	}
}

func (b *bot) pruneFlowsLocked() {
	now := time.Now()
	for key, flow := range b.flows {
		if now.After(flow.expiresAt) {
			delete(b.flows, key)
		}
	}
	for key, pending := range b.pendingUnlinks {
		if now.After(pending.expiresAt) {
			delete(b.pendingUnlinks, key)
		}
	}
}

func (b *bot) handleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	if strings.HasPrefix(data.CustomID, "fm-connect-undo:") {
		b.handleConnectUndoButton(s, i)
		return
	}
	if strings.HasPrefix(data.CustomID, "fm-unlink-confirm:") {
		b.handleUnlinkButton(s, i)
		return
	}
	if !strings.HasPrefix(data.CustomID, "fm-link:") {
		return
	}
	u := interactionUser(i)
	if u == nil {
		b.reply(s, i, "Не удалось определить пользователя.", nil)
		return
	}
	key := strings.TrimPrefix(data.CustomID, "fm-link:")
	b.mu.Lock()
	p, ok := b.pending[key]
	valid := ok && time.Now().Before(p.expiresAt) && p.discordID == u.ID && p.guildID == i.GuildID
	if valid {
		delete(b.pending, key)
	}
	b.mu.Unlock()
	if !valid {
		b.reply(s, i, "Подтверждение недействительно, истекло или принадлежит другому участнику. Повторите /character-link.", nil)
		return
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		log.Printf("Discord deferred response: %v", err)
		return
	}
	err := b.post(LinkPath, linkPayload{identity: b.identity(i, u), ActorUUID: p.actorUUID, CharacterID: p.characterID, OwnerDiscordUserID: p.ownerDiscordID}, nil)
	if err != nil {
		b.edit(s, i, "Привязка не выполнена: "+err.Error(), []discordgo.MessageComponent{})
	} else {
		b.edit(s, i, "Привязка Foundry выполнена. Владелец персонажа закреплён за указанным Discord-аккаунтом.", []discordgo.MessageComponent{})
	}
}

func unlinkConfirmationComponents(key string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{CustomID: "fm-unlink-confirm:" + key, Label: "Подтвердить отвязку", Style: discordgo.DangerButton},
		}},
	}
}

func (b *bot) sendConnectAudit(s *discordgo.Session, player *discordgo.User, actorName string, result connectResult) bool {
	if b.adminChannelID == "" {
		log.Printf("DISCORD_ADMIN_CHANNEL_ID is not set; self-connect undo button was not delivered for action %s", result.ActionID)
		return false
	}
	embed := &discordgo.MessageEmbed{
		Title: "Новая привязка персонажа",
		Description: fmt.Sprintf("Игрок <@%s> самостоятельно привязал персонажа из Foundry. Если мастер против этой привязки, её можно отменить кнопкой ниже.", player.ID),
		Color: 0xFEE75C,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Персонаж", Value: result.Character.Name, Inline: true},
			{Name: "Foundry Actor", Value: actorName, Inline: true},
			{Name: "Статус", Value: "Активен · занимает слот", Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Действие сохраняется в журнале; отмена не удаляет историю"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{CustomID: "fm-connect-undo:" + result.ActionID, Label: "Отменить привязку", Style: discordgo.DangerButton},
	}}}
	if _, err := s.ChannelMessageSendComplex(b.adminChannelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}, Components: components}); err != nil {
		log.Printf("Discord self-connect audit notification failed for action %s: %v", result.ActionID, err)
		return false
	}
	return true
}

func (b *bot) handleConnectUndoButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	u := interactionUser(i)
	if u == nil {
		b.reply(s, i, "Не удалось определить пользователя.", nil)
		return
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil { return }
	actionID := strings.TrimPrefix(i.MessageComponentData().CustomID, "fm-connect-undo:")
	var result struct {
		Status string `json:"status"`
		ActorUUID string `json:"actor_uuid"`
		CharacterName string `json:"character_name"`
	}
	payload := struct { identity; ActionID string `json:"action_id"` }{b.identity(i, u), actionID}
	if err := b.post(UndoConnectPath, payload, &result); err != nil {
		b.edit(s, i, "Отмена привязки не выполнена: "+err.Error(), nil)
		return
	}
	if result.Status == "already_undone" {
		b.edit(s, i, "Эта привязка уже была отменена ранее.", nil)
		return
	}
	if i.Message != nil {
		components := []discordgo.MessageComponent{}
		_, editErr := s.ChannelMessageEditComplex(&discordgo.MessageEdit{ID: i.Message.ID, Channel: i.ChannelID, Components: &components})
		if editErr != nil { log.Printf("Discord self-connect audit message update failed: %v", editErr) }
	}
	b.edit(s, i, fmt.Sprintf("✅ Привязка **%s** отменена. Actor снова доступен для привязки, персонаж приложения сохранён как черновик без владельца, слот освобождён.", result.CharacterName), nil)
}

func myCharactersEmbeds(result myCharactersResult, _ string) []*discordgo.MessageEmbed {
	slotLine := fmt.Sprintf("Активные слоты: **%d/%d** · свободно: **%d**", result.ActiveSlots, result.MaxActiveSlots, result.RemainingSlots)
	embed := &discordgo.MessageEmbed{
		Title:       "Мои персонажи",
		Description: slotLine,
		Color:       0x5865F2,
	}
	if len(result.Characters) == 0 {
		embed.Description += "\n\nПока нет привязанных персонажей. Используй `/character-connect`, чтобы выбрать Actor из Foundry."
		return []*discordgo.MessageEmbed{embed}
	}

	const maxCharacters = 25 // Discord allows at most 25 fields in one embed.
	for _, character := range result.Characters {
		lines := []string{
			fmt.Sprintf("Уровень **%d** · Опыт **%d XP**", character.Level, character.XP),
			characterStatusLabel(character.Status),
		}
		if character.Ancestry != "" {
			lines = append(lines, "**Родословная:** "+character.Ancestry)
		}
		if character.Class != "" {
			lines = append(lines, "**Класс:** "+character.Class)
		}
		name := strings.TrimSpace(character.Name)
		if name == "" {
			name = "Персонаж без имени"
		}
		if len([]rune(name)) > 256 {
			name = string([]rune(name)[:256])
		}
		value := strings.Join(lines, "\n")
		if len([]rune(value)) > 1024 {
			value = string([]rune(value)[:1021]) + "…"
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  value,
			Inline: false,
		})
		if len(embed.Fields) == maxCharacters {
			break
		}
	}
	if len(result.Characters) > maxCharacters {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Показано %d из %d персонажей", maxCharacters, len(result.Characters)),
		}
	}
	return []*discordgo.MessageEmbed{embed}
}

func (b *bot) handleUnlinkButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	key := strings.TrimPrefix(data.CustomID, "fm-unlink-confirm:")
	u := interactionUser(i)
	if u == nil {
		b.reply(s, i, "Не удалось определить пользователя.", nil)
		return
	}
	b.mu.Lock()
	pending, ok := b.pendingUnlinks[key]
	valid := ok && time.Now().Before(pending.expiresAt) && pending.discordID == u.ID && pending.guildID == i.GuildID
	if valid {
		delete(b.pendingUnlinks, key)
	}
	b.mu.Unlock()
	if !valid {
		b.reply(s, i, "Подтверждение отвязки недействительно или истекло. Повтори /character-unlink.", nil)
		return
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		return
	}
	id := b.identity(i, u)
	payload := map[string]any{
		"guild_id":         id.GuildID,
		"discord_user_id":  id.DiscordUserID,
		"discord_role_ids": id.DiscordRoleIDs,
		"actor_uuid":       pending.actorUUID,
	}
	if pending.ownerDiscordID != "" {
		payload["owner_discord_user_id"] = pending.ownerDiscordID
	}
	if err := b.post(UnlinkPath, payload, nil); err != nil {
		b.edit(s, i, "Не удалось отвязать Foundry Actor: "+err.Error(), nil)
		return
	}
	if pending.ownerDiscordID != "" {
		b.edit(s, i, "Foundry Actor отвязан. Владелец персонажа явно переназначен на <@"+pending.ownerDiscordID+">; история владения сохранена. Теперь его можно снова связать через /character-link.", nil)
	} else {
		b.edit(s, i, "Foundry Actor отвязан. Текущий владелец персонажа сохранён. Для повторной привязки используй /character-link.", nil)
	}
}

func (b *bot) editEmbeds(s *discordgo.Session, i *discordgo.InteractionCreate, embeds []*discordgo.MessageEmbed) {
	if len(embeds) == 0 {
		b.edit(s, i, "Нет данных для отображения.", nil)
		return
	}
	if len(embeds) > 10 {
		embeds = embeds[:10]
	}
	content := ""
	components := []discordgo.MessageComponent{}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Embeds: &embeds, Components: &components})
	if err != nil {
		log.Printf("Discord grouped embeds edit: %v", err)
	}
}

func (b *bot) editEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	content := ""
	components := []discordgo.MessageComponent{}
	embeds := []*discordgo.MessageEmbed{embed}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		log.Printf("Discord embed edit: %v", err)
	}
}

func resolveFoundryImageURL(image *string, publicBaseURL string) *string {
	if image == nil || strings.TrimSpace(*image) == "" {
		return image
	}
	source, err := url.Parse(strings.TrimSpace(*image))
	if err != nil {
		return nil
	}
	if source.IsAbs() {
		if source.Scheme == "http" || source.Scheme == "https" {
			value := source.String()
			return &value
		}
		return nil
	}
	base, err := url.Parse(strings.TrimSpace(publicBaseURL))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/"
	resolved := base.ResolveReference(&url.URL{Path: strings.TrimLeft(source.Path, "/")})
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return nil
	}
	value := resolved.String()
	return &value
}

func characterPortraitURL(image *string, publicBaseURL string) *string {
	if image != nil && strings.TrimSpace(*image) != "" {
		if resolved := resolveFoundryImageURL(image, publicBaseURL); resolved != nil {
			return resolved
		}
	}
	placeholder := "systems/pf2e/icons/default-icons/character.svg"
	return resolveFoundryImageURL(&placeholder, publicBaseURL)
}

func characterInventoryEmbed(c characterDetails) *discordgo.MessageEmbed {
	embeds := characterInventoryEmbeds(c)
	if len(embeds) == 0 {
		return &discordgo.MessageEmbed{Title: "Инвентарь · " + c.Identity.Name, Description: "В инвентаре нет предметов.", Color: 0x5865F2}
	}
	return embeds[0]
}

func characterInventoryEmbeds(c characterDetails) []*discordgo.MessageEmbed {
	containerNames := make(map[string]string)
	for _, item := range c.Items {
		if item.ID != "" && item.Type == "backpack" {
			containerNames[item.ID] = item.Name
		}
	}
	groupLabels := map[string]string{
		"weapon": "Оружие", "armor": "Доспехи", "shield": "Щиты", "equipment": "Снаряжение",
		"consumable": "Расходники", "backpack": "Контейнеры", "treasure": "Сокровища",
	}
	order := []string{"weapon", "armor", "shield", "equipment", "consumable", "backpack", "treasure"}
	groups := make(map[string][]string)
	total := 0
	for _, item := range c.Items {
		if !isInventoryItem(item) || item.Name == "" {
			continue
		}
		group := item.Type
		if groupLabels[group] == "" {
			group = "equipment"
		}
		line := item.Name
		if item.Quantity != nil && *item.Quantity > 1 {
			line = appendLocalizedCharacterSuffix(line, " × "+strconv.Itoa(*item.Quantity))
		}
		state := inventoryStateLabel(item, containerNames)
		if state != "" {
			line += " · **" + state + "**"
		}
		if item.Usage != nil {
			if usage := inventoryUsageLabel(*item.Usage); usage != "" {
				line += " · " + usage
			}
		}
		if item.ContainerID != nil && state != "В РЮКЗАКЕ" {
			if name := containerNames[*item.ContainerID]; name != "" {
				line += " · в " + name
			} else {
				line += " · внутри контейнера"
			}
		}
		groups[group] = append(groups[group], line)
		total++
	}
	embed := &discordgo.MessageEmbed{Title: "Инвентарь · " + c.Identity.Name, Color: 0x5865F2}
	if total == 0 {
		embed.Description = "В инвентаре нет предметов или они отсутствуют в последней синхронизации."
		return []*discordgo.MessageEmbed{embed}
	}
	usedRunes, truncated := 0, false
	for _, group := range order {
		if len(groups[group]) == 0 {
			continue
		}
		chunks := splitCharacterLines(groups[group], 20, 850)
		for page, chunk := range chunks {
			if len(embed.Fields) >= 25 {
				truncated = true
				break
			}
			name := groupLabels[group]
			if len(chunks) > 1 {
				name += fmt.Sprintf(" · %d/%d", page+1, len(chunks))
			}
			value := strings.Join(chunk, "\n")
			fieldRunes := len([]rune(name)) + len([]rune(value))
			if usedRunes+fieldRunes > 5500 {
				truncated = true
				break
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: name, Value: value, Inline: false})
			usedRunes += fieldRunes
		}
		if truncated {
			break
		}
	}
	if truncated && len(embed.Fields) < 25 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Список сокращён", Value: "Остальные предметы не поместились в одно сообщение Discord.", Inline: false})
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Всего предметов: %d · данные последней синхронизации Foundry", total)}
	return []*discordgo.MessageEmbed{embed}
}

func isInventoryItem(item characterItem) bool {
	if item.Category == "equipment" {
		return true
	}
	return item.Quantity != nil || item.Equipped != nil || item.Usage != nil || item.ContainerID != nil
}

func inventoryStateLabel(item characterItem, containerNames map[string]string) string {
	if item.Equipped == nil {
		return ""
	}
	equipped := item.Equipped
	states := make([]string, 0, 2)
	carryType := ""
	if equipped.CarryType != nil {
		carryType = strings.ToLower(strings.TrimSpace(*equipped.CarryType))
	}
	if carryType == "worn" && item.Type == "armor" && equipped.InSlot != nil && *equipped.InSlot {
		states = append(states, "НАДЕТО КАК ДОСПЕХ")
	}
	if equipped.Invested != nil && *equipped.Invested {
		states = append(states, "ИНВЕСТИРОВАНО")
	}
	switch carryType {
	case "held":
		states = append(states, "В РУКАХ")
	case "stowed":
		if item.ContainerID != nil && containerNames[*item.ContainerID] != "" {
			states = append(states, "В РЮКЗАКЕ")
		} else {
			states = append(states, "УБРАНО")
		}
	case "dropped":
		states = append(states, "БРОШЕНО")
	}
	return strings.Join(states, " · ")
}

func inventoryUsageLabel(usage string) string {
	switch usage {
	case "held-in-one-hand":
		return "1 рука"
	case "held-in-two-hands":
		return "2 руки"
	default:
		return ""
	}
}

func characterSpellsEmbed(c characterDetails) *discordgo.MessageEmbed {
	embeds := characterSpellsEmbeds(c)
	if len(embeds) == 0 {
		return &discordgo.MessageEmbed{Title: "Заклинания · " + c.Identity.Name, Description: "Заклинания не найдены.", Color: 0x5865F2}
	}
	return embeds[0]
}

func characterSpellsEmbeds(c characterDetails) []*discordgo.MessageEmbed {
	type spellGroup struct {
		category  int
		stateRank int
		tradition string
		rank      int
		list      string
		state     string
	}
	groups := make(map[spellGroup][]string)
	keys := make([]spellGroup, 0)
	addSpell := func(item characterItem, category, stateRank int, state string, rank int) {
		tradition := "Традиция не указана"
		if item.Tradition != nil && *item.Tradition != "" {
			tradition = spellTraditionLabel(*item.Tradition)
		}
		list := ""
		if item.SpellList != nil {
			list = *item.SpellList
		}
		if category == 2 {
			tradition = "Фокусная магия"
			list = ""
		}
		key := spellGroup{category: category, stateRank: stateRank, tradition: tradition, rank: rank, list: list, state: state}
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], item.Name)
	}

	for _, item := range c.Items {
		if item.Type != "spell" || item.Name == "" {
			continue
		}
		rank := -1
		if item.Level != nil {
			rank = *item.Level
		}
		if item.Cantrip {
			rank = 0
		}
		spellType := ""
		if item.SpellcastingType != nil {
			spellType = strings.ToLower(strings.TrimSpace(*item.SpellcastingType))
		}
		listName := ""
		if item.SpellList != nil {
			listName = strings.ToLower(*item.SpellList)
		}
		if spellType == "focus" || strings.Contains(listName, "focus") || strings.Contains(listName, "фокус") {
			focusOrder := 1
			if item.Cantrip || rank == 0 {
				focusOrder = 0
			}
			addSpell(item, 2, 0, "Фокусные заклинания", focusOrder)
			continue
		}
		if spellType == "spontaneous" {
			addSpell(item, 0, 0, "В репертуаре", rank)
			continue
		}
		if spellType == "prepared" {
			switch {
			case item.Prepared == nil:
				addSpell(item, 1, 2, "Статус подготовки не определён", rank)
			case *item.Prepared:
				if len(item.PreparedRanks) > 0 {
					for _, preparedRank := range item.PreparedRanks {
						addSpell(item, 1, 0, "Подготовлено", preparedRank)
					}
				} else {
					addSpell(item, 1, 0, "Подготовлено", rank)
				}
			default:
				addSpell(item, 1, 1, "Подготовка", rank)
			}
			continue
		}
		if item.Prepared != nil && *item.Prepared {
			addSpell(item, 1, 0, "Подготовлено", rank)
		} else {
			addSpell(item, 1, 1, "Доступно", rank)
		}
	}

	embed := &discordgo.MessageEmbed{Title: "Заклинания · " + c.Identity.Name, Color: 0x5865F2}
	if len(keys) == 0 {
		embed.Description = "В последней синхронизации Foundry заклинания не найдены."
		return []*discordgo.MessageEmbed{embed}
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if keys[i].category != keys[j].category {
			return keys[i].category < keys[j].category
		}
		if keys[i].stateRank != keys[j].stateRank {
			return keys[i].stateRank < keys[j].stateRank
		}
		if keys[i].category == 2 {
			if (keys[i].rank == 0) != (keys[j].rank == 0) {
				return keys[i].rank == 0
			}
			return strings.ToLower(keys[i].list) < strings.ToLower(keys[j].list)
		}
		if keys[i].rank != keys[j].rank {
			return keys[i].rank < keys[j].rank
		}
		if keys[i].tradition != keys[j].tradition {
			return keys[i].tradition < keys[j].tradition
		}
		return keys[i].list < keys[j].list
	})

	usedRunes, truncated := 0, false
	for _, key := range keys {
		names := groups[key]
		sort.SliceStable(names, func(i, j int) bool {
			return strings.ToLower(names[i]) < strings.ToLower(names[j])
		})
		chunks := splitCharacterLines(names, 20, 850)
		for page, chunk := range chunks {
			if len(embed.Fields) >= 25 {
				truncated = true
				break
			}
			rankLabel := "ранг не указан"
			if key.category == 2 {
				if key.rank == 0 {
					rankLabel = "чары"
				} else {
					rankLabel = "заклинания"
				}
			} else if key.rank == 0 {
				rankLabel = "чары"
			} else if key.rank > 0 {
				rankLabel = fmt.Sprintf("ранг %d", key.rank)
			}
			field := key.state + " · " + key.tradition + " · " + rankLabel
			if key.category == 0 {
				field = "СПОНТАННОЕ КОЛДОВСТВО · " + field
			} else if key.category == 1 {
				if key.stateRank == 0 {
					field = "ПОДГОТАВЛИВАЮЩЕЕ КОЛДОВСТВО · " + field
				} else {
					field = "ПОДГОТОВКА · " + field
				}
			} else {
				field = "ФОКУСНЫЕ ЗАКЛИНАНИЯ · " + field
			}
			if key.list != "" {
				field += " · " + key.list
			}
			if len(chunks) > 1 {
				field += fmt.Sprintf(" · %d/%d", page+1, len(chunks))
			}
			value := strings.Join(chunk, "\n")
			fieldRunes := len([]rune(field)) + len([]rune(value))
			if usedRunes+fieldRunes > 5500 {
				truncated = true
				break
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: field, Value: value, Inline: false})
			usedRunes += fieldRunes
		}
		if truncated {
			break
		}
	}
	if truncated {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Список превышает лимиты одного сообщения Discord."}
	} else {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: "Заклинания из последней синхронизации Foundry"}
	}
	return []*discordgo.MessageEmbed{embed}
}

func splitCharacterLines(lines []string, maxLines, maxRunes int) [][]string {
	var chunks [][]string
	current := make([]string, 0, maxLines)
	currentRunes := 0
	for _, line := range lines {
		lineRunes := len([]rune(line))
		if len(current) > 0 && (len(current) >= maxLines || currentRunes+lineRunes+1 > maxRunes) {
			chunks = append(chunks, current)
			current = make([]string, 0, maxLines)
			currentRunes = 0
		}
		current = append(current, line)
		currentRunes += lineRunes + 1
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func spellTraditionLabel(value string) string {
	switch strings.ToLower(value) {
	case "arcane":
		return "Арканная традиция"
	case "divine":
		return "Божественная традиция"
	case "occult":
		return "Оккультная традиция"
	case "primal":
		return "Природная традиция"
	default:
		return value
	}
}

func logCharacterCardDiagnostics(c characterDetails) {
	value := strings.TrimSpace(os.Getenv("FOUNDRY_CHARACTER_DIAGNOSTICS"))
	if !strings.EqualFold(value, "1") && !strings.EqualFold(value, "true") {
		return
	}
	className, ancestryName := "", ""
	if c.Build.Class != nil {
		className = c.Build.Class.Name
	}
	if c.Build.Ancestry != nil {
		ancestryName = c.Build.Ancestry.Name
	}
	hpCurrent, hpMaximum := (*int)(nil), (*int)(nil)
	if c.Attributes.HP != nil {
		hpCurrent, hpMaximum = c.Attributes.HP.Current, c.Attributes.HP.Max
	}
	log.Printf("[character-diagnostics] discord-card actor_uuid=%q name=%q class=%q ancestry=%q level=%s hp_current=%s hp_max=%s xp=%s/%s actor_ability_keys=%v reconstructed_ability_keys=%v",
		c.Identity.UUID, c.Identity.Name, className, ancestryName,
		diagnosticInt(c.Progression.Level), diagnosticInt(hpCurrent), diagnosticInt(hpMaximum),
		diagnosticInt(c.Progression.Experience.Value), diagnosticInt(c.Progression.Experience.Max),
		sortedAbilityKeys(c.Statistics.Abilities), sortedAbilityKeys(c.Statistics.ReconstructedAbilities))
	for _, key := range []string{"str", "dex", "con", "int", "wis", "cha"} {
		if actorValue, ok := c.Statistics.Abilities[key]; ok && (actorValue.Score != nil || actorValue.Modifier != nil) {
			log.Printf("[character-diagnostics] discord-card ability=%s source=actor score=%s modifier=%s derived=%t",
				key, diagnosticInt(actorValue.Score), diagnosticInt(actorValue.Modifier), actorValue.Derived)
			continue
		}
		if reconstructed, ok := c.Statistics.ReconstructedAbilities[key]; ok && (reconstructed.Score != nil || reconstructed.Modifier != nil) {
			log.Printf("[character-diagnostics] discord-card ability=%s source=reconstructed score=%s modifier=%s derived=%t",
				key, diagnosticInt(reconstructed.Score), diagnosticInt(reconstructed.Modifier), reconstructed.Derived)
			continue
		}
		log.Printf("[character-diagnostics] discord-card ability=%s source=missing", key)
	}
}

func diagnosticInt(value *int) string {
	if value == nil {
		return "<nil>"
	}
	return strconv.Itoa(*value)
}

func sortedAbilityKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func characterEmbed(c characterDetails) *discordgo.MessageEmbed {
	logCharacterCardDiagnostics(c)
	embed := &discordgo.MessageEmbed{Title: c.Identity.Name, Color: 0x5865F2}
	if c.Identity.Image != nil && (strings.HasPrefix(*c.Identity.Image, "https://") || strings.HasPrefix(*c.Identity.Image, "http://")) {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: *c.Identity.Image}
	}
	if c.Attributes.Dying != nil && c.Attributes.Dying.Current != nil && c.Attributes.Dying.Max != nil &&
		*c.Attributes.Dying.Max > 0 && *c.Attributes.Dying.Current >= *c.Attributes.Dying.Max {
		embed.Title = "💀 МЁРТВ · " + c.Identity.Name
		embed.Color = 0xED4245
	}

	className, ancestryName := "—", "—"
	if c.Build.Class != nil && strings.TrimSpace(c.Build.Class.Name) != "" {
		className = c.Build.Class.Name
	}
	if c.Build.Ancestry != nil && strings.TrimSpace(c.Build.Ancestry.Name) != "" {
		ancestryName = c.Build.Ancestry.Name
	}
	embed.Fields = append(embed.Fields,
		&discordgo.MessageEmbedField{Name: "Класс", Value: className, Inline: true},
		&discordgo.MessageEmbedField{Name: "Родословная", Value: ancestryName, Inline: true},
		&discordgo.MessageEmbedField{Name: "Уровень", Value: intValue(c.Progression.Level), Inline: true},
	)

	hpCurrent, hpMaximum := "—", "—"
	if c.Attributes.HP != nil {
		hpCurrent = intValue(c.Attributes.HP.Current)
		hpMaximum = intValue(c.Attributes.HP.Max)
	}
	hp := hpCurrent + " / " + hpMaximum
	if c.Attributes.HP != nil && c.Attributes.HP.Temp != nil && *c.Attributes.HP.Temp > 0 {
		hp += fmt.Sprintf(" (+%d временных)", *c.Attributes.HP.Temp)
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "ОЗ", Value: hp, Inline: true})

	abilityOrder := []string{"str", "dex", "con", "int", "wis", "cha"}
	abilityNames := map[string]string{"str": "СИЛ", "dex": "ЛОВ", "con": "ТЕЛ", "int": "ИНТ", "wis": "МДР", "cha": "ХАР"}
	abilities := make([]string, 0, len(abilityOrder))
	for _, key := range abilityOrder {
		value, ok := c.Statistics.Abilities[key]
		if !ok || (value.Score == nil && value.Modifier == nil) {
			if reconstructed, exists := c.Statistics.ReconstructedAbilities[key]; exists {
				value, ok = reconstructed, true
			}
		}
		modifier := value.Modifier
		if modifier == nil && value.Score != nil {
			calculated := *value.Score/2 - 5
			modifier = &calculated
		}
		if !ok || modifier == nil {
			abilities = append(abilities, abilityNames[key]+": —")
			continue
		}
		partial := ""
		if value.Score != nil && *value.Score%2 != 0 {
			partial = "½"
		}
		abilities = append(abilities, fmt.Sprintf("%s: %+d%s", abilityNames[key], *modifier, partial))
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "Характеристики", Value: strings.Join(abilities, " · "), Inline: false,
	})

	xpValue, xpMaximum := intValue(c.Progression.Experience.Value), intValue(c.Progression.Experience.Max)
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "Опыт", Value: xpValue + " / " + xpMaximum, Inline: true,
	})
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Данные персонажа из последней синхронизации Foundry"}
	return embed
}

type characterLevelEntry struct {
	name  string
	level *int
}

func sortCharacterLevelEntries(entries []characterLevelEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].level == nil && entries[j].level != nil {
			return false
		}
		if entries[i].level != nil && entries[j].level == nil {
			return true
		}
		if entries[i].level != nil && entries[j].level != nil && *entries[i].level != *entries[j].level {
			return *entries[i].level < *entries[j].level
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})
}

func characterAbilitiesEmbed(c characterDetails) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{Title: "Способности · " + c.Identity.Name, Color: 0x5865F2}
	classFeatures := make([]characterLevelEntry, 0)
	classAbilities := make([]characterLevelEntry, 0)
	ancestryFeatures := make([]characterLevelEntry, 0)
	ancestryAbilities := make([]characterLevelEntry, 0)
	skillAbilities := make([]characterLevelEntry, 0)
	otherAbilities := make([]characterLevelEntry, 0)
	seenClassFeatures := make(map[string]struct{})
	seenAncestryFeatures := make(map[string]struct{})

	if c.Progression.Level != nil {
		for _, feature := range c.Build.AncestryFeatures {
			if feature.Name == "" || feature.Level == nil || *feature.Level > *c.Progression.Level {
				continue
			}
			key := characterEntryKey(feature.Name)
			if _, exists := seenAncestryFeatures[key]; exists {
				continue
			}
			seenAncestryFeatures[key] = struct{}{}
			ancestryFeatures = append(ancestryFeatures, characterLevelEntry{name: feature.Name, level: feature.Level})
		}
	}

	if c.Build.ClassProficiencies != nil && c.Progression.Level != nil {
		for _, feature := range c.Build.ClassProficiencies.Features {
			if feature.Name == "" || feature.Level == nil || *feature.Level > *c.Progression.Level {
				continue
			}
			key := characterEntryKey(feature.Name)
			if _, exists := seenClassFeatures[key]; exists {
				continue
			}
			seenClassFeatures[key] = struct{}{}
			classFeatures = append(classFeatures, characterLevelEntry{name: feature.Name, level: feature.Level})
		}
	}

	for _, item := range c.Items {
		if item.Name == "" {
			continue
		}
		if item.Type == "action" {
			continue
		}
		if item.Type != "feat" {
			continue
		}
		entry := characterLevelEntry{name: item.Name, level: item.Level}
		// Class progression references may identify an owned feat whose Foundry
		// category is still "general". Suppress that Actor Item here as well,
		// otherwise the same class feature is rendered again under other abilities.
		if _, exists := seenClassFeatures[characterEntryKey(item.Name)]; exists {
			continue
		}
		switch stringValue(item.SystemCategory) {
		case "class":
			classAbilities = append(classAbilities, entry)
		case "classfeature":
			key := characterEntryKey(item.Name)
			if _, exists := seenClassFeatures[key]; !exists {
				seenClassFeatures[key] = struct{}{}
				classFeatures = append(classFeatures, entry)
			}
		case "ancestryfeature":
			key := characterEntryKey(item.Name)
			if _, exists := seenAncestryFeatures[key]; !exists {
				seenAncestryFeatures[key] = struct{}{}
				ancestryFeatures = append(ancestryFeatures, entry)
			}
		case "ancestry", "heritage":
			ancestryAbilities = append(ancestryAbilities, entry)
		case "skill":
			skillAbilities = append(skillAbilities, entry)
		default:
			otherAbilities = append(otherAbilities, entry)
		}
	}

	for _, entries := range [][]characterLevelEntry{classFeatures, classAbilities, ancestryFeatures, ancestryAbilities, skillAbilities, otherAbilities} {
		sortCharacterLevelEntries(entries)
	}
	addCharacterLevelField(embed, "Особенности класса", classFeatures)
	addCharacterLevelField(embed, "Способности класса", classAbilities)
	addCharacterLevelField(embed, "Особенности родословной", ancestryFeatures)
	addCharacterLevelField(embed, "Способности родословной", ancestryAbilities)
	addCharacterLevelField(embed, "Способности навыков", skillAbilities)
	addCharacterLevelField(embed, "Другие способности", otherAbilities)

	if len(embed.Fields) == 0 {
		embed.Description = "В снимке Foundry нет подробных данных о способностях."
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Данные из последней синхронизации Foundry"}
	return embed
}

func addCharacterLevelField(embed *discordgo.MessageEmbed, name string, entries []characterLevelEntry) {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.name)
	}
	addCharacterListField(embed, name, lines)
}

func characterSkillsEmbed(c characterDetails) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{Title: "Навыки · " + c.Identity.Name, Color: 0x5865F2}
	addCharacterListField(embed, "Все навыки", characterSkillLines(c.Skills.Standard, c.Skills.Lore))

	type skillAbilityGroup struct {
		name      string
		abilities []string
	}
	groupsByName := make(map[string][]string)
	for _, item := range c.Items {
		if item.Type != "feat" || stringValue(item.SystemCategory) != "skill" || item.Name == "" {
			continue
		}
		group := "Навык не определён"
		if item.Skill != nil && strings.TrimSpace(*item.Skill) != "" {
			group = characterSkillDisplayName(*item.Skill)
		}
		groupsByName[group] = append(groupsByName[group], item.Name)
	}
	groups := make([]skillAbilityGroup, 0, len(groupsByName))
	for name, abilities := range groupsByName {
		sort.SliceStable(abilities, func(i, j int) bool {
			return strings.ToLower(abilities[i]) < strings.ToLower(abilities[j])
		})
		groups = append(groups, skillAbilityGroup{name: name, abilities: abilities})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		leftUnknown := groups[i].name == "Навык не определён"
		rightUnknown := groups[j].name == "Навык не определён"
		if leftUnknown != rightUnknown {
			return !leftUnknown
		}
		if len(groups[i].abilities) != len(groups[j].abilities) {
			return len(groups[i].abilities) > len(groups[j].abilities)
		}
		return strings.ToLower(groups[i].name) < strings.ToLower(groups[j].name)
	})

	var skillAbilityLines []string
	for _, group := range groups {
		skillAbilityLines = append(skillAbilityLines, group.name)
		for _, ability := range group.abilities {
			skillAbilityLines = append(skillAbilityLines, "- "+ability)
		}
	}
	addCharacterListField(embed, "Способности навыков", skillAbilityLines)
	if len(embed.Fields) == 0 {
		embed.Description = "В снимке Foundry нет обученных навыков или способностей навыков."
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Навыки и способности из последней синхронизации Foundry"}
	return embed
}

func characterSkillDisplayName(slug string) string {
	names := map[string]string{
		"acrobatics": "Акробатика", "arcana": "Аркана", "athletics": "Атлетика",
		"crafting": "Ремесло", "deception": "Обман", "diplomacy": "Дипломатия",
		"intimidation": "Запугивание", "medicine": "Медицина", "nature": "Природа",
		"occultism": "Оккультизм", "performance": "Выступление", "religion": "Религия",
		"society": "Общество", "stealth": "Скрытность", "survival": "Выживание", "thievery": "Воровство",
	}
	if name := names[strings.ToLower(strings.TrimSpace(slug))]; name != "" {
		return name
	}
	if strings.HasPrefix(strings.ToLower(slug), "lore-") {
		return loreSkillName(slug)
	}
	return humanizeSkillKey(slug)
}

func characterMetricsEmbed(c characterDetails) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{Title: "Метрики кампании · " + c.Identity.Name,
		Description: "Сбор и отображение кампанийных метрик пока не подключены к карточке персонажа. Уровень и опыт доступны в основной карточке.", Color: 0x5865F2}
}

func directoryEmbed(title, description string, entries []string, maxShown int) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{Title: title, Description: description, Color: 0x5865F2}
	shown := entries
	if len(shown) > maxShown {
		shown = shown[:maxShown]
	}
	const perField = 8
	for start, fieldNumber := 0, 1; start < len(shown); start, fieldNumber = start+perField, fieldNumber+1 {
		end := start + perField
		if end > len(shown) {
			end = len(shown)
		}
		lines := make([]string, 0, end-start)
		for index, value := range shown[start:end] {
			lines = append(lines, fmt.Sprintf("%d. %s", start+index+1, value))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: fmt.Sprintf("Список · %d", fieldNumber), Value: strings.Join(lines, "\n"), Inline: false})
	}
	footer := fmt.Sprintf("Показано %d из %d", len(shown), len(entries))
	if len(entries) > len(shown) {
		footer += fmt.Sprintf(" · ещё %d не показано", len(entries)-len(shown))
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: footer}
	return embed
}

func characterStatusLabel(status string) string {
	switch status {
	case "draft":
		return "черновик"
	case "pending_approval":
		return "ожидает одобрения"
	case "active":
		return "активен"
	case "unavailable":
		return "недоступен"
	case "dead":
		return "погиб"
	case "retired":
		return "выведен из игры"
	case "npc":
		return "NPC"
	case "archived":
		return "архив"
	default:
		return status
	}
}
func addCharacterListField(embed *discordgo.MessageEmbed, name string, entries []string) {
	if len(entries) == 0 {
		return
	}
	const maxRunes = 1024
	chunk := make([]string, 0)
	chunkRunes := 0
	fieldNumber := 1
	flush := func() {
		if len(chunk) == 0 {
			return
		}
		fieldName := name
		if fieldNumber > 1 {
			fieldName = fmt.Sprintf("%s (%d)", name, fieldNumber)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: fieldName, Value: strings.Join(chunk, "\n"), Inline: false,
		})
		fieldNumber++
		chunk = make([]string, 0)
		chunkRunes = 0
	}
	for _, entry := range entries {
		line := strings.TrimSpace(entry)
		if line == "" {
			continue
		}
		lineRunes := len([]rune(line))
		if lineRunes > maxRunes {
			runes := []rune(line)
			for len(runes) > maxRunes {
				if len(chunk) > 0 {
					flush()
				}
				chunk = append(chunk, string(runes[:maxRunes]))
				flush()
				runes = runes[maxRunes:]
			}
			if len(runes) > 0 {
				line = string(runes)
				lineRunes = len(runes)
			} else {
				continue
			}
		}
		additional := lineRunes
		if len(chunk) > 0 {
			additional++
		}
		if chunkRunes+additional > maxRunes {
			flush()
		}
		chunk = append(chunk, line)
		chunkRunes += lineRunes
		if len(chunk) > 1 {
			chunkRunes++
		}
	}
	flush()
}

func appendUniqueCharacterEntry(entries *[]string, seen map[string]struct{}, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	key := characterEntryKey(name)
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}
	*entries = append(*entries, name)
}

func characterEntryKey(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	if slash := strings.Index(key, " / "); slash >= 0 {
		key = strings.TrimSpace(key[:slash])
	}
	return key
}

func actionDisplayName(item characterItem) string {
	name := item.Name
	if item.ActionType == nil {
		return name
	}
	labels := map[string]string{
		"action":   "действие",
		"reaction": "реакция",
		"free":     "свободное действие",
		"passive":  "пассивное",
	}
	if label := labels[stringValue(item.ActionType)]; label != "" {
		return appendLocalizedCharacterSuffix(name, " — "+label)
	}
	return name
}

// appendLocalizedCharacterSuffix keeps qualifiers next to the localized name
// instead of placing them after the English translation.
func appendLocalizedCharacterSuffix(name, suffix string) string {
	localized, translated, hasTranslation := strings.Cut(name, " / ")
	if !hasTranslation {
		return name + suffix
	}
	return localized + suffix + " / " + translated
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*value))
}

func characterSkillLines(standard, lore map[string]characterSkill) []string {
	type skillLine struct {
		name string
		rank int
	}
	names := map[string]string{
		"acrobatics": "Акробатика", "arcana": "Аркана", "athletics": "Атлетика",
		"crafting": "Ремесло", "deception": "Обман", "diplomacy": "Дипломатия",
		"intimidation": "Запугивание", "medicine": "Медицина", "nature": "Природа",
		"occultism": "Оккультизм", "performance": "Выступление", "religion": "Религия",
		"society": "Общество", "stealth": "Скрытность", "survival": "Выживание",
		"thievery": "Воровство",
	}
	entries := make([]skillLine, 0, len(standard)+len(lore))
	for key, skill := range standard {
		name := names[key]
		if name == "" {
			name = humanizeSkillKey(key)
		}
		rank := -1
		if skill.Rank != nil {
			rank = *skill.Rank
		}
		entries = append(entries, skillLine{name: name, rank: rank})
	}
	for key, skill := range lore {
		rank := -1
		if skill.Rank != nil {
			rank = *skill.Rank
		}
		entries = append(entries, skillLine{name: loreSkillName(key), rank: rank})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].rank != entries[j].rank {
			return entries[i].rank > entries[j].rank
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		label := "—"
		if entry.rank >= 0 {
			label = proficiencyLabel(&entry.rank)
		}
		lines = append(lines, fmt.Sprintf("%s — %s", entry.name, label))
	}
	return lines
}

func proficiencyLabel(rank *int) string {
	if rank == nil {
		return ""
	}
	switch *rank {
	case 0:
		return "Не обучен"
	case 1:
		return "Обучен"
	case 2:
		return "Эксперт"
	case 3:
		return "Мастер"
	case 4:
		return "Легенда"
	default:
		return ""
	}
}

func loreSkillName(key string) string {
	key = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(key)), "lore-")
	known := map[string]string{
		"architecture": "Знания архитектуры",
		"art":          "Знания искусства",
		"astronomy":    "Знания астрономии",
		"farming":      "Знания земледелия",
		"fishing":      "Знания рыболовства",
		"games":        "Знания игр",
		"genealogy":    "Знания генеалогии",
		"herbalism":    "Знания травничества",
		"hunting":      "Знания охоты",
		"legal":        "Знания права",
		"library":      "Знания библиотечного дела",
		"mercantile":   "Знания торговли",
		"mining":       "Знания горного дела",
		"sailing":      "Знания мореплавания",
		"scouting":     "Знания разведки",
		"warfare":      "Знания военного дела",
	}
	if name := known[key]; name != "" {
		return name
	}
	return "Знания: " + humanizeSkillKey(key)
}

func humanizeSkillKey(key string) string {
	key = strings.ReplaceAll(strings.ReplaceAll(key, "-", " "), "_", " ")
	key = strings.TrimSpace(key)
	if key == "" {
		return "Неизвестный навык"
	}
	runes := []rune(key)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

func intValue(value *int) string {
	if value == nil {
		return "—"
	}
	return strconv.Itoa(*value)
}

func (b *bot) fetchIntegrationStatus() (integrationStatus, error) {
	var status integrationStatus
	err := b.post(IntegrationStatusPath, struct{}{}, &status)
	return status, err
}

func integrationStatusEmbed(status integrationStatus) *discordgo.MessageEmbed {
	components := []struct {
		name  string
		value healthComponent
	}{
		{"PostgreSQL", status.Database},
		{"Foundry Relay", status.Relay},
		{"Foundry", status.Foundry},
	}
	embed := &discordgo.MessageEmbed{
		Title: "Состояние интеграций",
		Color: 0x57F287,
	}
	for _, item := range components {
		value := "🟢 Доступен"
		if !item.value.OK {
			value = "🔴 Недоступен"
			embed.Color = 0xED4245
		}
		if item.value.Detail != "" {
			value += "\n" + truncateLabel(item.value.Detail)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: item.name, Value: value, Inline: false,
		})
	}
	embed.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return embed
}

func (b *bot) showIntegrationStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	status, err := b.fetchIntegrationStatus()
	if err != nil {
		b.edit(s, i, "Не удалось проверить интеграции: "+err.Error(), nil)
		return
	}
	b.editEmbed(s, i, integrationStatusEmbed(status))
}

func (b *bot) monitorIntegrations(ctx context.Context, s *discordgo.Session) {
	b.checkIntegrationStatus(s)
	ticker := time.NewTicker(b.healthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.checkIntegrationStatus(s)
		}
	}
}

func (b *bot) checkIntegrationStatus(s *discordgo.Session) {
	status, err := b.fetchIntegrationStatus()
	if err != nil {
		b.reportAdminError("integration health check", err)
		return
	}
	allHealthy := status.Database.OK && status.Relay.OK && status.Foundry.OK
	state := fmt.Sprintf("database=%t;relay=%t;foundry=%t", status.Database.OK, status.Relay.OK, status.Foundry.OK)
	b.reportMu.Lock()
	changed := b.lastHealthState != state
	b.lastHealthState = state
	b.reportMu.Unlock()
	if changed {
		embed := integrationStatusEmbed(status)
		if allHealthy {
			embed.Title = "Интеграции доступны или восстановлены"
		} else {
			embed.Title = "Изменилось состояние интеграций"
		}
		b.sendAdminEmbed(s, embed)
	}
}

func (b *bot) sendAdminEmbed(s *discordgo.Session, embed *discordgo.MessageEmbed) {
	if b.adminChannelID == "" {
		return
	}
	if _, err := s.ChannelMessageSendEmbed(b.adminChannelID, embed); err != nil {
		log.Printf("Discord admin channel notification failed: %v", err)
	}
}

func (b *bot) reportAdminError(operation string, err error) {
	if err == nil {
		return
	}
	log.Printf("application error [%s]: %v", operation, err)
	if b.adminChannelID == "" || b.session == nil {
		return
	}
	fingerprint := operation + ": " + err.Error()
	now := time.Now()
	b.reportMu.Lock()
	if previous, ok := b.reportedErrors[fingerprint]; ok && now.Sub(previous) < 5*time.Minute {
		b.reportMu.Unlock()
		return
	}
	b.reportedErrors[fingerprint] = now
	b.reportMu.Unlock()
	embed := &discordgo.MessageEmbed{
		Title:       "Ошибка приложения",
		Color:       0xED4245,
		Description: truncateMessage("Операция: " + operation + "\nОшибка: " + err.Error()),
		Timestamp:   now.UTC().Format(time.RFC3339),
	}
	b.sendAdminEmbed(b.session, embed)
}

func (b *bot) identity(i *discordgo.InteractionCreate, u *discordgo.User) identity {
	roles := []string{}
	if i.Member != nil {
		roles = append(roles, i.Member.Roles...)
	}
	return identity{GuildID: i.GuildID, DiscordUserID: u.ID, DiscordRoleIDs: roles}
}
func interactionUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}
func commandOptions(options []*discordgo.ApplicationCommandInteractionDataOption) map[string]string {
	out := map[string]string{}
	for _, o := range options {
		switch o.Type {
		case discordgo.ApplicationCommandOptionString, discordgo.ApplicationCommandOptionUser:
			// Discord sends both string values and user snowflake IDs as strings.
			// Do not call StringValue for User options: discordgo panics on a type mismatch.
			value, ok := o.Value.(string)
			if ok {
				out[o.Name] = value
			}
		}
	}
	return out
}
func (b *bot) post(path string, payload, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, b.apiURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.apiToken)
	req.Header.Set("Content-Type", "application/json")
	client := b.client
	if path == FoundrySyncPath || path == StartupFoundrySyncPath {
		client = b.syncClient
	}
	resp, err := client.Do(req)
	if err != nil {
		b.reportAdminError("backend request "+path, err)
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		b.reportAdminError("backend response "+path, err)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &apiErr)
		if apiErr.Error == "" {
			apiErr.Error = http.StatusText(resp.StatusCode)
		}
		apiError := errors.New(apiErr.Error)
		if resp.StatusCode >= 500 {
			b.reportAdminError("backend request "+path, apiError)
		}
		return apiError
	}
	if target != nil && len(data) > 0 {
		if err := json.Unmarshal(data, target); err != nil {
			b.reportAdminError("decode backend response "+path, err)
			return err
		}
	}
	return nil
}
func (b *bot) reply(s *discordgo.Session, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	})
	if err != nil {
		log.Printf("Discord response: %v", err)
	}
}
func (b *bot) edit(s *discordgo.Session, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Components: &components})
	if err != nil {
		log.Printf("Discord response edit: %v", err)
	}
}
func randomKey() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
func requiredEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func truncateMessage(value string) string {
	const maxRunes = 1850
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes-20]) + "\n… список сокращён"
}
