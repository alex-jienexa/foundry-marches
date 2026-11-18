package calculator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

// RankValue and ProficiencyRanks are public DTOs from the PF2e boundary
// package; this calculator only populates their effective, auditable values.
type RankValue = pf2e.CalculatedProficiencyRankDto
type ProficiencyRanks = pf2e.CharacterProficiencyRanksDto

// ResolveProficiencyRanks starts from the class progression baseline, overlays
// explicitly stored Actor ranks, and applies only simple, static
// ActiveEffectLike upgrade rules found on the class or owned Actor Items.
//
// Compendium feature references are not executable rules. A reference whose
// level has been reached but whose document is not present among Actor Items
// is reported as unresolved, rather than guessed.
func ResolveProficiencyRanks(character pf2e.CharacterDto) ProficiencyRanks {
	return ResolveProficiencyRanksWithFeatureRules(character, nil)
}

// CompendiumItemResolver is implemented by the read-only Foundry/Relay
// adapter. It must return the Item document for the exact compendium UUID.
type CompendiumItemResolver interface {
	ResolveCompendiumItem(ctx context.Context, uuid string) ([]byte, error)
}

// ResolveProficiencyRanksWithResolver fetches class-feature documents for
// progression entries already reached by the character, decodes their Rule
// Elements, and then resolves the effective proficiency ranks.
func ResolveProficiencyRanksWithResolver(ctx context.Context, character pf2e.CharacterDto, resolver CompendiumItemResolver) ProficiencyRanks {
	featureRules, loadWarnings := LoadClassFeatureRuleElements(ctx, character, resolver)
	result := ResolveProficiencyRanksWithFeatureRules(character, featureRules)
	result.Warnings = mergeWarnings(result.Warnings, loadWarnings)
	result.Complete = result.Complete && len(result.Warnings) == 0
	return result
}

// LoadClassFeatureRuleElements resolves only reached class features that are
// not already embedded as Actor Items. A missing resolver or failed lookup is
// retained as a warning; it never silently produces a lower rank.
func LoadClassFeatureRuleElements(ctx context.Context, character pf2e.CharacterDto, resolver CompendiumItemResolver) (map[string][]pf2e.RuleElementDto, []string) {
	resolved := make(map[string][]pf2e.RuleElementDto)
	var warnings []string
	class := character.Build.ClassProficiencies
	if class == nil {
		return resolved, warnings
	}
	level := 0
	if character.Progression.Level != nil {
		level = *character.Progression.Level
	}
	for _, feature := range class.Features {
		if feature.Level == nil || level < *feature.Level || hasCompendiumSource(character.Items, feature.UUID) {
			continue
		}
		if resolver == nil {
			warnings = append(warnings, "no compendium resolver configured for "+feature.UUID)
			continue
		}
		payload, err := resolver.ResolveCompendiumItem(ctx, feature.UUID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("resolve class feature %s: %v", feature.UUID, err))
			continue
		}
		decoded, err := pf2e.DecodeItemResponse(payload)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("decode class feature %s: %v", feature.UUID, err))
			continue
		}
		parts := strings.Split(feature.UUID, ".")
		expectedID := parts[len(parts)-1]
		if decoded.Foundry.ID != expectedID {
			warnings = append(warnings, fmt.Sprintf("class feature UUID %s resolved to unexpected Item ID %s", feature.UUID, decoded.Foundry.ID))
			continue
		}
		resolved[feature.UUID] = decoded.Character.RuleElements
	}
	return resolved, warnings
}

// ResolveProficiencyRanksWithFeatureRules additionally accepts Rule Elements
// fetched from class-feature compendium UUIDs. The caller owns the read-only
// compendium lookup; this package only interprets the supplied declarative DTOs.
func ResolveProficiencyRanksWithFeatureRules(character pf2e.CharacterDto, featureRules map[string][]pf2e.RuleElementDto) ProficiencyRanks {
	result := ProficiencyRanks{
		Saves: make(map[string]RankValue), Defenses: make(map[string]RankValue),
		Attacks: make(map[string]RankValue), SpellcastingTraditions: make(map[string]RankValue),
		Skills: make(map[string]RankValue),
		Complete: true,
	}
	level := 0
	if character.Progression.Level != nil {
		level = *character.Progression.Level
	}

	class := character.Build.ClassProficiencies
	if class != nil {
		if class.PerceptionRank != nil {
			upgrade(&result.Perception, *class.PerceptionRank, sourceClass(character, "perception"))
		}
		for key, rank := range class.SavingThrows {
			upgradeMap(result.Saves, key, rank, sourceClass(character, "savingThrows."+key))
		}
		for key, rank := range class.Defenses {
			upgradeMap(result.Defenses, key, rank, sourceClass(character, "defenses."+key))
		}
		for key, rank := range class.Attacks {
			upgradeMap(result.Attacks, key, rank, sourceClass(character, "attacks."+key))
		}
		if class.SpellcastingRank != nil {
			upgrade(&result.Spellcasting, *class.SpellcastingRank, sourceClass(character, "spellcasting"))
		}
	}

	// Explicit current Actor values are at least as authoritative as class
	// baselines. The max/upgrade semantics also make repeated application safe.
	if p := character.Statistics.Proficiencies.Perception; p != nil && p.Rank != nil {
		upgrade(&result.Perception, *p.Rank, nonEmpty(p.Source, "actor perception"))
	}
	for key, value := range character.Statistics.Proficiencies.Saves {
		if value.Rank != nil {
			upgradeMap(result.Saves, key, *value.Rank, nonEmpty(value.Source, "actor save "+key))
		}
	}
	for key, value := range character.Statistics.Proficiencies.Defenses {
		if value.Rank != nil {
			upgradeMap(result.Defenses, key, *value.Rank, nonEmpty(value.Source, "actor defense "+key))
		}
	}
	for key, skill := range character.Skills.Standard {
		if skill.Rank != nil {
			upgradeMap(result.Skills, key, *skill.Rank, "Actor.system.skills."+key)
		}
	}
	for key, skill := range character.Skills.Lore {
		if skill.Rank != nil {
			upgradeMap(result.Skills, key, *skill.Rank, "Actor.system.skills."+key)
		}
	}

	// Apply class-level and owned-item rules. Other Rule Element kinds remain
	// preserved in DTOs but are outside this calculator's supported subset.
	if class != nil {
		applyRules(&result, class.RuleElements, sourceClass(character, "rules"), level)
	}
	for _, item := range character.Items {
		applyRules(&result, item.RuleElements, "Actor Item "+item.Name, level)
	}

	if class != nil {
		for _, feature := range class.Features {
			if feature.Level == nil || level < *feature.Level {
				continue
			}
			if hasCompendiumSource(character.Items, feature.UUID) {
				continue
			}
			if rules, ok := featureRules[feature.UUID]; ok {
				applyRules(&result, rules, "Class feature "+feature.Name, level)
				continue
			}
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("class feature %q (%s, level %d) is not present as an Actor Item and its compendium rules were not resolved",
					feature.Name, feature.UUID, *feature.Level))
		}
	}

	result.Complete = len(result.Warnings) == 0 && hasAnyRank(result)
	return result
}

func sourceClass(character pf2e.CharacterDto, field string) string {
	name := "class"
	if character.Build.Class != nil && character.Build.Class.Name != "" {
		name = character.Build.Class.Name
	}
	return name + ".system." + field
}

func nonEmpty(value, fallback string) string {
	if value != "" { return value }
	return fallback
}

func hasAnyRank(r ProficiencyRanks) bool {
	return r.Perception != nil || r.Spellcasting != nil || len(r.Saves)+len(r.Defenses)+len(r.Attacks)+len(r.SpellcastingTraditions)+len(r.Skills) > 0
}

func hasCompendiumSource(items []pf2e.CharacterItemDto, uuid string) bool {
	for _, item := range items {
		if item.CompendiumSource != nil && *item.CompendiumSource == uuid {
			return true
		}
	}
	return false
}

func upgrade(current **RankValue, rank int, source string) {
	if *current == nil || rank > (*current).Rank {
		*current = &RankValue{Rank: rank, Sources: []string{source}}
		return
	}
	if rank == (*current).Rank {
		(*current).Sources = appendUnique((*current).Sources, source)
	}
}

func upgradeMap(target map[string]RankValue, key string, rank int, source string) {
	current, exists := target[key]
	if !exists || rank > current.Rank {
		target[key] = RankValue{Rank: rank, Sources: []string{source}}
		return
	}
	if rank == current.Rank {
		current.Sources = appendUnique(current.Sources, source)
		target[key] = current
	}
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value { return values }
	}
	return append(values, value)
}

func applyRules(result *ProficiencyRanks, rules []pf2e.RuleElementDto, source string, level int) {
	choices := make(map[string]string)
	for _, rule := range rules {
		if rule.Key != "ChoiceSet" || rule.Flag == "" || len(rule.Selection) == 0 {
			continue
		}
		var selection string
		if json.Unmarshal(rule.Selection, &selection) == nil && selection != "" {
			choices[rule.Flag] = selection
		}
	}
	for _, rule := range rules {
		if rule.Key != "ActiveEffectLike" || !strings.Contains(rule.Path, ".rank") {
			continue
		}
		if rule.Mode != "upgrade" {
			if strings.Contains(rule.Path, ".rank") {
				result.Warnings = append(result.Warnings, source+": unsupported ActiveEffectLike mode "+rule.Mode+" for "+rule.Path)
			}
			continue
		}
		path, ok := resolvePath(rule.Path, choices)
		if !ok {
			result.Warnings = append(result.Warnings, source+": unresolved ActiveEffectLike path "+rule.Path)
			continue
		}
		if hasPredicate(rule.Predicate) {
			result.Warnings = append(result.Warnings, source+": conditional ActiveEffectLike is not evaluated: "+path)
			continue
		}
		rank, ok := rankValueForLevel(rule.Value, level)
		if !ok {
			result.Warnings = append(result.Warnings, source+": unsupported ActiveEffectLike value for "+path)
			continue
		}
		if rank < 0 || rank > 4 {
			result.Warnings = append(result.Warnings, source+": proficiency rank outside 0..4 for "+path)
			continue
		}
		if !applyRankPath(result, path, rank, source+" ActiveEffectLike") {
			result.Warnings = append(result.Warnings, source+": unsupported proficiency path "+path)
		}
	}
}

func resolvePath(path string, choices map[string]string) (string, bool) {
	for strings.Contains(path, "{item|flags.") {
		start := strings.Index(path, "{item|flags.")
		endOffset := strings.Index(path[start:], "}")
		if endOffset < 0 { return "", false }
		end := start + endOffset
		token := path[start+len("{item|flags.") : end]
		parts := strings.Split(token, ".")
		if len(parts) < 3 || parts[len(parts)-2] != "rulesSelections" {
			return "", false
		}
		selection, ok := choices[parts[len(parts)-1]]
		if !ok { return "", false }
		path = path[:start] + selection + path[end+1:]
	}
	return path, true
}

func hasPredicate(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) == nil && len(values) == 0 {
		return false
	}
	return true
}

func rankValueForLevel(raw json.RawMessage, level int) (int, bool) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" { return 0, false }
	var number int
	if json.Unmarshal(raw, &number) == nil { return number, true }
	var bracketSet struct {
		Brackets []struct {
			Start *int `json:"start"`
			End   *int `json:"end"`
			Value *int `json:"value"`
		} `json:"brackets"`
	}
	if json.Unmarshal(raw, &bracketSet) != nil || len(bracketSet.Brackets) == 0 {
		return 0, false
	}
	for _, bracket := range bracketSet.Brackets {
		start := 1
		if bracket.Start != nil { start = *bracket.Start }
		if level < start || (bracket.End != nil && level > *bracket.End) || bracket.Value == nil {
			continue
		}
		return *bracket.Value, true
	}
	return 0, false
}

func applyRankPath(result *ProficiencyRanks, path string, rank int, source string) bool {
	parts := strings.Split(path, ".")
	if len(parts) < 4 || parts[len(parts)-1] != "rank" {
		return false
	}
	switch {
	case path == "system.attributes.perception.rank":
		upgrade(&result.Perception, rank, source)
	case len(parts) == 4 && parts[0] == "system" && parts[1] == "saves":
		upgradeMap(result.Saves, parts[2], rank, source)
	case len(parts) == 4 && parts[0] == "system" && parts[1] == "skills":
		upgradeMap(result.Skills, parts[2], rank, source)
	case len(parts) == 5 && parts[0] == "system" && parts[1] == "proficiencies" && parts[2] == "defenses":
		upgradeMap(result.Defenses, parts[3], rank, source)
	case len(parts) == 5 && parts[0] == "system" && parts[1] == "proficiencies" && parts[2] == "attacks":
		upgradeMap(result.Attacks, parts[3], rank, source)
	case path == "system.proficiencies.spellcasting.rank":
		upgrade(&result.Spellcasting, rank, source)
	case len(parts) == 5 && parts[0] == "system" && parts[1] == "proficiencies" && parts[2] == "spellcasting":
		upgradeMap(result.SpellcastingTraditions, parts[3], rank, source)
	default:
		return false
	}
	return true
}

