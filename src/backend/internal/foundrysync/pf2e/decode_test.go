package pf2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const fixtureActor = "mxgc0DRWC8bA9ruI-20261118T073146Z-actor.json"
const fixtureResponse = "mxgc0DRWC8bA9ruI-20261118T073146Z-response.json"

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "docs", "foundry", name)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath(t, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestDecodeActorRealFixture(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil {
		t.Fatal(err)
	}
	c := decoded.Character
	if c.Identity.ID != "mxgc0DRWC8bA9ruI" || c.Identity.Name != "АНДРЕЙ" {
		t.Fatalf("unexpected identity: %+v", c.Identity)
	}
	if c.Source.CoreVersion != "14.367" || c.Source.SystemID != "pf2e" || c.Source.SystemVersion != "8.7.0" {
		t.Fatalf("unexpected source: %+v", c.Source)
	}
	if c.Source.MigrationVersion == nil || *c.Source.MigrationVersion != 0.961 {
		t.Fatalf("unexpected migration version: %+v", c.Source.MigrationVersion)
	}
	if c.Progression.Level == nil || *c.Progression.Level != 1 || c.Progression.Experience.Value == nil || *c.Progression.Experience.Value != 0 {
		t.Fatalf("unexpected progression: %+v", c.Progression)
	}
	if c.Attributes.HP == nil || c.Attributes.HP.Current == nil || *c.Attributes.HP.Current != 20 {
		t.Fatalf("unexpected HP: %+v", c.Attributes.HP)
	}
	if c.Attributes.HP.Max != nil {
		t.Fatalf("maximum HP is not stored in this Actor snapshot: %+v", c.Attributes.HP)
	}
	if c.Resources.HeroPoints == nil || c.Resources.HeroPoints.Value == nil || *c.Resources.HeroPoints.Value != 1 {
		t.Fatalf("unexpected hero points: %+v", c.Resources.HeroPoints)
	}
	if len(c.Items) != 16 {
		t.Fatalf("got %d Items, want 16", len(c.Items))
	}
	if c.Build.Class == nil || c.Build.Class.Name != "Воин / Fighter" {
		t.Fatalf("class reference not decoded: %+v", c.Build.Class)
	}
	if c.Build.Background == nil || c.Build.Background.Name != "Канонир / Cannoneer" {
		t.Fatalf("background reference not decoded: %+v", c.Build.Background)
	}
	if c.Build.KeyAbility == nil || *c.Build.KeyAbility != "str" {
		t.Fatalf("key ability not decoded: %+v", c.Build.KeyAbility)
	}
	if _, ok := c.Skills.Standard["thievery"]; !ok {
		t.Fatal("thievery skill missing")
	}
	if _, ok := c.Skills.Standard["athletics"]; ok {
		t.Fatal("decoder must not synthesize Athletics from class Rule Elements or background")
	}
	if len(c.Skills.Lore) != 0 {
		t.Fatalf("unexpected Lore skills synthesized from Background: %+v", c.Skills.Lore)
	}
	if c.Biography.Age == nil || *c.Biography.Age != "" {
		t.Fatalf("empty but present age should be preserved: %+v", c.Biography.Age)
	}
	if len(decoded.RawSnapshot) == 0 {
		t.Fatal("raw Actor snapshot was not retained")
	}

	var weapon, armor, potion, coin *CharacterItemDto
	for i := range c.Items {
		switch c.Items[i].Type {
		case "weapon":
			weapon = &c.Items[i]
		case "armor":
			armor = &c.Items[i]
		case "consumable":
			potion = &c.Items[i]
		case "treasure":
			if c.Items[i].Name == "Серебряные монеты / Silver Pieces" {
				coin = &c.Items[i]
			}
		}
	}
	if weapon == nil || weapon.Quantity == nil || *weapon.Quantity != 1 {
		t.Fatalf("weapon quantity not decoded: %+v", weapon)
	}
	if armor == nil || armor.Equipped == nil || armor.Equipped.Invested == nil || !*armor.Equipped.Invested {
		t.Fatalf("armor equipped state not decoded: %+v", armor)
	}
	if potion == nil || potion.Level == nil || *potion.Level != 3 {
		t.Fatalf("potion level not decoded: %+v", potion)
	}
	if coin == nil || coin.Quantity == nil || *coin.Quantity != 34 {
		t.Fatalf("coin quantity not decoded: %+v", coin)
	}
}

func TestDecodeRelayEnvelopeFixture(t *testing.T) {
	decoded, err := DecodeRelayResponse(readFixture(t, fixtureResponse))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Character.Identity.ID != "mxgc0DRWC8bA9ruI" {
		t.Fatalf("unexpected Actor ID %q", decoded.Character.Identity.ID)
	}
}

func TestDecodeItemUnknownTypeIsPreserved(t *testing.T) {
	raw := []byte(`{"_id":"custom1","name":"Custom","type":"future-type","system":{"quantity":7,"newField":{"x":1}},"flags":{"module":{"keep":true}}}`)
	item, err := DecodeItem(raw)
	if err != nil {
		t.Fatal(err)
	}
	if item.Character.Category != ItemCategoryOther {
		t.Fatalf("category = %q", item.Character.Category)
	}
	if item.Character.Quantity == nil || *item.Character.Quantity != 7 {
		t.Fatal("quantity missing")
	}
	if _, ok := item.Foundry.System["newField"]; !ok {
		t.Fatal("unknown system field lost from raw Item DTO")
	}
	if _, ok := item.Foundry.Flags["module"]; !ok {
		t.Fatal("unknown flags lost from raw Item DTO")
	}
	if string(item.RawSnapshot) != string(raw) {
		t.Fatal("standalone Item raw snapshot was not retained")
	}
}

func TestDecodeActorRejectsIncompleteItemsCollection(t *testing.T) {
	for _, raw := range []string{
		`{"_id":"a1","type":"character"}`,
		`{"_id":"a1","type":"character","items":null}`,
		`{"_id":"a1","type":"character","items":{}}`,
	} {
		if _, err := DecodeActor([]byte(raw)); err == nil {
			t.Fatalf("expected error for %s", raw)
		}
	}
}

func TestDecodeActorRejectsDuplicateItemIDs(t *testing.T) {
	raw := []byte(`{"_id":"a1","type":"character","items":[{"_id":"same","type":"feat"},{"_id":"same","type":"action"}]}`)
	if _, err := DecodeActor(raw); err == nil {
		t.Fatal("expected duplicate Item ID to be rejected")
	}
}

func TestDecodeActorKeepsMissingOptionalValuesNil(t *testing.T) {
	raw := []byte(`{"_id":"a1","type":"character","name":"Test","items":[],"system":{"abilities":null,"skills":{}}}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Character.Progression.Level != nil {
		t.Fatal("missing level must remain nil")
	}
	if decoded.Character.Attributes.HP != nil {
		t.Fatal("missing HP must remain nil")
	}
}

func TestDecodeSkillsSeparatesLoreWithoutDerivingIt(t *testing.T) {
	raw := []byte(`{"lore-warfare":{"rank":1},"athletics":{"rank":2},"thievery":{"rank":3}}`)
	skills := decodeSkills(raw)
	if _, ok := skills.Lore["lore-warfare"]; !ok {
		t.Fatal("Lore skill not separated")
	}
	if _, ok := skills.Standard["athletics"]; !ok {
		t.Fatal("standard skill missing")
	}
	if len(skills.Standard) != 2 {
		t.Fatalf("unexpected standard skills: %+v", skills.Standard)
	}
}


func TestDecodeStatisticsKeepsMissingValuesUnknown(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil { t.Fatal(err) }
	stats := decoded.Character.Statistics
	if len(stats.Abilities) != 0 {
		t.Fatalf("null Actor.system.abilities must not be reconstructed: %+v", stats.Abilities)
	}
	if stats.Proficiencies.Perception != nil || len(stats.Proficiencies.Saves) != 0 {
		t.Fatalf("missing Actor proficiencies must remain unavailable: %+v", stats.Proficiencies)
	}
	class := decoded.Character.Build.ClassProficiencies
	if class == nil || class.PerceptionRank == nil || *class.PerceptionRank != 2 {
		t.Fatalf("class progression profile not decoded: %+v", class)
	}
	if class.SavingThrows["fortitude"] != 2 || class.SavingThrows["reflex"] != 2 || class.SavingThrows["will"] != 1 {
		t.Fatalf("unexpected class save baseline: %+v", class.SavingThrows)
	}
	if class.AdditionalSkills == nil || *class.AdditionalSkills != 3 {
		t.Fatalf("class additional skill count missing: %+v", class)
	}
}

func TestDecodeEquipmentCalculationInputs(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil { t.Fatal(err) }
	var armor *CharacterItemDto
	for i := range decoded.Character.Items {
		if decoded.Character.Items[i].Type == "armor" { armor = &decoded.Character.Items[i]; break }
	}
	if armor == nil || armor.Armor == nil || armor.Runes == nil {
		t.Fatalf("armor calculation inputs missing: %+v", armor)
	}
	if armor.Armor.ACBonus == nil || *armor.Armor.ACBonus != 2 ||
		armor.Armor.DexCap == nil || *armor.Armor.DexCap != 3 {
		t.Fatalf("unexpected armor values: %+v", armor.Armor)
	}
	if armor.Runes.Potency == nil || *armor.Runes.Potency != 1 ||
		armor.Runes.Resilient == nil || *armor.Runes.Resilient != 1 {
		t.Fatalf("unexpected rune values: %+v", armor.Runes)
	}
}

func TestDecodeExplicitAbilityAndProficiencyInputs(t *testing.T) {
	raw := []byte(`{"_id":"a1","type":"character","items":[],"system":{
		"abilities":{"str":{"value":18,"mod":4},"dex":{"value":14,"mod":2}},
		"proficiencies":{
			"perception":{"rank":2},
			"savingThrows":{"fortitude":{"rank":3},"reflex":{"rank":2}},
			"defenses":{"unarmored":{"rank":1}}
		}
	}}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	stats := decoded.Character.Statistics
	if got := stats.Abilities["str"]; got.Score == nil || *got.Score != 18 || got.Modifier == nil || *got.Modifier != 4 {
		t.Fatalf("unexpected Strength input: %+v", got)
	}
	if stats.Proficiencies.Perception == nil || stats.Proficiencies.Perception.Rank == nil || *stats.Proficiencies.Perception.Rank != 2 {
		t.Fatalf("perception proficiency missing: %+v", stats.Proficiencies.Perception)
	}
	if got := stats.Proficiencies.Saves["fortitude"]; got.Rank == nil || *got.Rank != 3 {
		t.Fatalf("Fortitude proficiency missing: %+v", got)
	}
	if got := stats.Proficiencies.Defenses["unarmored"]; got.Rank == nil || *got.Rank != 1 {
		t.Fatalf("unarmored proficiency missing: %+v", got)
	}
}
