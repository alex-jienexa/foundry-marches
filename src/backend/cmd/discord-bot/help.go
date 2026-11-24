package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type helpStatus struct {
	Registered bool `json:"registered"`
	Linked     bool `json:"linked"`
}

func helpEmbed(status helpStatus) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "FoundryMarches — справка",
		Color: 0x5865F2,
	}
	switch {
	case !status.Registered:
		embed.Description = "Ты ещё не зарегистрирован в приложении. Начни с регистрации, затем сообщи мастеру, какой лист персонажа Foundry нужно привязать к твоему аккаунту."
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "1. Регистрация", Value: "Выполни команду `/register`. Бот создаст аккаунт и выдаст доступ к каналам кампании.", Inline: false},
			{Name: "2. Подготовка листа", Value: "Открой свой лист персонажа в Foundry и убедись, что мастер может определить, что это твой персонаж. Для самостоятельной привязки мы отдельно настроим подтверждение связи Foundry и Discord.", Inline: false},
			{Name: "3. Привязка", Value: "После регистрации обратись к мастеру: сейчас привязку выполняет мастер через `/character-link`, чтобы случайно не назначить чужой лист.", Inline: false},
		}
	case !status.Linked:
		embed.Description = "Аккаунт зарегистрирован, но подтверждённый персонаж пока не привязан."
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Как привязать персонажа", Value: "Открой лист своего персонажа в Foundry и обратись к мастеру для привязки. Сейчас мастер выполняет её через `/character-link`. Самостоятельная привязка одной командой появится следующим улучшением.", Inline: false},
			{Name: "Важно", Value: "Не создавай отдельный черновик самостоятельно — мастер свяжет лист Foundry с записью приложения.", Inline: false},
		}
	default:
		embed.Description = "Твой аккаунт зарегистрирован, и у тебя есть привязанный персонаж Foundry."
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Персонаж", Value: "Открой свою карточку командой `/my-character`.", Inline: false},
			{Name: "Полезные команды", Value: fmt.Sprintf(
				"`/my-character` — открыть своего персонажа\n`/character name:<имя>` — найти персонажа по имени\n`/character-inventory name:<имя>` — посмотреть инвентарь\n`/character-spells name:<имя>` — посмотреть заклинания\n`/help` — открыть эту справку",
			), Inline: false},
			{Name: "Изменения листа", Value: "Данные персонажа обновляются из Foundry. Если изменил характеристики, экипировку или заклинания, дождись синхронизации.", Inline: false},
		}
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Справка доступна всем участникам сервера"}
	return embed
}
