package main

import (
	"strings"
	"testing"
)

func TestHelpEmbedShowsRegistrationAndLinkGuidance(t *testing.T) {
	embed := helpEmbed(helpStatus{})
	if embed.Title == "" || len(embed.Fields) < 3 {
		t.Fatalf("unexpected unregistered help embed: %+v", embed)
	}
	if !strings.Contains(embed.Fields[0].Value, "/register") {
		t.Fatalf("registration guidance missing: %+v", embed.Fields)
	}
	if !strings.Contains(embed.Fields[2].Value, "/character-link") {
		t.Fatalf("link guidance missing: %+v", embed.Fields)
	}
}

func TestHelpEmbedShowsConnectGuidanceForRegisteredUnlinkedUser(t *testing.T) {
	embed := helpEmbed(helpStatus{Registered: true})
	if len(embed.Fields) < 2 || !strings.Contains(embed.Fields[0].Value, "/character-link") {
		t.Fatalf("unexpected registered/unlinked help embed: %+v", embed)
	}
	if !strings.Contains(embed.Description, "не привязан") {
		t.Fatalf("missing unlinked state: %q", embed.Description)
	}
}

func TestHelpEmbedShowsCharacterCommandsForLinkedUser(t *testing.T) {
	embed := helpEmbed(helpStatus{Registered: true, Linked: true})
	if len(embed.Fields) < 2 {
		t.Fatalf("unexpected linked help embed: %+v", embed)
	}
	for _, command := range []string{"/my-character", "/character name:", "/character-inventory", "/character-spells"} {
		if !strings.Contains(embed.Fields[1].Value, command) {
			t.Errorf("help command list missing %q: %q", command, embed.Fields[1].Value)
		}
	}
}
