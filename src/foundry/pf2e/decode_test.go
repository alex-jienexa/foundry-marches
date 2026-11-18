package pf2e

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "foundry", name)
	data, err := os.ReadFile(path)
	if err != nil { t.Fatalf("read fixture %s: %v", path, err) }
	return data
}

func TestDecodeCharacterFromActorFixture(t *testing.T) {
	payload := fixture(t, "mxgc0DRWC8bA9ruI-20261118T073146Z-actor.json")
	got, err := DecodeCharacter(payload)
	if err != nil { t.Fatal(err) }

	if got.Identity.ID != "mxgc0DRWC8bA9ruI" || got.Identity.UUID != "Actor.mxgc0DRWC8bA9ruI" {
		t.Fatalf("unexpected identity: %+v", got.Identity)
	}
	if got.Identity.Name != "АНДРЕЙ" || got.Source.SystemID != "pf2e" || got.Source.SystemVersion != "8.7.0" {
		t.Fatalf("unexpected source/identity: %+v %+v", got.Identity, got.Source)
	}
	if got.Progression.Level == nil || *got.Progression.Level != 1 {
		t.Fatalf("level = %v, want 1", got.Progression.Level)
	}
	if got.Progression.Experience == nil || got.Progression.Experience.Value == nil || *got.Progression.Experience.Value != 0 {
		t.Fatalf("unexpected XP: %+v", got.Progression.Experience)
	}
	if got.Attributes.HP == nil || got.Attributes.HP.Value == nil || *got.Attributes.HP.Value != 20 {
		t.Fatalf("unexpected HP: %+v", got.Attributes.HP)
	}
	if got.Resources.HeroPoints == nil || got.Resources.HeroPoints.Value == nil || *got.Resources.HeroPoints.Value != 1 {
		t.Fatalf("unexpected hero points: %+v", got.Resources.HeroPoints)
	}
	if len(got.Items) != 16 { t.Fatalf("items = %d, want 16", len(got.Items)) }
	if got.Build.Ancestry == nil || got.Build.Ancestry.Name != "Человек / Human" { t.Fatalf("ancestry not decoded: %+v", got.Build.Ancestry) }
	if got.Build.Heritage == nil || got.Build.Class == nil || got.Build.Background == nil {
		t.Fatalf("identity items not linked: %+v", got.Build)
	}
	if got.Skills["thievery"].Rank == nil || *got.Skills["thievery"].Rank != 3 {
		t.Fatalf("thievery rank not decoded: %+v", got.Skills["thievery"])
	}
	if !bytes.Equal(bytes.TrimSpace(got.Attributes.Abilities), []byte("null")) {
		t.Fatalf("explicit null abilities lost: %s", got.Attributes.Abilities)
	}
	if len(got.RawActor) == 0 || len(got.RawSystem) == 0 { t.Fatal("raw snapshots must be retained") }

	var weapon, armor, coin, potion *CharacterItemDTO
	for i := range got.Items {
		item := &got.Items[i]
		switch item.Type {
		case "weapon": weapon = item
		case "armor": armor = item
		case "treasure":
			if item.Quantity != nil && *item.Quantity == 11 { coin = item }
		case "consumable": potion = item
		}
	}
	if weapon == nil || weapon.Name != "Длинный меч / Longsword" { t.Fatalf("weapon missing: %+v", weapon) }
	if armor == nil || armor.Equipped == nil || armor.Equipped.Invested == nil || !*armor.Equipped.Invested {
		t.Fatalf("armor equipped state not decoded: %+v", armor)
	}
	if coin == nil { t.Fatal("gold coin stack/quantity not decoded") }
	if potion == nil || potion.Level == nil || *potion.Level != 3 { t.Fatalf("potion level not decoded: %+v", potion) }
}

func TestDecodeCharacterFromRelayEnvelope(t *testing.T) {
	payload := fixture(t, "mxgc0DRWC8bA9ruI-20261118T073146Z-response.json")
	got, err := DecodeRelayCharacter(payload)
	if err != nil { t.Fatal(err) }
	if got.Identity.ID != "mxgc0DRWC8bA9ruI" { t.Fatalf("wrong actor: %+v", got.Identity) }
}

func TestDecodeRejectsIncompleteActorBeforeTreatingItemsAsEmpty(t *testing.T) {
	cases := []string{
		`{"_id":"a","name":"A","type":"character","system":{}}`,
		`{"_id":"a","name":"A","type":"character","system":{},"items":null}`,
		`{"_id":"a","name":"A","type":"character","system":{},"items":{}}`,
	}
	for _, input := range cases {
		if _, err := DecodeCharacter([]byte(input)); err == nil { t.Fatalf("expected error for %s", input) }
	}
}

func TestDecodePreservesUnknownItemTypeAndRawFields(t *testing.T) {
	input := []byte(`{
		"_id":"actor-1","name":"Test","type":"character","img":"actor.webp",
		"system":{"details":{"level":{"value":1}},"skills":{},"abilities":null,"vendorExtension":{"keep":true}},
		"items":[{"_id":"item-1","name":"Mystery","type":"future-document","system":{"slug":"mystery","futureField":{"x":7}}}]
	}`)
	got, err := DecodeCharacter(input)
	if err != nil { t.Fatal(err) }
	if len(got.Items) != 1 || got.Items[0].Category != ItemCategoryOther {
		t.Fatalf("unknown item was not retained: %+v", got.Items)
	}
	if !bytes.Contains(got.Items[0].Details, []byte("futureField")) || !bytes.Contains(got.RawSystem, []byte("vendorExtension")) {
		t.Fatal("unknown source fields were lost")
	}
}
