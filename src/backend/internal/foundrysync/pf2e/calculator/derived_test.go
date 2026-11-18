package calculator

import (
	"testing"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

func TestCalculateFighterDerivedStatsFromPersistentInputs(t *testing.T) {
	decoded, err := pf2e.DecodeActor(loadActorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	decoded.Character.Statistics.Abilities = map[string]pf2e.AbilityScoreDto{
		"str": {Modifier: intPointer(4)},
		"dex": {Modifier: intPointer(4)},
		"con": {Modifier: intPointer(2)},
		"wis": {Modifier: intPointer(1)},
	}
	got := CalculateDerivedStats(decoded.Character)
	assertValue(t, "Perception", got.Perception, 6)
	assertValue(t, "AC", got.AC, 19)
	assertValue(t, "Fortitude", got.Saves["fortitude"], 8)
	assertValue(t, "Reflex", got.Saves["reflex"], 10)
	assertValue(t, "Will", got.Saves["will"], 5)
	assertValue(t, "Athletics", got.Skills["athletics"], 7)
	if got.Proficiencies.Perception == nil || got.Proficiencies.Perception.Rank != 2 {
		t.Fatalf("effective proficiency DTO missing Perception rank: %+v", got.Proficiencies)
	}
	if got.Proficiencies.Spellcasting == nil || got.Proficiencies.Spellcasting.Rank != 0 {
		t.Fatalf("effective proficiency DTO missing spellcasting baseline: %+v", got.Proficiencies)
	}
	if got.Version != DerivedCalculationVersion {
		t.Fatalf("calculation version = %q", got.Version)
	}
}

func TestCalculateDerivedStatsDoesNotInventMissingAbilityScores(t *testing.T) {
	decoded, err := pf2e.DecodeActor(loadActorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	got := CalculateDerivedStats(decoded.Character)
	if got.Perception.Value != nil || got.Perception.Status != pf2e.CalculationIncomplete {
		t.Fatalf("Perception should remain incomplete without Wisdom: %+v", got.Perception)
	}
	if got.AC.Value != nil || got.AC.Status != pf2e.CalculationIncomplete {
		t.Fatalf("AC should remain incomplete without Dexterity: %+v", got.AC)
	}
	if got.Saves["fortitude"].Value != nil {
		t.Fatalf("Fortitude must not be fabricated: %+v", got.Saves["fortitude"])
	}
}

func TestCalculateIncludesOnlyStaticItemTypeBonuses(t *testing.T) {
	decoded, err := pf2e.DecodeActor(loadActorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	decoded.Character.Statistics.Abilities = map[string]pf2e.AbilityScoreDto{
		"wis": {Modifier: intPointer(1)},
	}
	decoded.Character.Items = append(decoded.Character.Items, pf2e.CharacterItemDto{
		ID: "static-item", Name: "Perception charm", Type: "feat",
		RuleElements: []pf2e.RuleElementDto{{
			Key: "FlatModifier", ModifierType: "item",
			Selector: []byte(`"perception"`), Value: []byte("2"),
		}},
	})
	got := CalculateDerivedStats(decoded.Character)
	assertValue(t, "Perception with item bonus", got.Perception, 8)
}

func assertValue(t *testing.T, name string, stat pf2e.DerivedStatDto, want int) {
	t.Helper()
	if stat.Status != pf2e.CalculationComplete || stat.Value == nil || *stat.Value != want {
		t.Fatalf("%s = %+v, want complete value %d", name, stat, want)
	}
}

func TestItemBonusesOfSameTypeDoNotStack(t *testing.T) {
	decoded, err := pf2e.DecodeActor(loadActorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	decoded.Character.Statistics.Abilities = map[string]pf2e.AbilityScoreDto{
		"str": {Modifier: intPointer(4)},
		"dex": {Modifier: intPointer(4)},
		"con": {Modifier: intPointer(2)},
		"wis": {Modifier: intPointer(1)},
	}
	decoded.Character.Items = append(decoded.Character.Items, pf2e.CharacterItemDto{
		ID: "magic-ac", Name: "AC item bonus", Type: "feat",
		RuleElements: []pf2e.RuleElementDto{{
			Key: "FlatModifier", ModifierType: "item",
			Selector: []byte(`"ac"`), Value: []byte("5"),
		}},
	}, pf2e.CharacterItemDto{
		ID: "magic-save", Name: "Save item bonus", Type: "feat",
		RuleElements: []pf2e.RuleElementDto{{
			Key: "FlatModifier", ModifierType: "item",
			Selector: []byte(`"fortitude"`), Value: []byte("2"),
		}},
	})
	got := CalculateDerivedStats(decoded.Character)
	assertValue(t, "AC with non-stacking item bonus", got.AC, 21)
	assertValue(t, "Fortitude with non-stacking item bonus", got.Saves["fortitude"], 9)
}
