package characters

import (
	"encoding/json"
	"testing"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

func TestMapFoundryActorPreservesNormalizedDataAndSnapshot(t *testing.T) {
	level, xp, quantity, hp := 4, 325, 2, 48
	image := "https://example.invalid/actor.webp"
	decoded := pf2e.DecodedActor{
		Character: pf2e.CharacterDto{
			Identity:    pf2e.CharacterIdentityDto{ID: "actor-id", UUID: "Actor.actor-id", Name: "Test Hero", Type: "character", Image: &image},
			Progression: pf2e.CharacterProgressionDto{Level: &level, Experience: pf2e.ExperienceDto{Value: &xp}},
			Build:       pf2e.CharacterBuildDto{Class: &pf2e.CharacterItemReferenceDto{Name: "Fighter"}},
			Attributes:  pf2e.CharacterAttributesDto{HP: &pf2e.HitPointsDto{Current: &hp}},
			Items:       []pf2e.CharacterItemDto{{ID: "item-1", Name: "Longsword", Quantity: &quantity}},
		},
		RawSnapshot: json.RawMessage(`{"_id":"actor-id","name":"Test Hero","items":[]}`),
	}
	got, err := MapFoundryActor(FoundrySyncInput{
		WorldID: "world-uuid", CharacterID: "character-uuid", ActorUUID: "Actor.actor-id", Decoded: decoded,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Character.Name != "Test Hero" || got.Character.Class == nil || *got.Character.Class != "Fighter" {
		t.Fatalf("character projection mismatch: %+v", got.Character)
	}
	if got.Character.Level == nil || *got.Character.Level != 4 || got.Character.XP == nil || *got.Character.XP != 325 {
		t.Fatalf("progression was not mapped: %+v", got.Character)
	}
	if got.Character.HPCurrent == nil || *got.Character.HPCurrent != 48 || got.Character.HPMax != nil {
		t.Fatalf("HP presence was not preserved: %+v", got.Character)
	}
	if len(got.Inventory) != 1 || got.Inventory[0].FoundryItemID != "item-1" || got.Inventory[0].Quantity == nil {
		t.Fatalf("inventory mapping mismatch: %+v", got.Inventory)
	}
	if !json.Valid(got.Character.Attributes) || !json.Valid(got.Character.Skills) || !json.Valid(got.Inventory[0].State) {
		t.Fatal("normalized JSON payload is invalid")
	}
	if len(got.SnapshotHash) != 64 {
		t.Fatalf("unexpected SHA-256 length: %q", got.SnapshotHash)
	}
	if string(got.ActorSnapshot) != string(decoded.RawSnapshot) {
		t.Fatal("raw Actor snapshot changed during mapping")
	}
}

func TestMapFoundryActorDoesNotInventMissingQuantityOrLevel(t *testing.T) {
	decoded := pf2e.DecodedActor{
		Character: pf2e.CharacterDto{
			Identity: pf2e.CharacterIdentityDto{ID: "actor-id", UUID: "Actor.actor-id", Name: "Test Hero", Type: "character"},
			Items:    []pf2e.CharacterItemDto{{ID: "item-1", Name: "Unstacked item"}},
		},
		RawSnapshot: json.RawMessage(`{"_id":"actor-id","name":"Test Hero"}`),
	}
	got, err := MapFoundryActor(FoundrySyncInput{
		WorldID: "world", CharacterID: "character", ActorUUID: "Actor.actor-id", Decoded: decoded,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Character.Level != nil || got.Inventory[0].Quantity != nil {
		t.Fatal("mapper must preserve absent level and quantity")
	}
	var state map[string]any
	if err := json.Unmarshal(got.Inventory[0].State, &state); err != nil {
		t.Fatal(err)
	}
	if value, ok := state["quantity"]; !ok || value != nil {
		t.Fatalf("item state lost absent quantity: %s", got.Inventory[0].State)
	}
}

func TestMapFoundryActorRejectsInvalidSnapshotAndLevel(t *testing.T) {
	level := 31
	base := FoundrySyncInput{
		WorldID: "world", CharacterID: "character", ActorUUID: "Actor.actor-id",
		Decoded: pf2e.DecodedActor{
			Character:   pf2e.CharacterDto{Identity: pf2e.CharacterIdentityDto{ID: "actor-id", Name: "Test Hero", Type: "character"}},
			RawSnapshot: json.RawMessage(`{"valid":true}`),
		},
	}
	base.Decoded.Character.Progression.Level = &level
	if _, err := MapFoundryActor(base); err == nil {
		t.Fatal("expected out-of-range level to fail")
	}
	base.Decoded.Character.Progression.Level = nil
	base.Decoded.RawSnapshot = json.RawMessage(`{invalid}`)
	if _, err := MapFoundryActor(base); err == nil {
		t.Fatal("expected invalid snapshot to fail")
	}
}
