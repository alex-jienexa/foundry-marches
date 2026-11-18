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
	if !ok { t.Fatal("runtime.Caller failed") }
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "docs", "foundry", name)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath(t, name))
	if err != nil { t.Fatalf("read fixture %s: %v", name, err) }
	return data
}

func TestDecodeActorRealFixture(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil { t.Fatal(err) }
	c := decoded.Character
	if c.Identity.ID != "mxgc0DRWC8bA9ruI" || c.Identity.Name != "АНДРЕЙ" { t.Fatalf("unexpected identity: %+v", c.Identity) }
	if c.Source.CoreVersion != "14.367" || c.Source.SystemID != "pf2e" || c.Source.SystemVersion != "8.7.0" { t.Fatalf("unexpected source: %+v", c.Source) }
	if c.Progression.Level == nil || *c.Progression.Level != 1 || c.Progression.Experience.Value == nil || *c.Progression.Experience.Value != 0 { t.Fatalf("unexpected progression: %+v", c.Progression) }
	if c.Attributes.HP == nil || c.Attributes.HP.Current == nil || *c.Attributes.HP.Current != 20 { t.Fatalf("unexpected HP: %+v", c.Attributes.HP) }
	if c.Attributes.Abilities != nil { t.Fatalf("null system.abilities must remain nil: %+v", c.Attributes.Abilities) }
	if c.Resources.HeroPoints == nil || *c.Resources.HeroPoints.Value != 1 { t.Fatalf("unexpected hero points: %+v", c.Resources.HeroPoints) }
	if len(c.Items) != 16 { t.Fatalf("got %d Items, want 16", len(c.Items)) }
	if c.Build.Class == nil || c.Build.Class.Name != "Воин / Fighter" { t.Fatalf("class reference not decoded: %+v", c.Build.Class) }
	if c.Build.Background == nil || c.Build.Background.Name != "Канонир / Cannoneer" { t.Fatalf("background reference not decoded: %+v", c.Build.Background) }
	if _, ok := c.Skills["thievery"]; !ok { t.Fatal("thievery skill missing") }
	if _, ok := c.Skills["athletics"]; ok { t.Fatal("decoder must not synthesize Athletics from class Rule Elements or background") }

	var weapon, armor, potion, coin *CharacterItemDto
	for i := range c.Items {
		switch c.Items[i].Type {
		case "weapon": weapon = &c.Items[i]
		case "armor": armor = &c.Items[i]
		case "consumable": potion = &c.Items[i]
		case "treasure": if c.Items[i].Name == "Серебряные монеты / Silver Pieces" { coin = &c.Items[i] }
		}
	}
	if weapon == nil || weapon.Quantity == nil || *weapon.Quantity != 1 { t.Fatalf("weapon quantity not decoded: %+v", weapon) }
	if armor == nil || armor.Equipment == nil || armor.Equipment.Invested == nil || !*armor.Equipment.Invested { t.Fatalf("armor equipment state not decoded: %+v", armor) }
	if potion == nil || potion.Level == nil || *potion.Level != 3 { t.Fatalf("potion level not decoded: %+v", potion) }
	if coin == nil || coin.Quantity == nil || *coin.Quantity != 34 { t.Fatalf("coin quantity not decoded: %+v", coin) }
	if len(decoded.RawItems) != 16 || len(decoded.RawActor) == 0 { t.Fatal("raw snapshots were not retained") }
}

func TestDecodeRelayEnvelopeFixture(t *testing.T) {
	decoded, err := DecodeRelayResponse(readFixture(t, fixtureResponse))
	if err != nil { t.Fatal(err) }
	if decoded.Character.Identity.ID != "mxgc0DRWC8bA9ruI" { t.Fatalf("unexpected Actor ID %q", decoded.Character.Identity.ID) }
}

func TestDecodeItemUnknownTypeIsPreserved(t *testing.T) {
	raw := []byte(`{"_id":"custom1","name":"Custom","type":"future-type","system":{"quantity":7,"newField":{"x":1}},"flags":{"module":{"keep":true}}}`)
	item, err := DecodeItem(raw)
	if err != nil { t.Fatal(err) }
	if item.normalized.Category != ItemCategoryOther { t.Fatalf("category = %q", item.normalized.Category) }
	if item.normalized.Quantity == nil || *item.normalized.Quantity != 7 { t.Fatal("quantity missing") }
	if _, ok := item.normalized.Details["newField"]; !ok { t.Fatal("unknown system field lost") }
	if _, ok := item.raw.Flags["module"]; !ok { t.Fatal("unknown flags lost") }
}

func TestDecodeActorRejectsIncompleteItemsCollection(t *testing.T) {
	for _, raw := range []string{
		`{"_id":"a1","type":"character"}`,
		`{"_id":"a1","type":"character","items":null}`,
		`{"_id":"a1","type":"character","items":{}}`,
	} {
		if _, err := DecodeActor([]byte(raw)); err == nil { t.Fatalf("expected error for %s", raw) }
	}
}

func TestDecodeActorKeepsMissingOptionalValuesNil(t *testing.T) {
	raw := []byte(`{"_id":"a1","type":"character","name":"Test","items":[],"system":{"abilities":null,"skills":{}}}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	if decoded.Character.Attributes.Abilities != nil { t.Fatal("null abilities must remain nil") }
	if decoded.Character.Progression.Level != nil { t.Fatal("missing level must remain nil") }
	if decoded.Character.Attributes.HP != nil { t.Fatal("missing HP must remain nil") }
}
