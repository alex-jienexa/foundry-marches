package main

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandOptionsReadDiscordUserMention(t *testing.T) {
	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "owner", Type: discordgo.ApplicationCommandOptionUser, Value: "123456789012345678"},
	}
	got := commandOptions(options)
	if got["owner"] != "123456789012345678" {
		t.Fatalf("owner option = %q, want Discord user ID", got["owner"])
	}
}

func TestCommandOptionsReadStringValue(t *testing.T) {
	options := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: "Андрей"},
	}
	got := commandOptions(options)
	if got["name"] != "Андрей" {
		t.Fatalf("name option = %q, want Андрей", got["name"])
	}
}

func TestDirectoryEmbedUsesReadableStatusesAndCounts(t *testing.T) {
	embed := directoryEmbed("Персонажи", "Список", []string{"Андрей · " + characterStatusLabel("draft"), "Вихорь · " + characterStatusLabel("active")}, 1)
	if embed.Title != "Персонажи" || len(embed.Fields) != 1 {
		t.Fatalf("unexpected directory embed: %+v", embed)
	}
	if !strings.Contains(embed.Fields[0].Value, "Андрей · черновик") {
		t.Fatalf("directory field = %q, missing localized status", embed.Fields[0].Value)
	}
	if embed.Footer == nil || embed.Footer.Text != "Показано 1 из 2 · ещё 1 не показано" {
		t.Fatalf("directory footer = %+v", embed.Footer)
	}
}

func TestPageBounds(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		page      int
		wantStart int
		wantEnd   int
	}{
		{name: "first page", total: 60, page: 0, wantStart: 0, wantEnd: 25},
		{name: "middle page", total: 60, page: 1, wantStart: 25, wantEnd: 50},
		{name: "last partial page", total: 60, page: 2, wantStart: 50, wantEnd: 60},
		{name: "page is clamped", total: 26, page: 99, wantStart: 25, wantEnd: 26},
		{name: "negative page is clamped", total: 26, page: -1, wantStart: 0, wantEnd: 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := pageBounds(tt.total, tt.page)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("pageBounds(%d, %d) = (%d, %d), want (%d, %d)", tt.total, tt.page, gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestTruncateLabel(t *testing.T) {
	short := "Андрей"
	if got := truncateLabel(short); got != short {
		t.Fatalf("truncateLabel(%q) = %q", short, got)
	}
	long := "персонаж"
	for range 120 {
		long += "я"
	}
	if got := truncateLabel(long); len([]rune(got)) > 100 {
		t.Fatalf("truncateLabel returned %d runes, want <= 100", len([]rune(got)))
	}
}

func TestCharacterEmbedUsesDecodedFields(t *testing.T) {
	level := 2
	currentHP, maxHP := 18, 24
	c := characterDetails{}
	c.Identity.Name = "Тестовый герой"
	c.Identity.UUID = "Actor.test"
	c.Progression.Level = &level
	c.Attributes.HP = &struct {
		Current *int `json:"current"`
		Temp    *int `json:"temp"`
		Max     *int `json:"max"`
	}{Current: &currentHP, Max: &maxHP}

	embed := characterEmbed(c)
	if embed.Title != "Тестовый герой" {
		t.Fatalf("embed title = %q", embed.Title)
	}
	if len(embed.Fields) < 2 {
		t.Fatalf("embed has %d fields, want at least level and HP", len(embed.Fields))
	}
	if embed.Footer == nil || embed.Footer.Text == "" {
		t.Fatal("embed footer must identify the Foundry snapshot")
	}
}

func TestIntegrationStatusEmbedMarksUnavailableComponents(t *testing.T) {
	embed := integrationStatusEmbed(integrationStatus{
		Database: healthComponent{OK: true, Detail: "db ok"},
		Relay:    healthComponent{OK: false, Detail: "relay timeout"},
		Foundry:  healthComponent{OK: true, Detail: "world connected"},
	})
	if embed.Color != 0xED4245 {
		t.Fatalf("embed color = %#x, want error color", embed.Color)
	}
	if len(embed.Fields) != 3 {
		t.Fatalf("embed fields = %d, want 3", len(embed.Fields))
	}
}

func TestCharacterEmbedSeparatesAbilitiesActionsAndEquipment(t *testing.T) {
	level := 1
	featureLevel := 1
	futureFeatureLevel := 3
	one, two, three, four, zero := 1, 2, 3, 4, 0
	classCategory := "class"
	ancestryCategory := "ancestry"
	classFeatureCategory := "classfeature"
	generalCategory := "general"
	skillCategory := "skill"
	reaction := "reaction"

	c := characterDetails{}
	c.Identity.Name = "Андрей"
	c.Identity.UUID = "Actor.andrey"
	c.Progression.Level = &level
	c.Build.Ancestry = &characterReference{Name: "Человек"}
	c.Build.Heritage = &characterReference{Name: "Видящий во мраке"}
	c.Build.Background = &characterReference{Name: "Уличный бродяга"}
	c.Build.Class = &characterReference{Name: "Воин"}
	keyAbility := "str"
	c.Build.KeyAbility = &keyAbility
	c.Build.ClassProficiencies = &classProficiencyDetails{Features: []classFeatureReference{
		{Name: "Реагирующий удар", Level: &featureLevel},
		{Name: "Блок щитом", Level: &featureLevel},
		{Name: "Храбрость", Level: &futureFeatureLevel},
	}}
	c.Items = []characterItem{
		{Name: "Двойной разрез / Double Slice", Type: "feat", SystemCategory: &classCategory},
		{Name: "Видящий во мраке / Gloomseer", Type: "feat", SystemCategory: &ancestryCategory},
		{Name: "Реагирующий удар / Reactive Strike", Type: "action", ActionType: &reaction},
		{Name: "Реагирующий удар / Reactive Strike", Type: "feat", SystemCategory: &classFeatureCategory},
		{Name: "Блок щитом / Shield Block", Type: "feat", SystemCategory: &generalCategory},
		{Name: "Сильный носильщик / Hefty Hauler", Type: "feat", SystemCategory: &skillCategory},
		{Name: "Длинный меч / Longsword", Type: "weapon", Category: "equipment"},
		{Name: "Золотые монеты / Gold Pieces", Type: "treasure", Category: "equipment"},
	}
	c.Items[len(c.Items)-1].Quantity = &two
	c.Skills.Standard = map[string]characterSkill{
		"athletics":    {Rank: &one},
		"diplomacy":    {Rank: &zero},
		"intimidation": {Rank: &four},
		"thievery":     {Rank: &three},
	}
	c.Skills.Lore = map[string]characterSkill{
		"lore-warfare": {Rank: &two},
	}

	embed := characterEmbed(c)
	fields := make(map[string]string, len(embed.Fields))
	for _, field := range embed.Fields {
		fields[field.Name] = field.Value
	}
	for _, shown := range []string{"Родословная", "Наследие", "Предыстория", "Класс"} {
		if _, exists := fields[shown]; !exists {
			t.Errorf("fundamental choice %q must be displayed when present", shown)
		}
	}
	checks := []struct {
		field string
		want  string
	}{
		{"Способности класса", "Двойной разрез"},
		{"Способности родословной", "Видящий во мраке"},
		{"Действия", "Реагирующий удар — реакция"},
		{"Особенности класса", "Реагирующий удар"},
		{"Другие способности", "Сильный носильщик"},
		{"Предметы", "Длинный меч"},
		{"Предметы", "Золотые монеты × 2"},
		{"Ключевая характеристика", "Сила"},
		{"Характеристики", "Значения не сохранены"},
	}
	for _, check := range checks {
		if !strings.Contains(fields[check.field], check.want) {
			t.Errorf("field %q = %q, want it to contain %q", check.field, fields[check.field], check.want)
		}
	}
	if strings.Contains(fields["Особенности класса"], "Храбрость") {
		t.Error("class feature above the character's level must be hidden")
	}
	if strings.Contains(fields["Другие способности"], "Блок щитом") {
		t.Errorf("class feature duplicate must not appear under other abilities: %q", fields["Другие способности"])
	}
	if strings.Contains(fields["Предметы"], "Двойной разрез") ||
		strings.Contains(fields["Предметы"], "Реагирующий удар") ||
		strings.Contains(fields["Предметы"], "Блок щитом") {
		t.Errorf("abilities and actions must not appear as equipment: %q", fields["Предметы"])
	}
	if strings.Count(fields["Особенности класса"], "Реагирующий удар") != 1 {
		t.Errorf("class feature must not be duplicated: %q", fields["Особенности класса"])
	}
}

func TestResolveFoundryImageURL(t *testing.T) {
	relative := "systems/pf2e/icons/default-icons/character.svg"
	got := resolveFoundryImageURL(&relative, "https://foundry.example.test")
	if got == nil || *got != "https://foundry.example.test/systems/pf2e/icons/default-icons/character.svg" {
		t.Fatalf("resolved image URL = %v", got)
	}
	absolute := "https://cdn.example.test/portrait.webp"
	got = resolveFoundryImageURL(&absolute, "")
	if got == nil || *got != absolute {
		t.Fatalf("absolute image URL = %v", got)
	}
	got = resolveFoundryImageURL(&relative, "")
	if got != nil {
		t.Fatalf("relative image without public base should be omitted, got %v", *got)
	}
}


func TestCharacterEmbedUsesPublicFoundryPortrait(t *testing.T) {
	path := "assets/westmarches/pc/portrait.jpg"
	resolved := resolveFoundryImageURL(&path, "https://foundry.jiene.ru")
	if resolved == nil || *resolved != "https://foundry.jiene.ru/assets/westmarches/pc/portrait.jpg" {
		t.Fatalf("resolved portrait URL = %v", resolved)
	}
	character := characterDetails{}
	character.Identity.Name = "Портретный персонаж"
	character.Identity.Image = resolved
	embed := characterEmbed(character)
	if embed.Image == nil || embed.Image.URL != *resolved {
		t.Fatalf("embed image = %+v, want resolved public portrait URL", embed.Image)
	}
}

func TestCharacterEmbedDoesNotRenderMissingHPMaximumAsDash(t *testing.T) {
	current := 19
	c := characterDetails{}
	c.Identity.Name = "Андрей"
	c.Attributes.HP = &struct {
		Current *int `json:"current"`
		Temp    *int `json:"temp"`
		Max     *int `json:"max"`
	}{Current: &current}
	embed := characterEmbed(c)
	for _, field := range embed.Fields {
		if field.Name == "Здоровье" {
			if field.Value != "19 ОЗ" {
				t.Fatalf("HP field = %q, want current HP without fabricated maximum", field.Value)
			}
			return
		}
	}
	t.Fatal("HP field missing")
}

func TestCharacterInventoryAndSpellEmbeds(t *testing.T) {
	level := 2
	quantity := 3
	c := characterDetails{}
	c.Identity.Name = "Андрей"
	c.Items = []characterItem{
		{Name: "Меч", Type: "weapon", Category: "equipment", Quantity: &quantity},
		{Name: "Двойной разрез", Type: "feat", Category: "ability"},
		{Name: "Огненный снаряд", Type: "spell", Level: &level},
	}
	inventory := characterInventoryEmbed(c)
	if !strings.Contains(inventory.Fields[0].Value, "Меч × 3") ||
		strings.Contains(inventory.Fields[0].Value, "Двойной разрез") {
		t.Fatalf("unexpected inventory embed: %+v", inventory)
	}
	spells := characterSpellsEmbed(c)
	if len(spells.Fields) != 1 || !strings.Contains(spells.Fields[0].Value, "Огненный снаряд · ранг 2") {
		t.Fatalf("unexpected spell embed: %+v", spells)
	}
}


func TestCharacterInventoryIncludesAllPhysicalItemTypesAndState(t *testing.T) {
	one := 1
	worn := "worn"
	stowed := "stowed"
	inSlot := true
	invested := true
	container := "pack"
	c := characterDetails{}
	c.Identity.Name = "Малахия"
	c.Items = []characterItem{
		{ID:"weapon",Name:"Кинжал",Type:"weapon",Category:"equipment",Quantity:&one,Equipped:&characterEquipped{CarryType:&worn,InSlot:&inSlot}},
		{ID:"armor",Name:"Кожаный доспех",Type:"armor",Category:"equipment",Equipped:&characterEquipped{CarryType:&worn,InSlot:&inSlot,Invested:&invested}},
		{ID:"gear",Name:"Верёвка",Type:"equipment",Category:"equipment",ContainerID:&container,Equipped:&characterEquipped{CarryType:&worn}},
		{ID:"potion",Name:"Зелье",Type:"consumable",Category:"equipment",Quantity:&one,Equipped:&characterEquipped{CarryType:&stowed}},
		{ID:"pack",Name:"Рюкзак",Type:"backpack",Category:"equipment"},
		{ID:"coin",Name:"Монеты",Type:"treasure",Category:"equipment",Quantity:&one},
		{ID:"custom",Name:"Талисман",Type:"customPhysical",Category:"other",Quantity:&one},
	}
	embeds := characterInventoryEmbeds(c)
	if len(embeds) < 5 {
		t.Fatalf("got %d grouped inventory embeds, want all physical item types", len(embeds))
	}
	joined := ""
	for _, embed := range embeds {
		for _, field := range embed.Fields { joined += field.Name + "\n" + field.Value + "\n" }
	}
	for _, want := range []string{"Кинжал","Кожаный доспех","Верёвка","Зелье","Рюкзак","Монеты","Талисман","НАДЕТО","ИНВЕСТИРОВАНО","УБРАНО","в Рюкзак"} {
		if !strings.Contains(joined,want) { t.Errorf("inventory output missing %q: %s",want,joined) }
	}
}

func TestCharacterSpellsGroupByTraditionAndRank(t *testing.T) {
	one, two := 1, 2
	arcane, primal := "arcane", "primal"
	arcaneList, primalList := "Арканная книга", "Природная книга"
	prepared := true
	c := characterDetails{}
	c.Identity.Name = "Заклинатель"
	c.Items = []characterItem{
		{Name:"Щит",Type:"spell",Level:&one,Tradition:&arcane,SpellList:&arcaneList,Prepared:&prepared},
		{Name:"Магическая стрела",Type:"spell",Level:&one,Tradition:&arcane,SpellList:&arcaneList},
		{Name:"Лечебное растение",Type:"spell",Level:&two,Tradition:&primal,SpellList:&primalList},
	}
	embeds := characterSpellsEmbeds(c)
	if len(embeds) != 2 { t.Fatalf("got %d spell groups, want 2",len(embeds)) }
	joined := ""
	for _, embed := range embeds { for _, field := range embed.Fields { joined += field.Name + "\n" + field.Value + "\n" } }
	for _, want := range []string{"Арканная традиция","Природная традиция","ранг 1","ранг 2","подготовлено","Магическая стрела","Лечебное растение"} {
		if !strings.Contains(joined,want) { t.Errorf("spell output missing %q: %s",want,joined) }
	}
}

func TestCharacterEmbedMarksDyingAtMaximum(t *testing.T) {
	current, maximum := 4, 4
	c := characterDetails{}
	c.Identity.Name = "Павший"
	c.Attributes.Dying = &characterDying{Current:&current,Max:&maximum}
	embed := characterEmbed(c)
	if !strings.Contains(embed.Title,"МЁРТВ") { t.Fatalf("title = %q, want death marker",embed.Title) }
	if embed.Color != 0xED4245 { t.Fatalf("color = %#x, want death color",embed.Color) }
}

func TestCharacterSkillLinesUseProficiencyNamesAndHideUntrained(t *testing.T) {
	trained, expert, master, legendary, untrained := 1, 2, 3, 4, 0
	lines := characterSkillLines(
		map[string]characterSkill{
			"athletics":    {Rank: &trained},
			"crafting":     {Rank: &expert},
			"intimidation": {Rank: &legendary},
			"thievery":     {Rank: &master},
			"diplomacy":    {Rank: &untrained},
		},
		map[string]characterSkill{},
	)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Атлетика — Обучен", "Ремесло — Эксперт", "Воровство — Мастер", "Запугивание — Легенда"} {
		if !strings.Contains(joined, want) {
			t.Errorf("skill lines %q do not contain %q", joined, want)
		}
	}
	if strings.Contains(joined, "Дипломатия") || strings.Contains(joined, "0") {
		t.Errorf("untrained skill must be omitted: %q", joined)
	}
}
