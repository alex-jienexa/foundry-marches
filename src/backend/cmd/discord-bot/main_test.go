package main

import (
	"strings"
	"testing"
)

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
	for _, removed := range []string{"Родословная", "Наследие", "Предыстория", "Класс"} {
		if _, exists := fields[removed]; exists {
			t.Errorf("fundamental choice %q must not be displayed", removed)
		}
	}
	if embed.Description != "" {
		t.Errorf("fundamental choices must not be moved into description: %q", embed.Description)
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
