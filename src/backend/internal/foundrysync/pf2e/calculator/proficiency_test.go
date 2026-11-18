package calculator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

func loadActorFixture(t *testing.T) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok { t.Fatal("runtime.Caller failed") }
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "..", "docs", "foundry", "mxgc0DRWC8bA9ruI-20261118T073146Z-actor.json")
	data, err := os.ReadFile(path)
	if err != nil { t.Fatalf("read Actor fixture: %v", err) }
	return data
}

func TestResolveFighterClassBaselineAndSelectedSkill(t *testing.T) {
	decoded, err := pf2e.DecodeActor(loadActorFixture(t))
	if err != nil { t.Fatal(err) }
	got := ResolveProficiencyRanks(decoded.Character)
	if got.Perception == nil || got.Perception.Rank != 2 {
		t.Fatalf("perception rank = %+v, want 2", got.Perception)
	}
	for key, want := range map[string]int{"fortitude": 2, "reflex": 2, "will": 1} {
		if rank := got.Saves[key]; rank.Rank != want {
			t.Fatalf("%s rank = %+v, want %d", key, rank, want)
		}
	}
	for key, want := range map[string]int{"heavy": 1, "light": 1, "medium": 1, "unarmored": 1} {
		if rank := got.Defenses[key]; rank.Rank != want {
			t.Fatalf("%s defense rank = %+v, want %d", key, rank, want)
		}
	}
	for key, want := range map[string]int{"advanced": 1, "martial": 2, "simple": 2, "unarmed": 2} {
		if rank := got.Attacks[key]; rank.Rank != want {
			t.Fatalf("%s attack rank = %+v, want %d", key, rank, want)
		}
	}
	if got.Spellcasting == nil || got.Spellcasting.Rank != 0 {
		t.Fatalf("spellcasting rank = %+v, want 0", got.Spellcasting)
	}
	if rank := got.Skills["athletics"]; rank.Rank != 1 {
		t.Fatalf("selected fighter skill rank = %+v, want trained (1)", rank)
	}
	if !got.Complete || len(got.Warnings) != 0 {
		t.Fatalf("unexpected incomplete result: %+v", got.Warnings)
	}
}

func TestResolveOnlySupportedStaticAndBracketedUpgrades(t *testing.T) {
	character := pf2e.CharacterDto{
		Progression: pf2e.CharacterProgressionDto{Level: intPointer(8)},
		Build: pf2e.CharacterBuildDto{
			ClassProficiencies: &pf2e.ClassProficiencyProfileDto{
				PerceptionRank: intPointer(1),
				RuleElements: []pf2e.RuleElementDto{
					{Key: "ActiveEffectLike", Mode: "upgrade", Path: "system.attributes.perception.rank", Value: []byte("2")},
					{Key: "ActiveEffectLike", Mode: "upgrade", Path: "system.saves.will.rank", Value: []byte(`{"brackets":[{"start":1,"end":6,"value":2},{"start":7,"value":3}]}`)},
				{Key: "ActiveEffectLike", Mode: "upgrade", Path: "system.proficiencies.spellcasting.arcane.rank", Value: []byte("2")},
				},
			},
		},
	}
	got := ResolveProficiencyRanks(character)
	if got.Perception == nil || got.Perception.Rank != 2 { t.Fatalf("perception upgrade not applied: %+v", got.Perception) }
	if got.Saves["will"].Rank != 3 { t.Fatalf("level bracket not applied: %+v", got.Saves["will"]) }
	if got.SpellcastingTraditions["arcane"].Rank != 2 { t.Fatalf("tradition spellcasting rank not applied: %+v", got.SpellcastingTraditions) }
	if len(got.Warnings) != 0 { t.Fatalf("unexpected warnings: %v", got.Warnings) }
}

func TestResolveConditionalRuleIsReportedNotExecuted(t *testing.T) {
	character := pf2e.CharacterDto{
		Build: pf2e.CharacterBuildDto{ClassProficiencies: &pf2e.ClassProficiencyProfileDto{
			SavingThrows: map[string]int{"will": 1},
			RuleElements: []pf2e.RuleElementDto{{
				Key: "ActiveEffectLike", Mode: "upgrade", Path: "system.saves.will.rank",
				Value: []byte("2"), Predicate: []byte(`["feature:some-condition"]`),
			}},
		}},
	}
	got := ResolveProficiencyRanks(character)
	if got.Saves["will"].Rank != 1 { t.Fatalf("conditional upgrade should not be applied: %+v", got.Saves["will"]) }
	if got.Complete || len(got.Warnings) == 0 { t.Fatalf("conditional rule should mark result incomplete: %+v", got) }
}



func TestResolveProficiencyFromResolvedClassFeatureReference(t *testing.T) {
	character := pf2e.CharacterDto{
		Progression: pf2e.CharacterProgressionDto{Level: intPointer(3)},
		Build: pf2e.CharacterBuildDto{
			ClassProficiencies: &pf2e.ClassProficiencyProfileDto{
				PerceptionRank: intPointer(1),
				Features: []pf2e.ClassFeatureReferenceDto{{
					UUID: "Compendium.pf2e.classfeatures.Item.feature1",
					Name: "Perception Mastery", Level: intPointer(3),
				}},
			},
		},
	}
	withoutResolvedRules := ResolveProficiencyRanks(character)
	if withoutResolvedRules.Complete || len(withoutResolvedRules.Warnings) == 0 {
		t.Fatalf("missing compendium feature rules should be reported: %+v", withoutResolvedRules)
	}
	resolved := map[string][]pf2e.RuleElementDto{
		"Compendium.pf2e.classfeatures.Item.feature1": {{
			Key: "ActiveEffectLike", Mode: "upgrade",
			Path: "system.attributes.perception.rank", Value: []byte("2"),
		}},
	}
	got := ResolveProficiencyRanksWithFeatureRules(character, resolved)
	if got.Perception == nil || got.Perception.Rank != 2 || !got.Complete || len(got.Warnings) != 0 {
		t.Fatalf("resolved feature did not upgrade perception: %+v", got)
	}
}


type fakeCompendiumResolver map[string][]byte

func (f fakeCompendiumResolver) ResolveCompendiumItem(_ context.Context, uuid string) ([]byte, error) {
	payload, ok := f[uuid]
	if !ok { return nil, fmt.Errorf("not found: %s", uuid) }
	return payload, nil
}

func TestResolverLoadsClassFeatureByCompendiumUUID(t *testing.T) {
	const featureUUID = "Compendium.pf2e.classfeatures.Item.feature1"
	character := pf2e.CharacterDto{
		Progression: pf2e.CharacterProgressionDto{Level: intPointer(3)},
		Statistics: pf2e.CharacterStatisticsInputDto{
			Abilities: map[string]pf2e.AbilityScoreDto{"wis": {Modifier: intPointer(1)}},
		},
		Build: pf2e.CharacterBuildDto{
			ClassProficiencies: &pf2e.ClassProficiencyProfileDto{
				PerceptionRank: intPointer(1),
				Features: []pf2e.ClassFeatureReferenceDto{{
					UUID: featureUUID, Name: "Perception Mastery", Level: intPointer(3),
				}},
			},
		},
	}
	resolver := fakeCompendiumResolver{
		featureUUID: []byte(`{"type":"entity-result","data":{"_id":"feature1","name":"Perception Mastery","type":"feat","system":{"rules":[{"key":"ActiveEffectLike","mode":"upgrade","path":"system.attributes.perception.rank","value":2}]}}}`),
	}
	got := CalculateDerivedStatsWithResolver(context.Background(), character, resolver)
	if got.Proficiencies.Perception == nil || got.Proficiencies.Perception.Rank != 2 || !got.Proficiencies.Complete {
		t.Fatalf("resolved class feature rank missing: %+v", got.Proficiencies)
	}
	if got.Perception.Value == nil || *got.Perception.Value != 8 {
		t.Fatalf("derived Perception should include resolved Expert rank: %+v", got.Perception)
	}
}
