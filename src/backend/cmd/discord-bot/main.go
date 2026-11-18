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
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	RegisterPath = "/api/internal/discord/register"
	ActorsPath = "/api/internal/discord/actors"
	CharactersPath = "/api/internal/discord/characters"
	PreviewPath = "/api/internal/discord/link/preview"
	LinkPath = "/api/internal/discord/link"
	GrantRolePath = "/api/internal/discord/roles/grant"
	MapRolePath = "/api/internal/discord/roles/map"
)

type bot struct {
	apiURL, apiToken, guildID string
	client *http.Client
	mu sync.Mutex
	pending map[string]pendingLink
}
type pendingLink struct { actorUUID, characterID, discordID, guildID string; expiresAt time.Time }
type identity struct {
	GuildID string `json:"guild_id"`
	DiscordUserID string `json:"discord_user_id"`
	DiscordRoleIDs []string `json:"discord_role_ids"`
}
type registerPayload struct {
	identity
	Username string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}
type linkPayload struct {
	identity
	ActorUUID string `json:"actor_uuid"`
	CharacterID string `json:"character_id"`
}
type actor struct { ActorUUID string `json:"actor_uuid"`; Name string `json:"name"` }
type character struct { ID string `json:"id"`; Name string `json:"name"`; Status string `json:"status"` }
type linkPreview struct {
	ActorUUID string `json:"actor_uuid"`
	ActorName string `json:"actor_name"`
	CharacterID string `json:"character_id"`
	CharacterName string `json:"character_name"`
}

func main() {
	token, appID := requiredEnv("DISCORD_BOT_TOKEN"), requiredEnv("DISCORD_APPLICATION_ID")
	guildID, apiToken := requiredEnv("DISCORD_GUILD_ID"), requiredEnv("DISCORD_BOT_API_TOKEN")
	b := &bot{apiURL: strings.TrimRight(requiredEnv("BACKEND_INTERNAL_URL"), "/"), apiToken: apiToken,
		guildID: guildID, client: &http.Client{Timeout: 8*time.Second}, pending: make(map[string]pendingLink)}
	if len(apiToken) < 32 { log.Fatal("DISCORD_BOT_API_TOKEN must contain at least 32 characters") }
	s, err := discordgo.New("Bot " + token); if err != nil { log.Fatal(err) }
	s.Identify.Intents = discordgo.IntentsGuilds
	s.AddHandler(b.handle)
	if err := s.Open(); err != nil { log.Fatal(err) }
	defer s.Close()
	if err := registerCommands(s, appID, guildID); err != nil { log.Fatal(err) }
	log.Printf("FoundryMarches bot connected; guild=%s", guildID)
	select {}
}

func registerCommands(s *discordgo.Session, appID, guildID string) error {
	commands := []*discordgo.ApplicationCommand{
		{Name:"register", Description:"Зарегистрироваться в FoundryMarches"},
		{Name:"foundry-actors", Description:"Показать несвязанных персонажей Foundry"},
		{Name:"characters", Description:"Показать персонажей приложения и их ID"},
		{Name:"character-link", Description:"Подготовить привязку Actor к персонажу",
			Options:[]*discordgo.ApplicationCommandOption{
				{Name:"actor_uuid", Description:"UUID из /foundry-actors", Type:discordgo.ApplicationCommandOptionString, Required:true},
				{Name:"character_id", Description:"ID из /characters", Type:discordgo.ApplicationCommandOptionString, Required:true}}},
		{Name:"role-grant", Description:"Назначить роль зарегистрированному участнику",
			Options:[]*discordgo.ApplicationCommandOption{
				{Name:"discord_user_id", Description:"Discord ID участника", Type:discordgo.ApplicationCommandOptionString, Required:true},
				{Name:"role_key", Description:"Роль приложения", Type:discordgo.ApplicationCommandOptionString, Required:true,
					Choices:[]*discordgo.ApplicationCommandOptionChoice{{Name:"Игрок",Value:"player"},{Name:"Мастер",Value:"gm"}}}}},
		{Name:"role-map", Description:"Сопоставить Discord-роль с ролью приложения",
			Options:[]*discordgo.ApplicationCommandOption{
				{Name:"discord_role_id", Description:"ID Discord-роли", Type:discordgo.ApplicationCommandOptionString, Required:true},
				{Name:"role_key", Description:"Роль приложения", Type:discordgo.ApplicationCommandOptionString, Required:true,
					Choices:[]*discordgo.ApplicationCommandOptionChoice{{Name:"Игрок",Value:"player"},{Name:"Мастер",Value:"gm"}}}}},
	}
	for _, cmd := range commands { if _,err:=s.ApplicationCommandCreate(appID,guildID,cmd); err!=nil { return fmt.Errorf("register %s: %w",cmd.Name,err) } }
	return nil
}

func (b *bot) handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID != b.guildID { b.reply(s,i,"Бот работает только на настроенном сервере.",nil); return }
	if i.Type == discordgo.InteractionMessageComponent { b.handleButton(s,i); return }
	if i.Type != discordgo.InteractionApplicationCommand { return }
	if err:=s.InteractionRespond(i.Interaction,&discordgo.InteractionResponse{Type:discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data:&discordgo.InteractionResponseData{Flags:discordgo.MessageFlagsEphemeral}}); err!=nil { return }
	u:=interactionUser(i); if u==nil { b.edit(s,i,"Не удалось определить пользователя.",nil); return }
	id:=b.identity(i,u)
	switch i.ApplicationCommandData().Name {
	case "register":
		var out struct { UserID string `json:"user_id"` }
		err:=b.post(RegisterPath,registerPayload{identity:id,Username:u.Username,AvatarURL:u.AvatarURL("256")},&out)
		if err!=nil { b.edit(s,i,"Регистрация не выполнена: "+err.Error(),nil); return }
		b.edit(s,i,"Регистрация завершена. Назначена базовая роль player. Внутренний ID: "+out.UserID,nil)
	case "foundry-actors":
		var out struct { Actors []actor `json:"actors"` }
		if err:=b.post(ActorsPath,id,&out); err!=nil { b.edit(s,i,"Ошибка: "+err.Error(),nil); return }
		lines:=[]string{}; for _,a:=range out.Actors { lines=append(lines,fmt.Sprintf("• %s — %s",a.Name,a.ActorUUID)) }
		if len(lines)==0 { b.edit(s,i,"Несвязанных Actor типа character нет. Сначала синхронизируйте Foundry.",nil) } else { b.edit(s,i,"Несвязанные Actor (до 100):\n"+strings.Join(lines,"\n"),nil) }
	case "characters":
		var out struct { Characters []character `json:"characters"` }
		if err:=b.post(CharactersPath,id,&out); err!=nil { b.edit(s,i,"Ошибка: "+err.Error(),nil); return }
		lines:=[]string{}; for _,c:=range out.Characters { lines=append(lines,fmt.Sprintf("• %s [%s] — %s",c.Name,c.Status,c.ID)) }
		if len(lines)==0 { b.edit(s,i,"Персонажей приложения пока нет. Foundry sync не создаёт их автоматически.",nil) } else { b.edit(s,i,"Персонажи приложения (до 200):\n"+strings.Join(lines,"\n"),nil) }
	case "character-link":
		o:=commandOptions(i.ApplicationCommandData().Options)
		p:=linkPayload{identity:id,ActorUUID:o["actor_uuid"],CharacterID:o["character_id"]}
		var preview linkPreview
		if err:=b.post(PreviewPath,p,&preview); err!=nil { b.edit(s,i,"Нельзя подготовить привязку: "+err.Error(),nil); return }
		key,err:=randomKey(); if err!=nil { b.edit(s,i,"Не удалось создать подтверждение.",nil); return }
		b.mu.Lock(); b.pending[key]=pendingLink{p.ActorUUID,p.CharacterID,u.ID,i.GuildID,time.Now().Add(10*time.Minute)}; b.mu.Unlock()
		components:=[]discordgo.MessageComponent{discordgo.ActionsRow{Components:[]discordgo.MessageComponent{
			discordgo.Button{Label:"Подтвердить привязку",Style:discordgo.SuccessButton,CustomID:"fm-link:"+key}}}}
		b.edit(s,i,fmt.Sprintf("Проверьте привязку:\nFoundry: %s (%s)\nПерсонаж: %s (%s)\nВладелец персонажа не изменится.",preview.ActorName,preview.ActorUUID,preview.CharacterName,preview.CharacterID),components)
	case "role-grant":
		o:=commandOptions(i.ApplicationCommandData().Options)
		payload:=struct { identity; Target string `json:"target_discord_user_id"`; Role string `json:"role_key"` }{id,o["discord_user_id"],o["role_key"]}
		if err:=b.post(GrantRolePath,payload,nil); err!=nil { b.edit(s,i,"Не удалось назначить роль: "+err.Error(),nil) } else { b.edit(s,i,"Роль назначена.",nil) }
	case "role-map":
		o:=commandOptions(i.ApplicationCommandData().Options)
		payload:=struct { identity; RoleID string `json:"discord_role_id"`; Role string `json:"role_key"` }{id,o["discord_role_id"],o["role_key"]}
		if err:=b.post(MapRolePath,payload,nil); err!=nil { b.edit(s,i,"Не удалось сопоставить роль: "+err.Error(),nil) } else { b.edit(s,i,"Сопоставление сохранено.",nil) }
	default: b.edit(s,i,"Неизвестная команда.",nil)
	}
}

func (b *bot) handleButton(s *discordgo.Session,i *discordgo.InteractionCreate) {
	data:=i.MessageComponentData(); if !strings.HasPrefix(data.CustomID,"fm-link:") { return }
	u:=interactionUser(i); if u==nil { b.reply(s,i,"Не удалось определить пользователя.",nil); return }
	key:=strings.TrimPrefix(data.CustomID,"fm-link:")
	b.mu.Lock(); p,ok:=b.pending[key]; if ok { delete(b.pending,key) }; b.mu.Unlock()
	if !ok || time.Now().After(p.expiresAt) || p.discordID!=u.ID || p.guildID!=i.GuildID {
		b.reply(s,i,"Подтверждение недействительно, истекло или принадлежит другому участнику. Повторите /character-link.",nil); return
	}
	if err:=s.InteractionRespond(i.Interaction,&discordgo.InteractionResponse{Type:discordgo.InteractionResponseDeferredMessageUpdate}); err!=nil { return }
	err:=b.post(LinkPath,linkPayload{identity:b.identity(i,u),ActorUUID:p.actorUUID,CharacterID:p.characterID},nil)
	if err!=nil { b.edit(s,i,"Привязка не выполнена: "+err.Error(),[]discordgo.MessageComponent{}) } else { b.edit(s,i,"Привязка выполнена. Владение персонажем не менялось.",[]discordgo.MessageComponent{}) }
}

func (b *bot) identity(i *discordgo.InteractionCreate,u *discordgo.User) identity {
	roles:=[]string{}; if i.Member!=nil { roles=append(roles,i.Member.Roles...) }
	return identity{GuildID:i.GuildID,DiscordUserID:u.ID,DiscordRoleIDs:roles}
}
func interactionUser(i *discordgo.InteractionCreate)*discordgo.User { if i.Member!=nil && i.Member.User!=nil{return i.Member.User}; return i.User }
func commandOptions(options []*discordgo.ApplicationCommandInteractionDataOption)map[string]string {
	out:=map[string]string{}; for _,o:=range options { out[o.Name]=o.StringValue() }; return out
}
func (b *bot) post(path string,payload,target any)error {
	body,err:=json.Marshal(payload); if err!=nil{return err}
	req,err:=http.NewRequest(http.MethodPost,b.apiURL+path,bytes.NewReader(body)); if err!=nil{return err}
	req.Header.Set("Authorization","Bearer "+b.apiToken); req.Header.Set("Content-Type","application/json")
	resp,err:=b.client.Do(req); if err!=nil{return err}; defer resp.Body.Close()
	data,err:=io.ReadAll(io.LimitReader(resp.Body,1<<20)); if err!=nil{return err}
	if resp.StatusCode<200||resp.StatusCode>=300 { var apiErr struct{Error string `json:"error"`}; _=json.Unmarshal(data,&apiErr); if apiErr.Error==""{apiErr.Error=http.StatusText(resp.StatusCode)}; return errors.New(apiErr.Error) }
	if target!=nil&&len(data)>0{return json.Unmarshal(data,target)}; return nil
}
func (b *bot) reply(s *discordgo.Session,i *discordgo.InteractionCreate,content string,components []discordgo.MessageComponent) {
	_,err:=s.InteractionResponse(i.Interaction,&discordgo.InteractionResponse{Type:discordgo.InteractionResponseChannelMessageWithSource,
		Data:&discordgo.InteractionResponseData{Content:content,Flags:discordgo.MessageFlagsEphemeral,Components:components}})
	if err!=nil{log.Printf("Discord response: %v",err)}
}
func (b *bot) edit(s *discordgo.Session,i *discordgo.InteractionCreate,content string,components []discordgo.MessageComponent) {
	_,err:=s.InteractionResponseEdit(i.Interaction,&discordgo.WebhookEdit{Content:&content,Components:&components})
	if err!=nil{log.Printf("Discord response edit: %v",err)}
}
func randomKey()(string,error){var raw [12]byte;if _,err:=rand.Read(raw[:]);err!=nil{return "",err};return hex.EncodeToString(raw[:]),nil}
func requiredEnv(key string)string{v:=strings.TrimSpace(os.Getenv(key));if v==""{log.Fatalf("required environment variable %s is not set",key)};return v}
