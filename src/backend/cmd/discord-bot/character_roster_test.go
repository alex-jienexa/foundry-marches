package main

import (
	"strings"
	"testing"
)

func TestMyCharactersEmbedsGroupsRosterIntoOneEmbed(t *testing.T) {
	result := myCharactersResult{
		ActiveSlots:    2,
		MaxActiveSlots: 2,
		RemainingSlots: 0,
		Characters: []ownedCharacter{
			{
				ID: "character-1", ActorUUID: "Actor.one", Name: "Андрей",
				Level: 5, XP: 1200, Ancestry: "Человек", Class: "Воин", Status: "active",
			},
			{
				ID: "character-2", ActorUUID: "Actor.two", Name: "Кукла-Персонаж",
				Level: 1, XP: 0, Ancestry: "Гном", Status: "active",
			},
		},
	}

	embeds := myCharactersEmbeds(result, "")
	if len(embeds) != 1 {
		t.Fatalf("got %d embeds, want exactly one grouped embed", len(embeds))
	}
	embed := embeds[0]
	if !strings.Contains(embed.Description, "2/2") || !strings.Contains(embed.Description, "свободно: **0**") {
		t.Fatalf("slot usage missing from description: %q", embed.Description)
	}
	if len(embed.Fields) != 2 {
		t.Fatalf("got %d character fields, want 2", len(embed.Fields))
	}
	first := embed.Fields[0]
	if first.Name != "Андрей" || !strings.Contains(first.Value, "Уровень **5**") ||
		!strings.Contains(first.Value, "1200 XP") || !strings.Contains(first.Value, "Человек") ||
		!strings.Contains(first.Value, "Воин") {
		t.Fatalf("first character summary mismatch: %+v", first)
	}
	second := embed.Fields[1]
	if second.Name != "Кукла-Персонаж" || !strings.Contains(second.Value, "Гном") ||
		!strings.Contains(second.Value, "Уровень **1**") {
		t.Fatalf("second character summary mismatch: %+v", second)
	}
}

func TestMyCharactersEmbedsShowsEmptyRosterAndSlots(t *testing.T) {
	embeds := myCharactersEmbeds(myCharactersResult{ActiveSlots: 0, MaxActiveSlots: 2, RemainingSlots: 2}, "")
	if len(embeds) != 1 {
		t.Fatalf("got %d embeds, want 1", len(embeds))
	}
	if len(embeds[0].Fields) != 0 || !strings.Contains(embeds[0].Description, "свободно: **2**") ||
		!strings.Contains(embeds[0].Description, "/character-connect") {
		t.Fatalf("empty roster guidance missing: %+v", embeds[0])
	}
}
