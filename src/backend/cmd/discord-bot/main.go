package main

import (
	"bytes"
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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	RegisterPath          = "/api/internal/discord/register"
	ActorsPath            = "/api/internal/discord/actors"
	CharactersPath        = "/api/internal/discord/characters"
	PreviewPath           = "/api/internal/discord/link/preview"
	LinkPath              = "/api/internal/discord/link"
	GrantRolePath         = "/api/internal/discord/roles/grant"
	MapRolePath           = "/api/internal/discord/roles/map"
	DraftCharacterPath    = "/api/internal/discord/characters/draft"
	CharacterCreatePath   = "/api/internal/discord/characters/create"
	CharacterSearchPath   = "/api/internal/discord/characters/search"
	CharacterDetailPath   = "/api/internal/discord/characters/detail"
	MyCharactersPath      = "/api/internal/discord/characters/mine"
	IntegrationStatusPath = "/api/internal/discord/integration-status"
)

type bot struct {
	apiURL, apiToken, guildID string
	foundryPublicBaseURL      string
	campaignRoleID            string
	adminChannelID            string
	healthInterval            time.Duration
	client                    *http.Client
	session                   *discordgo.Session
	mu                        sync.Mutex
	reportMu                  sync.Mutex
	pending                   map[string]pendingLink
	flows                     map[string]*selectionFlow
	reportedErrors            map[string]time.Time
	lastHealthState           string
}
type pendingLink struct {
	actorUUID, characterID, discordID, guildID, ownerDiscordID string
	expiresAt                                                  time.Time
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
		ClassProficiencies *classProficiencyDetails `json:"classProficiencies"`
	} `json:"build"`
	Attributes struct {
		HP *struct {
			Current *int `json:"current"`
			Temp    *int `json:"temp"`
			Max     *int `json:"max"`
		} `json:"hp"`
		Dying *struct {
			Current *int `json:"current"`
			Max     *int `json:"max"`
		} `json:"dying"`
	} `json:"attributes"`
	Statistics struct {
		Abilities map[string]struct {
			Score    *int `json:"score"`
			Modifier *int `json:"modifier"`
		} `json:"abilities"`
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
		client: &http.Client{Timeout: 12 * time.Second}, pending: make(map[string]pendingLink),
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
	if err := registerCommands(s, appID, guildID); err != nil {
		log.Fatal(err)
	}
	log.Printf("FoundryMarches bot connected; guild=%s", guildID)
	if adminChannelID != "" {
		go b.monitorIntegrations(s)
	} else {
		log.Printf("DISCORD_ADMIN_CHANNEL_ID is not set; Discord health alerts are disabled")
	}
	select {}
}

func registerCommands(s *discordgo.Session, appID, guildID string) error {
	commands := []*discordgo.ApplicationCommand{
		{Name: "register", Description: "Зарегистрироваться в FoundryMarches"},
		{Name: "foundry-actors", Description: "Показать несвязанных персонажей Foundry"},
		{Name: "characters", Description: "Показать список персонажей приложения"},
		{Name: "character-link", Description: "Связать Foundry Actor с персонажем и назначить владельца", Options: []*discordgo.ApplicationCommandOption{{Name: "owner", Description: "Владелец персонажа (обязательно)", Type: discordgo.ApplicationCommandOptionUser, Required: true}}},
		{Name: "character-create", Description: "Создать черновик персонажа для себя или игрока", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}, {Name: "owner", Description: "Владелец персонажа (для мастера)", Type: discordgo.ApplicationCommandOptionUser, Required: false}}},
		{Name: "character-draft", Description: "Создать черновик персонажа из выбранного Foundry Actor"},
		{Name: "character", Description: "Найти персонажа по имени (GM видит всех)", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя или часть имени персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}}},
		{Name: "my-character", Description: "Показать своего привязанного персонажа"},
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
	case "character-link":
		o := commandOptions(i.ApplicationCommandData().Options)
		b.startSelection(s, i, id, u, "link", o["owner"])
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
	if err := b.post(ActorsPath, id, &actorResult); err != nil {
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
	}
	b.edit(s, i, label, components)
}

func (b *bot) startCharacterLookup(s *discordgo.Session, i *discordgo.InteractionCreate, id identity, u *discordgo.User, query, mode string) {
	var result struct {
		Actors []actor `json:"actors"`
	}
	if err := b.post(CharacterSearchPath, struct {
		identity
		Query string `json:"query"`
	}{id, query}, &result); err != nil {
		b.edit(s, i, "Не удалось найти персонажа: "+err.Error(), nil)
		return
	}
	if len(result.Actors) == 0 {
		b.edit(s, i, "Персонажи с таким именем не найдены в последней синхронизации Foundry.", nil)
		return
	}
	flow := &selectionFlow{mode: mode, stage: "actor", discordID: u.ID, guildID: i.GuildID,
		actors: result.Actors, expiresAt: time.Now().Add(10 * time.Minute)}
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
	var result struct {
		Actors []actor `json:"actors"`
	}
	if err := b.post(MyCharactersPath, id, &result); err != nil {
		b.edit(s, i, "Не удалось получить твоих персонажей: "+err.Error(), nil)
		return
	}
	if len(result.Actors) == 0 {
		b.edit(s, i, "У тебя пока нет привязанного и подтверждённого персонажа. Попроси мастера связать персонажа с твоим Discord-аккаунтом через `/character-link`.", nil)
		return
	}
	flow := &selectionFlow{mode: "view", stage: "actor", discordID: u.ID, guildID: i.GuildID,
		actors: result.Actors, expiresAt: time.Now().Add(10 * time.Minute)}
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
	if strings.HasPrefix(data.CustomID, "fm-link:") {
		b.handleButton(s, i)
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
			details.Identity.Image = resolveFoundryImageURL(details.Identity.Image, b.foundryPublicBaseURL)
			switch mode {
			case "inventory":
				b.editEmbeds(s, i, characterInventoryEmbeds(details))
			case "spells":
				b.editEmbeds(s, i, characterSpellsEmbeds(details))
			default:
				b.editEmbed(s, i, characterEmbed(details))
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
}

func (b *bot) handleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
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

func (b *bot) editEmbeds(s *discordgo.Session, i *discordgo.InteractionCreate, embeds []*discordgo.MessageEmbed) {
	if len(embeds) == 0 {
		b.edit(s, i, "Нет данных для отображения.", nil)
		return
	}
	b.editEmbed(s, i, embeds[0])
	for _, embed := range embeds[1:] {
		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Embeds: []*discordgo.MessageEmbed{embed}}); err != nil {
			log.Printf("Discord follow-up embed: %v", err)
			return
		}
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
	groupLabels := map[string]string{"weapon":"Оружие","armor":"Доспехи","shield":"Щиты","equipment":"Снаряжение","consumable":"Расходники","backpack":"Контейнеры","treasure":"Сокровища"}
	order := []string{"weapon","armor","shield","equipment","consumable","backpack","treasure"}
	groups := make(map[string][]string)
	total := 0
	for _, item := range c.Items {
		if item.Category != "equipment" || item.Name == "" { continue }
		group := item.Type
		if groupLabels[group] == "" { group = "equipment" }
		line := item.Name
		if item.Quantity != nil && *item.Quantity > 1 { line = appendLocalizedCharacterSuffix(line, " × "+strconv.Itoa(*item.Quantity)) }
		if state := inventoryStateLabel(item.Equipped); state != "" { line += " · **" + state + "**" }
		if item.Usage != nil { if usage := inventoryUsageLabel(*item.Usage); usage != "" { line += " · " + usage } }
		if item.ContainerID != nil {
			if name := containerNames[*item.ContainerID]; name != "" { line += " · в " + name } else { line += " · внутри контейнера" }
		}
		groups[group] = append(groups[group], line)
		total++
	}
	if total == 0 { return []*discordgo.MessageEmbed{{Title:"Инвентарь · "+c.Identity.Name,Description:"В инвентаре нет предметов или они отсутствуют в последней синхронизации.",Color:0x5865F2}} }
	var embeds []*discordgo.MessageEmbed
	for _, group := range order {
		lines := groups[group]
		for start := 0; start < len(lines); start += 12 {
			end := start + 12
			if end > len(lines) { end = len(lines) }
			embed := &discordgo.MessageEmbed{Title:"Инвентарь · "+c.Identity.Name,Color:0x5865F2}
			embed.Fields = append(embed.Fields,&discordgo.MessageEmbedField{Name:fmt.Sprintf("%s · %d/%d",groupLabels[group],start/12+1,(len(lines)+11)/12),Value:strings.Join(lines[start:end],"\n"),Inline:false})
			embed.Footer = &discordgo.MessageEmbedFooter{Text:fmt.Sprintf("Всего предметов: %d · данные последней синхронизации Foundry",total)}
			embeds=append(embeds,embed)
		}
	}
	return embeds
}

func inventoryStateLabel(equipped *characterEquipped) string {
	if equipped == nil { return "состояние неизвестно" }
	if equipped.InSlot != nil && *equipped.InSlot {
		if equipped.Invested != nil && *equipped.Invested { return "НАДЕТО · ИНВЕСТИРОВАНО" }
		return "НАДЕТО"
	}
	if equipped.CarryType != nil {
		switch *equipped.CarryType {
		case "stowed": return "УБРАНО"
		case "held": return "В РУКАХ"
		case "worn":
			if equipped.Invested != nil && *equipped.Invested { return "НОСИТСЯ · ИНВЕСТИРОВАНО" }
			return "НОСИТСЯ"
		}
	}
	if equipped.Invested != nil && *equipped.Invested { return "ИНВЕСТИРОВАНО" }
	return ""
}

func inventoryUsageLabel(usage string) string {
	switch usage {
	case "held-in-one-hand": return "1 рука"
	case "held-in-two-hands": return "2 руки"
	case "worn": return "носимое"
	case "wornbackpack": return "рюкзак"
	default: return ""
	}
}

func characterSpellsEmbed(c characterDetails) *discordgo.MessageEmbed {
	embeds := characterSpellsEmbeds(c)
	if len(embeds) == 0 { return &discordgo.MessageEmbed{Title:"Заклинания · "+c.Identity.Name,Description:"Заклинания не найдены.",Color:0x5865F2} }
	return embeds[0]
}

func characterSpellsEmbeds(c characterDetails) []*discordgo.MessageEmbed {
	type spellGroup struct { tradition string; rank int; list string }
	groups := make(map[spellGroup][]string)
	keys := make([]spellGroup,0)
	for _, item := range c.Items {
		if item.Type != "spell" || item.Name == "" { continue }
		tradition := "Традиция не указана"
		if item.Tradition != nil && *item.Tradition != "" { tradition = spellTraditionLabel(*item.Tradition) }
		rank := 0
		if item.Level != nil { rank = *item.Level }
		list := ""
		if item.SpellList != nil { list = *item.SpellList }
		key := spellGroup{tradition:tradition,rank:rank,list:list}
		if _,ok:=groups[key]; !ok { keys=append(keys,key) }
		line:=item.Name
		if rank>0 { line+=fmt.Sprintf(" · ранг %d",rank) } else { line+=" · ранг не указан" }
		if item.Prepared!=nil && *item.Prepared { line+=" · подготовлено" }
		groups[key]=append(groups[key],line)
	}
	if len(keys)==0 { return []*discordgo.MessageEmbed{{Title:"Заклинания · "+c.Identity.Name,Description:"В последней синхронизации Foundry заклинания не найдены.",Color:0x5865F2}} }
	sort.Slice(keys,func(i,j int)bool{if keys[i].tradition!=keys[j].tradition{return keys[i].tradition<keys[j].tradition};if keys[i].rank!=keys[j].rank{return keys[i].rank<keys[j].rank};return keys[i].list<keys[j].list})
	var embeds []*discordgo.MessageEmbed
	for _,key:=range keys {
		lines:=groups[key]
		for start:=0;start<len(lines);start+=12 {
			end:=start+12;if end>len(lines){end=len(lines)}
			field:=key.tradition
			if key.rank>0 { field+=fmt.Sprintf(" · ранг %d",key.rank) } else { field+=" · ранг не указан" }
			if key.list!="" { field+=" · "+key.list }
			embed:=&discordgo.MessageEmbed{Title:"Заклинания · "+c.Identity.Name,Color:0x5865F2}
			embed.Fields=append(embed.Fields,&discordgo.MessageEmbedField{Name:field,Value:strings.Join(lines[start:end],"\n"),Inline:false})
			embed.Footer=&discordgo.MessageEmbedFooter{Text:fmt.Sprintf("Всего групп: %d · данные последней синхронизации Foundry",len(keys))}
			embeds=append(embeds,embed)
		}
	}
	return embeds
}

func spellTraditionLabel(value string) string {
	switch strings.ToLower(value) {
	case "arcane": return "Арканная традиция"
	case "divine": return "Божественная традиция"
	case "occult": return "Оккультная традиция"
	case "primal": return "Природная традиция"
	default: return value
	}
}

func characterEmbed(c characterDetails) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: c.Identity.Name,
		Color: 0x5865F2,
	}
	if c.Identity.Image != nil &&
		(strings.HasPrefix(*c.Identity.Image, "https://") || strings.HasPrefix(*c.Identity.Image, "http://")) {
		embed.Image = &discordgo.MessageEmbedImage{URL: *c.Identity.Image}
	}

	addBuildField := func(label string, ref *characterReference) {
		if ref != nil && strings.TrimSpace(ref.Name) != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: label, Value: ref.Name, Inline: true})
		}
	}
	addBuildField("Родословная", c.Build.Ancestry)
	addBuildField("Наследие", c.Build.Heritage)
	addBuildField("Предыстория", c.Build.Background)
	addBuildField("Класс", c.Build.Class)

	if c.Attributes.Dying != nil && c.Attributes.Dying.Current != nil && c.Attributes.Dying.Max != nil && *c.Attributes.Dying.Max > 0 && *c.Attributes.Dying.Current >= *c.Attributes.Dying.Max {
		embed.Title = "💀 МЁРТВ · " + c.Identity.Name
		embed.Color = 0xED4245
	}

	level := "—"
	if c.Progression.Level != nil {
		level = strconv.Itoa(*c.Progression.Level)
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "Уровень", Value: level, Inline: true,
	})
	if c.Attributes.HP != nil {
		hp := intValue(c.Attributes.HP.Current)
		if c.Attributes.HP.Max != nil {
			hp += " / " + strconv.Itoa(*c.Attributes.HP.Max)
		} else {
			hp += " ОЗ"
		}
		if c.Attributes.HP.Temp != nil && *c.Attributes.HP.Temp > 0 {
			hp += fmt.Sprintf(" (+%d временных)", *c.Attributes.HP.Temp)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Здоровье", Value: hp, Inline: true,
		})
	}

	xp := intValue(c.Progression.Experience.Value)
	if c.Progression.Experience.Max != nil {
		xp += " / " + strconv.Itoa(*c.Progression.Experience.Max)
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "Опыт", Value: xp, Inline: true,
	})

	if c.Build.KeyAbility != nil {
		keyAbilityNames := map[string]string{"str": "Сила", "dex": "Ловкость", "con": "Телосложение", "int": "Интеллект", "wis": "Мудрость", "cha": "Харизма"}
		if label := keyAbilityNames[strings.ToLower(*c.Build.KeyAbility)]; label != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Ключевая характеристика", Value: label, Inline: true})
		}
	}

	abilityOrder := []string{"str", "dex", "con", "int", "wis", "cha"}
	abilityNames := map[string]string{
		"str": "СИЛ",
		"dex": "ЛОВ",
		"con": "ТЕЛ",
		"int": "ИНТ",
		"wis": "МДР",
		"cha": "ХАР",
	}
	abilities := make([]string, 0, len(c.Statistics.Abilities))
	for _, key := range abilityOrder {
		if value, ok := c.Statistics.Abilities[key]; ok {
			line := abilityNames[key] + ": "
			if value.Score != nil {
				line += strconv.Itoa(*value.Score)
			} else {
				line += "—"
			}
			if value.Modifier != nil {
				line += fmt.Sprintf(" (%+d)", *value.Modifier)
			}
			abilities = append(abilities, line)
		}
	}
	if len(abilities) == 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Характеристики", Value: "Значения не сохранены в Actor snapshot", Inline: false})
	} else {
		addCharacterListField(embed, "Характеристики", abilities)
	}

	if c.Resources.HeroPoints != nil {
		hero := intValue(c.Resources.HeroPoints.Value)
		if c.Resources.HeroPoints.Max != nil {
			hero += " / " + strconv.Itoa(*c.Resources.HeroPoints.Max)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Очки героизма", Value: hero, Inline: true,
		})
	}

	classAbilities := make([]string, 0)
	ancestryAbilities := make([]string, 0)
	actions := make([]string, 0)
	classFeatures := make([]string, 0)
	otherAbilities := make([]string, 0)
	items := make([]string, 0)
	seenClassFeatures := make(map[string]struct{})

	if c.Build.ClassProficiencies != nil && c.Progression.Level != nil {
		for _, feature := range c.Build.ClassProficiencies.Features {
			if feature.Name == "" || feature.Level == nil || *feature.Level > *c.Progression.Level {
				continue
			}
			appendUniqueCharacterEntry(&classFeatures, seenClassFeatures, feature.Name)
		}
	}

	for _, item := range c.Items {
		if item.Name == "" {
			continue
		}
		switch {
		case item.Category == "equipment":
			line := item.Name
			if item.Quantity != nil && *item.Quantity > 1 {
				line = appendLocalizedCharacterSuffix(line, " × "+strconv.Itoa(*item.Quantity))
			}
			items = append(items, line)
		case item.Type == "action":
			actions = append(actions, actionDisplayName(item))
		case item.Type == "feat":
			switch stringValue(item.SystemCategory) {
			case "class":
				classAbilities = append(classAbilities, item.Name)
			case "ancestry", "ancestryfeature", "heritage":
				ancestryAbilities = append(ancestryAbilities, item.Name)
			case "classfeature":
				appendUniqueCharacterEntry(&classFeatures, seenClassFeatures, item.Name)
			default:
				if _, isClassFeature := seenClassFeatures[characterEntryKey(item.Name)]; !isClassFeature {
					otherAbilities = append(otherAbilities, item.Name)
				}
			}
		}
	}
	addCharacterListField(embed, "Способности класса", classAbilities)
	addCharacterListField(embed, "Способности родословной", ancestryAbilities)
	addCharacterListField(embed, "Действия", actions)
	addCharacterListField(embed, "Особенности класса", classFeatures)
	addCharacterListField(embed, "Другие способности", otherAbilities)

	skills := characterSkillLines(c.Skills.Standard, c.Skills.Lore)
	addCharacterListField(embed, "Навыки", skills)
	addCharacterListField(embed, "Предметы", items)

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Данные персонажа из последней синхронизации Foundry",
	}
	return embed
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
	const maxEntries = 10
	if len(entries) > maxEntries {
		remaining := len(entries) - maxEntries
		entries = append(append([]string(nil), entries[:maxEntries]...), fmt.Sprintf("… и ещё %d", remaining))
	}
	value := strings.Join(entries, "\n")
	if len([]rune(value)) > 1024 {
		value = string([]rune(value)[:1023]) + "…"
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: name, Value: value, Inline: false,
	})
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
	names := map[string]string{
		"acrobatics":   "Акробатика",
		"arcana":       "Аркана",
		"athletics":    "Атлетика",
		"crafting":     "Ремесло",
		"deception":    "Обман",
		"diplomacy":    "Дипломатия",
		"intimidation": "Запугивание",
		"medicine":     "Медицина",
		"nature":       "Природа",
		"occultism":    "Оккультизм",
		"performance":  "Выступление",
		"religion":     "Религия",
		"society":      "Общество",
		"stealth":      "Скрытность",
		"survival":     "Выживание",
		"thievery":     "Воровство",
	}
	keys := make([]string, 0, len(standard))
	for key := range standard {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(standard)+len(lore))
	for _, key := range keys {
		skill := standard[key]
		label := proficiencyLabel(skill.Rank)
		if label == "" {
			continue
		}
		name := names[key]
		if name == "" {
			name = humanizeSkillKey(key)
		}
		lines = append(lines, fmt.Sprintf("%s — %s", name, label))
	}

	loreKeys := make([]string, 0, len(lore))
	for key := range lore {
		loreKeys = append(loreKeys, key)
	}
	sort.Strings(loreKeys)
	for _, key := range loreKeys {
		label := proficiencyLabel(lore[key].Rank)
		if label == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %s", loreSkillName(key), label))
	}
	return lines
}

func proficiencyLabel(rank *int) string {
	if rank == nil {
		return ""
	}
	switch *rank {
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

func (b *bot) monitorIntegrations(s *discordgo.Session) {
	b.checkIntegrationStatus(s)
	ticker := time.NewTicker(b.healthInterval)
	defer ticker.Stop()
	for range ticker.C {
		b.checkIntegrationStatus(s)
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
	resp, err := b.client.Do(req)
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
