package main

import (
	"strings"
	"testing"
)

func TestMyCharactersEmbedsShowsSlotsAndCharacterSummary(t *testing.T) {
	portrait := "portrait.webp"
	result := myCharactersResult{
		ActiveSlots: 1,
		MaxActiveSlots: 2,
		RemainingSlots: 1,
		Characters: []ownedCharacter{{
			ID: "character-1",
			ActorUUID: "Actor.one",
			Name: "Андрей",
			PortraitURL: &portrait,
			Level: 5,
			XP: 1200,
			Ancestry: "Человек",
			Class: "Воин",
			Status: "active",
		}},
	}

	embeds := myCharactersEmbeds(result, "https://foundry.example.test")
	if len(embeds) != 1 {
		t.Fatalf("got %d embeds, want 1", len(embeds))
	}
	embed := embeds[0]
	if !strings.Contains(embed.Description, "1/2") || !strings.Contains(embed.Description, "свободно: **1**") {
		t.Fatalf("slot usage missing from description: %q", embed.Description)
	}
	if !strings.Contains(embed.Description, "Уровень **5**") || !strings.Contains(embed.Description, "1200 XP") {
		t.Fatalf("level/XP missing from description: %q", embed.Description)
	}
	if embed.Thumbnail == nil || embed.Thumbnail.URL != "https://foundry.example.test/portrait.webp" {
		t.Fatalf("portrait URL not resolved: %+v", embed.Thumbnail)
	}
	if len(embed.Fields) != 2 || embed.Fields[0].Value != "Человек" || embed.Fields[1].Value != "Воин" {
		t.Fatalf("ancestry/class fields mismatch: %+v", embed.Fields)
	}
}

func TestMyCharactersEmbedsShowsEmptyRosterAndSlots(t *testing.T) {
	embeds := myCharactersEmbeds(myCharactersResult{ActiveSlots: 0, MaxActiveSlots: 2, RemainingSlots: 2}, "")
	if len(embeds) != 1 {
		t.Fatalf("got %d embeds, want 1", len(embeds))
	}
	if !strings.Contains(embeds[0].Description, "свободно: **2**") || !strings.Contains(embeds[0].Description, "/character-connect") {
		t.Fatalf("empty roster guidance missing: %q", embeds[0].Description)
	}
}
