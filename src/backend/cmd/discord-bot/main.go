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
	CharacterSearchPath   = "/api/internal/discord/characters/search"
	CharacterDetailPath   = "/api/internal/discord/characters/detail"
	IntegrationStatusPath = "/api/internal/discord/integration-status"
)

type bot struct {
	apiURL, apiToken, guildID string
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
	actorUUID, characterID, discordID, guildID string
	expiresAt                                  time.Time
}
type selectionFlow struct {
	mode, stage, discordID, guildID, actorUUID string
	actors                                     []actor
	characters                                 []character
	page                                       int
	expiresAt                                  time.Time
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
	ActorUUID   string `json:"actor_uuid"`
	CharacterID string `json:"character_id"`
}
type actor struct {
	ActorUUID string `json:"actor_uuid"`
	Name      string `json:"name"`
}
type character struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
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
		ClassProficiencies *classProficiencyDetails `json:"classProficiencies"`
	} `json:"build"`
	Attributes struct {
		HP *struct {
			Current *int `json:"current"`
			Temp    *int `json:"temp"`
			Max     *int `json:"max"`
		} `json:"hp"`
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
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Category       string  `json:"category"`
	SystemCategory *string `json:"systemCategory"`
	ActionType     *string `json:"actionType"`
	Quantity       *int    `json:"quantity"`
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
	healthInterval := 60 * time.Second
	if raw := strings.TrimSpace(os.Getenv("DISCORD_HEALTH_INTERVAL_SECONDS")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < 15 || seconds > 3600 {
			log.Fatal("DISCORD_HEALTH_INTERVAL_SECONDS must be between 15 and 3600")
		}
		healthInterval = time.Duration(seconds) * time.Second
	}
	b := &bot{apiURL: strings.TrimRight(requiredEnv("BACKEND_INTERNAL_URL"), "/"), apiToken: apiToken,
		guildID: guildID, adminChannelID: adminChannelID, healthInterval: healthInterval,
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
		{Name: "characters", Description: "Показать персонажей приложения и их ID"},
		{Name: "character-link", Description: "Выбрать Foundry Actor и персонажа приложения для привязки"},
		{Name: "character-draft", Description: "Создать черновик персонажа из выбранного Foundry Actor"},
		{Name: "character", Description: "Найти своего персонажа (GM видит всех) и показать данные Foundry", Options: []*discordgo.ApplicationCommandOption{{Name: "name", Description: "Имя или часть имени персонажа", Type: discordgo.ApplicationCommandOptionString, Required: true}}},
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
		b.edit(s, i, "Регистрация завершена. Назначена базовая роль player. Внутренний ID: "+out.UserID, nil)
	case "foundry-actors":
		var out struct {
			Actors []actor `json:"actors"`
		}
		if err := b.post(ActorsPath, id, &out); err != nil {
			b.edit(s, i, "Ошибка: "+err.Error(), nil)
			return
		}
		lines := []string{}
		for _, a := range out.Actors {
			lines = append(lines, fmt.Sprintf("• %s — %s", a.Name, a.ActorUUID))
		}
		if len(lines) == 0 {
			b.edit(s, i, "Несвязанных Actor типа character нет. Сначала синхронизируйте Foundry.", nil)
		} else {
			b.edit(s, i, truncateMessage("Несвязанные Actor (до 100):\n"+strings.Join(lines, "\n")), nil)
		}
	case "characters":
		var out struct {
			Characters []character `json:"characters"`
		}
		if err := b.post(CharactersPath, id, &out); err != nil {
			b.edit(s, i, "Ошибка: "+err.Error(), nil)
			return
		}
		lines := []string{}
		for _, c := range out.Characters {
			lines = append(lines, fmt.Sprintf("• %s [%s] — %s", c.Name, c.Status, c.ID))
		}
		if len(lines) == 0 {
			b.edit(s, i, "Персонажей приложения пока нет. Foundry sync не создаёт их автоматически.", nil)
		} else {
			b.edit(s, i, truncateMessage("Персонажи приложения (до 200):\n"+strings.Join(lines, "\n")), nil)
		}
	case "character-link":
		b.startSelection(s, i, id, u, "link")
	case "character-draft":
		b.startSelection(s, i, id, u, "draft")
	case "character":
		o := commandOptions(i.ApplicationCommandData().Options)
		b.startCharacterLookup(s, i, id, u, o["name"])
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
			b.edit(s, i, "Роль назначена.", nil)
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
			b.edit(s, i, "Сопоставление сохранено.", nil)
		}
	default:
		b.edit(s, i, "Неизвестная команда.", nil)
	}
}

func (b *bot) startSelection(s *discordgo.Session, i *discordgo.InteractionCreate, id identity, u *discordgo.User, mode string) {
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
		actors: actorResult.Actors, expiresAt: time.Now().Add(10 * time.Minute)}
	if mode == "link" {
		var characterResult struct {
			Characters []character `json:"characters"`
		}
		if err := b.post(CharactersPath, id, &characterResult); err != nil {
			b.edit(s, i, "Не удалось получить список персонажей: "+err.Error(), nil)
			return
		}
		if len(characterResult.Characters) == 0 {
			b.edit(s, i, "Персонажей приложения пока нет. Сначала используй /character-draft, чтобы создать черновик из Foundry Actor.", nil)
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
		label = "Выбери Foundry Actor для привязки:"
	}
	b.edit(s, i, label, components)
}

func (b *bot) startCharacterLookup(s *discordgo.Session, i *discordgo.InteractionCreate, id identity, u *discordgo.User, query string) {
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
	b.edit(s, i, fmt.Sprintf("Найдено персонажей: %d. Выбери нужного:", len(result.Actors)), b.actorSelectionComponents(key, flow))
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
		if flow.mode == "view" {
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
			b.editEmbed(s, i, characterEmbed(details))
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
			b.edit(s, i, fmt.Sprintf("Создан черновик персонажа **%s**. ID: `%s`. Статус: draft. Владелец не назначен. Следующий шаг — /character-link для привязки.", created.Name, created.ID), nil)
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
		payload := linkPayload{identity: b.identity(i, u), ActorUUID: actorUUID, CharacterID: chosen.ID}
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
		b.pending[confirmKey] = pendingLink{payload.ActorUUID, payload.CharacterID, u.ID, i.GuildID, time.Now().Add(10 * time.Minute)}
		b.mu.Unlock()
		components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "Подтвердить привязку", Style: discordgo.SuccessButton, CustomID: "fm-link:" + confirmKey}}}}
		b.edit(s, i, fmt.Sprintf("Проверьте привязку:\nFoundry: %s (%s)\nПерсонаж: %s (%s)\nВладелец персонажа не изменится.", preview.ActorName, preview.ActorUUID, preview.CharacterName, preview.CharacterID), components)
	}
}

func (b *bot) actorSelectionComponents(key string, flow *selectionFlow) []discordgo.MessageComponent {
	start, end := pageBounds(len(flow.actors), flow.page)
	minValues := 1
	options := make([]discordgo.SelectMenuOption, 0, end-start)
	for _, item := range flow.actors[start:end] {
		options = append(options, discordgo.SelectMenuOption{Label: truncateLabel(item.Name), Value: item.ActorUUID, Description: truncateLabel(item.ActorUUID)})
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
		options = append(options, discordgo.SelectMenuOption{Label: truncateLabel(item.Name + " [" + item.Status + "]"), Value: item.ID, Description: truncateLabel(item.ID)})
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
	err := b.post(LinkPath, linkPayload{identity: b.identity(i, u), ActorUUID: p.actorUUID, CharacterID: p.characterID}, nil)
	if err != nil {
		b.edit(s, i, "Привязка не выполнена: "+err.Error(), []discordgo.MessageComponent{})
	} else {
		b.edit(s, i, "Привязка выполнена. Владение персонажем не менялось.", []discordgo.MessageComponent{})
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

func characterEmbed(c characterDetails) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: c.Identity.Name,
		Color: 0x5865F2,
	}
	if c.Identity.Image != nil &&
		(strings.HasPrefix(*c.Identity.Image, "https://") || strings.HasPrefix(*c.Identity.Image, "http://")) {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: *c.Identity.Image}
	}

	level := "—"
	if c.Progression.Level != nil {
		level = strconv.Itoa(*c.Progression.Level)
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "Уровень", Value: level, Inline: true,
	})
	if c.Attributes.HP != nil {
		hp := fmt.Sprintf("%s / %s", intValue(c.Attributes.HP.Current), intValue(c.Attributes.HP.Max))
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
	addCharacterListField(embed, "Характеристики", abilities)

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
				line += " × " + strconv.Itoa(*item.Quantity)
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
		Text: "Источник: последний сохранённый снимок Foundry · " + c.Identity.UUID,
	}
	return embed
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
		return name + " — " + label
	}
	return name
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
		out[o.Name] = o.StringValue()
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
