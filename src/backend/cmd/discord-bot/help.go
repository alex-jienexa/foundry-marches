package main

import "github.com/bwmarrin/discordgo"

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
		embed.Description = "Ты ещё не зарегистрирован в приложении. Сначала создай аккаунт, после этого сможешь самостоятельно привязать персонажа."
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "1. Регистрация", Value: "Выполни команду `/register`. Бот создаст аккаунт и выдаст доступ к каналам кампании.", Inline: false},
			{Name: "2. Подготовка персонажа", Value: "Убедись, что твой лист персонажа уже есть в Foundry и мир синхронизирован с приложением.", Inline: false},
			{Name: "3. Привязка", Value: "Выполни `/character-connect` и выбери своего персонажа из списка. Он сразу займёт один из двух активных слотов.", Inline: false},
		}
	case !status.Linked:
		embed.Description = "Аккаунт зарегистрирован, но персонаж Foundry пока не привязан."
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Как привязать персонажа", Value: "Выполни `/character-connect`, выбери своего Actor из Foundry — приложение автоматически создаст персонажа и свяжет его с твоим аккаунтом.", Inline: false},
			{Name: "Активные слоты", Value: "Одновременно можно иметь до двух активных персонажей. Новая привязка сразу занимает один слот.", Inline: false},
			{Name: "Если выбрал не тот лист", Value: "Мастер увидит привязку в мастер-чате и сможет отменить её. При отмене слот освободится.", Inline: false},
		}
	default:
		embed.Description = "Твой аккаунт зарегистрирован, и у тебя есть привязанный персонаж Foundry."
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Мои персонажи", Value: "`/my-characters` — список персонажей, портреты, уровень, опыт, родословная, класс и свободные слоты.", Inline: false},
			{Name: "Полезные команды", Value: "`/my-character` — открыть карточку персонажа
`/character-connect` — привязать ещё одного персонажа (не более двух активных)
`/character name:<имя>` — найти персонажа по имени
`/character-inventory name:<имя>` — посмотреть инвентарь
`/character-spells name:<имя>` — посмотреть заклинания
`/help` — открыть эту справку", Inline: false},
			{Name: "Изменения листа", Value: "Данные персонажей обновляются из Foundry. Если изменил характеристики, экипировку или заклинания, дождись синхронизации.", Inline: false},
		}
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Справка доступна всем участникам сервера"}
	return embed
}
