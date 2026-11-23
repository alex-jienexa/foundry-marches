package main

import "testing"

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
