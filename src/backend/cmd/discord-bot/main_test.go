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

func TestCharacterAbilitiesEmbedSeparatesAbilitiesAndActions(t *testing.T) {
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

	embed := characterAbilitiesEmbed(c)
	fields := make(map[string]string, len(embed.Fields))
	for _, field := range embed.Fields {
		fields[field.Name] = field.Value
	}
	checks := []struct {
		field string
		want  string
	}{
		{"Способности класса", "Двойной разрез"},
		{"Способности родословной", "Видящий во мраке"},
		{"Действия", "Реагирующий удар — реакция"},
		{"Особенности класса", "Реагирующий удар"},
		{"Способности навыков", "Сильный носильщик"},
		{"Другие способности", "Сильный носильщик"},
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
	if strings.Count(fields["Особенности класса"], "Реагирующий удар") != 1 {
		t.Errorf("class feature must not be duplicated: %q", fields["Особенности класса"])
	}
}

func TestCharacterSpellsEmbedSeparatesPreparedSlotsFromPreparationList(t *testing.T) {
	preparedType := "prepared"
	tradition := "arcane"
	list := "Арканная книга"
	prepared := true
	unprepared := false
	c := characterDetails{}
	c.Identity.Name = "Маг"
	c.Items = []characterItem{
		{Name:"Свет",Type:"spell",Level:characterTestIntPointer(0),SpellcastingType:&preparedType,Tradition:&tradition,SpellList:&list,Prepared:&prepared,PreparedRanks:[]int{0}},
		{Name:"Магическая стрела",Type:"spell",Level:characterTestIntPointer(1),SpellcastingType:&preparedType,Tradition:&tradition,SpellList:&list,Prepared:&prepared,PreparedRanks:[]int{1}},
		{Name:"Щит",Type:"spell",Level:characterTestIntPointer(1),SpellcastingType:&preparedType,Tradition:&tradition,SpellList:&list,Prepared:&unprepared},
	}
	embed := characterSpellsEmbed(c)
	fields := make(map[string]string)
	for _, field := range embed.Fields { fields[field.Name] = field.Value }
	var cantripPrepared, rankOnePrepared, rankOneBook bool
	for name, value := range fields {
		if strings.Contains(name, "Подготовлено") && strings.Contains(name, "чары") && strings.Contains(value, "Свет") { cantripPrepared = true }
		if strings.Contains(name, "Подготовлено") && strings.Contains(name, "ранг 1") && strings.Contains(value, "Магическая стрела") { rankOnePrepared = true }
		if strings.Contains(name, "Подготовка") && strings.Contains(name, "ранг 1") && strings.Contains(value, "Щит") { rankOneBook = true }
	}
	if !cantripPrepared || !rankOnePrepared || !rankOneBook {
		t.Fatalf("prepared/book groups incorrect: %+v", fields)
	}
}

func characterTestIntPointer(value int) *int { return &value }

func TestCharacterNavigationHighlightsAndDisablesActivePage(t *testing.T) {
	initial := characterDetailComponents("flow", "overview")
	initialFirst := initial[0].(discordgo.ActionsRow)
	initialCard := initialFirst.Components[0].(discordgo.Button)
	if initialCard.Label != "Карточка" || initialCard.Style != discordgo.PrimaryButton || !initialCard.Disabled {
		t.Fatalf("initial card button should be first, highlighted and disabled: %+v", initialCard)
	}
	components := characterDetailComponents("flow", "skills")
	if len(components) != 2 { t.Fatalf("navigation rows = %d, want 2", len(components)) }
	first := components[0].(discordgo.ActionsRow)
	second := components[1].(discordgo.ActionsRow)
	buttons := append(append([]discordgo.MessageComponent{}, first.Components...), second.Components...)
	if len(buttons) != 6 { t.Fatalf("navigation buttons = %d, want 6", len(buttons)) }
	labels := make([]string, 0, len(buttons))
	activeFound := false
	for _, component := range buttons {
		button := component.(discordgo.Button)
		labels = append(labels, button.Label)
		if button.Label == "Навыки" {
			activeFound = button.Style == discordgo.PrimaryButton && button.Disabled
		}
		if button.Label == "Карточка" && (button.Style != discordgo.SecondaryButton || button.Disabled) {
			t.Fatalf("inactive card button should remain enabled and secondary: %+v", button)
		}
		if button.Label == "К карточке" { t.Fatal("legacy return button must not be present") }
	}
	if !activeFound { t.Fatalf("active skills button was not highlighted and disabled: %v", labels) }
}

func TestCharacterSkillsEmbedGroupsSkillFeatsBySkill(t *testing.T) {
	diplomacy, athletics := "diplomacy", "athletics"
	category := "skill"
	trained := 1
	c := characterDetails{}
	c.Identity.Name = "Андрей"
	c.Skills.Standard = map[string]characterSkill{"diplomacy": {Rank: &trained}}
	c.Items = []characterItem{
		{Name: "Остроумная реплика / Bon Mot", Type: "feat", SystemCategory: &category, Skill: &diplomacy},
		{Name: "Атлетическая мощь / Powerful Leap", Type: "feat", SystemCategory: &category, Skill: &athletics},
	}
	embed := characterSkillsEmbed(c)
	fields := make(map[string]string)
	for _, field := range embed.Fields { fields[field.Name] = field.Value }
	if !strings.Contains(fields["Обучение навыкам"], "Дипломатия — Обучен") {
		t.Fatalf("skills field = %q, missing trained Diplomacy", fields["Обучение навыкам"])
	}
	if !strings.Contains(fields["Способности · Дипломатия"], "Остроумная реплика") {
		t.Fatalf("Diplomacy skill-feat group = %q", fields["Способности · Дипломатия"])
	}
	if !strings.Contains(fields["Способности · Атлетика"], "Атлетическая мощь") {
		t.Fatalf("Athletics skill-feat group = %q", fields["Способности · Атлетика"])
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
		if field.Name == "ОЗ" {
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
	if len(spells.Fields) != 1 || !strings.Contains(spells.Fields[0].Name, "ранг 2") || !strings.Contains(spells.Fields[0].Value, "Огненный снаряд") {
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
		{ID: "weapon", Name: "Кинжал", Type: "weapon", Category: "equipment", Quantity: &one, Equipped: &characterEquipped{CarryType: &worn, InSlot: &inSlot}},
		{ID: "armor", Name: "Кожаный доспех", Type: "armor", Category: "equipment", Equipped: &characterEquipped{CarryType: &worn, InSlot: &inSlot, Invested: &invested}},
		{ID: "gear", Name: "Верёвка", Type: "equipment", Category: "equipment", ContainerID: &container, Equipped: &characterEquipped{CarryType: &worn}},
		{ID: "potion", Name: "Зелье", Type: "consumable", Category: "equipment", Quantity: &one, Equipped: &characterEquipped{CarryType: &stowed}},
		{ID: "pack", Name: "Рюкзак", Type: "backpack", Category: "equipment"},
		{ID: "coin", Name: "Монеты", Type: "treasure", Category: "equipment", Quantity: &one},
		{ID: "custom", Name: "Талисман", Type: "customPhysical", Category: "other", Quantity: &one},
	}
	embeds := characterInventoryEmbeds(c)
	if len(embeds) != 1 {
		t.Fatalf("got %d inventory messages, want all groups in one message", len(embeds))
	}
	joined := ""
	for _, embed := range embeds {
		for _, field := range embed.Fields {
			joined += field.Name + "\n" + field.Value + "\n"
		}
	}
	for _, want := range []string{"Кинжал", "Кожаный доспех", "Верёвка", "Зелье", "Рюкзак", "Монеты", "Талисман", "НАДЕТО", "ИНВЕСТИРОВАНО", "УБРАНО", "в Рюкзак"} {
		if !strings.Contains(joined, want) {
			t.Errorf("inventory output missing %q: %s", want, joined)
		}
	}
}

func TestCharacterSpellsGroupByTraditionAndRank(t *testing.T) {
	zero, one, two := 0, 1, 2
	arcane, primal := "arcane", "primal"
	arcaneList, primalList := "Арканная книга", "Природная книга"
	prepared := true
	c := characterDetails{}
	c.Identity.Name = "Заклинатель"
	c.Items = []characterItem{
		{Name: "Щит", Type: "spell", Level: &one, Tradition: &arcane, SpellList: &arcaneList, Prepared: &prepared},
		{Name: "Магическая стрела", Type: "spell", Level: &one, Tradition: &arcane, SpellList: &arcaneList},
		{Name: "Зажечь огонь", Type: "spell", Level: &zero, Tradition: &arcane, SpellList: &arcaneList},
		{Name: "Лечебное растение", Type: "spell", Level: &two, Tradition: &primal, SpellList: &primalList},
	}
	embeds := characterSpellsEmbeds(c)
	if len(embeds) != 1 || len(embeds[0].Fields) != 3 {
		t.Fatalf("got %d spell embeds with %d fields, want one message with 3 grouped fields", len(embeds), len(embeds[0].Fields))
	}
	joined := ""
	for _, embed := range embeds {
		for _, field := range embed.Fields {
			joined += field.Name + "\n" + field.Value + "\n"
		}
	}
	for _, want := range []string{"Арканная традиция", "Природная традиция", "ранг 0", "ранг 1", "ранг 2", "подготовлено", "Магическая стрела", "Зажечь огонь", "Лечебное растение"} {
		if !strings.Contains(joined, want) {
			t.Errorf("spell output missing %q: %s", want, joined)
		}
	}
}

func TestCharacterSummaryCardContainsOnlyCoreFields(t *testing.T) {
	level, xp, hp, maxHP := 2, 400, 2, 24
	c := characterDetails{}
	c.Identity.Name = "Малахия"
	c.Build.Class = &characterReference{Name: "Следователь"}
	c.Build.Ancestry = &characterReference{Name: "Человек"}
	c.Progression.Level = &level
	c.Progression.Experience.Value = &xp
	c.Attributes.HP = &struct {
		Current *int `json:"current"`
		Temp    *int `json:"temp"`
		Max     *int `json:"max"`
	}{Current: &hp, Max: &maxHP}
	score, modifier := 12, 1
	c.Statistics.Abilities = map[string]struct {
		Score    *int `json:"score"`
		Modifier *int `json:"modifier"`
		Derived  bool `json:"derived"`
	}{"int": {Score: &score, Modifier: &modifier}}
	embed := characterEmbed(c)
	fields := make(map[string]string, len(embed.Fields))
	for _, field := range embed.Fields {
		fields[field.Name] = field.Value
	}
	for _, name := range []string{"Класс", "Родословная", "Уровень", "ОЗ", "Характеристики", "Опыт"} {
		if fields[name] == "" {
			t.Errorf("summary card is missing %q", name)
		}
	}
	if fields["ОЗ"] != "2 / 24" || !strings.Contains(fields["Характеристики"], "ИНТ: 12 (+1)") || fields["Опыт"] != "400" {
		t.Fatalf("summary card fields = %#v", fields)
	}
	for _, name := range []string{"Способности класса", "Действия", "Навыки", "Предметы"} {
		if fields[name] != "" {
			t.Errorf("detailed field %q should not be in summary card", name)
		}
	}
}

func TestInventoryStateDoesNotTreatEveryWornItemAsWorn(t *testing.T) {
	worn := "worn"
	inSlot := false
	invested := false
	ordinary := characterItem{Type: "equipment", Equipped: &characterEquipped{CarryType: &worn, InSlot: &inSlot, Invested: &invested}}
	if got := inventoryStateLabel(ordinary, nil); got != "" {
		t.Fatalf("ordinary worn carry state = %q, want no misleading worn label", got)
	}
	armorSlot := true
	armor := characterItem{Type: "armor", Equipped: &characterEquipped{CarryType: &worn, InSlot: &armorSlot}}
	if got := inventoryStateLabel(armor, nil); got != "НАДЕТО КАК ДОСПЕХ" {
		t.Fatalf("armor state = %q", got)
	}
	stowed := "stowed"
	container := "backpack-id"
	item := characterItem{Type: "equipment", ContainerID: &container, Equipped: &characterEquipped{CarryType: &stowed}}
	if got := inventoryStateLabel(item, map[string]string{container: "Рюкзак"}); got != "В РЮКЗАКЕ" {
		t.Fatalf("stowed item state = %q", got)
	}
}

func TestCharacterDetailButtonsStayOnOneMessage(t *testing.T) {
	for _, section := range []string{"overview", "abilities", "skills", "inventory", "spells", "metrics"} {
		rows := characterDetailComponents("abc123", section)
		if len(rows) != 2 {
			t.Fatalf("%s navigation rows = %d, want 2", section, len(rows))
		}
		total := 0
		for _, component := range rows {
			row, ok := component.(discordgo.ActionsRow)
			if !ok {
				t.Fatalf("%s navigation component has type %T, want ActionsRow", section, component)
			}
			total += len(row.Components)
		}
		if total != 6 {
			t.Fatalf("%s navigation buttons = %d, want 6", section, total)
		}
	}
}

func TestCharacterEmbedMarksDyingAtMaximum(t *testing.T) {
	current, maximum := 4, 4
	c := characterDetails{}
	c.Identity.Name = "Павший"
	c.Attributes.Dying = &characterDying{Current: &current, Max: &maximum}
	embed := characterEmbed(c)
	if !strings.Contains(embed.Title, "МЁРТВ") {
		t.Fatalf("title = %q, want death marker", embed.Title)
	}
	if embed.Color != 0xED4245 {
		t.Fatalf("color = %#x, want death color", embed.Color)
	}
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
