package calculator

import (
	"encoding/json"
	"strings"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

const DerivedCalculationVersion = "pf2e-static-1"

// CalculateDerivedStats calculates only persistent, statically identifiable
// values. It deliberately excludes situational modifiers, conditions,
// temporary effects, predicates and dynamic Rule Element expressions.
func CalculateDerivedStats(character pf2e.CharacterDto) pf2e.CharacterDerivedStatsDto {
	return CalculateDerivedStatsWithFeatureRules(character, nil)
}

// CalculateDerivedStatsWithFeatureRules includes Rule Elements resolved from
// the class feature compendium references in the Actor's class progression.
func CalculateDerivedStatsWithFeatureRules(character pf2e.CharacterDto, featureRules map[string][]pf2e.RuleElementDto) pf2e.CharacterDerivedStatsDto {
	ranks := ResolveProficiencyRanksWithFeatureRules(character, featureRules)
	result := pf2e.CharacterDerivedStatsDto{
		Proficiencies: ranks,
		Saves: make(map[string]pf2e.DerivedStatDto),
		Skills: make(map[string]pf2e.DerivedStatDto),
		Version: DerivedCalculationVersion,
	}

	result.Perception = calculateCheck(character, "perception", "wis", ranks.Perception, ranks, true)
	for save, ability := range map[string]string{
		"fortitude": "con",
		"reflex": "dex",
		"will": "wis",
	} {
		rank, ok := ranks.Saves[save]
		var rankPtr *RankValue
		if ok { rankPtr = &rank }
		result.Saves[save] = calculateSave(character, save, ability, rankPtr, ranks)
	}
	result.AC = calculateAC(character, ranks)

	for skill, rank := range ranks.Skills {
		ability := skillAbility(skill)
		result.Skills[skill] = calculateCheck(character, skill, ability, &rank, ranks, true)
	}
	return result
}

func calculateCheck(character pf2e.CharacterDto, name, ability string, rank *RankValue, ranks ProficiencyRanks, includeItemBonus bool) pf2e.DerivedStatDto {
	if ability == "" {
		return unavailable("No ability mapping is defined for "+name)
	}
	abilityMod, ok := abilityModifier(character, ability)
	if !ok {
		return incomplete("Missing "+strings.ToUpper(ability)+" ability modifier")
	}
	if rank == nil {
		return incomplete("Missing proficiency rank for "+name)
	}
	level, ok := characterLevel(character)
	if rank.Rank > 0 && !ok {
		return incomplete("Missing character level for trained proficiency")
	}
	proficiency := proficiencyBonus(level, rank.Rank)
	breakdown := []pf2e.ModifierBreakdownDto{
		{Source: "ability."+ability, Kind: pf2e.ModifierAbility, Value: abilityMod},
		{Source: strings.Join(rank.Sources, ", "), Kind: pf2e.ModifierProficiency, Value: proficiency},
	}
	bonus := 0
	var sources, warnings []string
	if includeItemBonus {
		bonus, sources, warnings = staticItemBonus(character, name)
		if bonus != 0 {
			breakdown = append(breakdown, pf2e.ModifierBreakdownDto{Source: strings.Join(sources, ", "), Kind: pf2e.ModifierItem, Value: bonus})
		}
	}
	return totalStat(abilityMod+proficiency+bonus, breakdown, mergeWarnings(warnings, ranks.Warnings))
}

func calculateSave(character pf2e.CharacterDto, save, ability string, rank *RankValue, ranks ProficiencyRanks) pf2e.DerivedStatDto {
	if rank == nil {
		return incomplete("Missing "+save+" proficiency rank")
	}
	stat := calculateCheck(character, save, ability, rank, ranks, false)
	if stat.Status != pf2e.CalculationComplete {
		return stat
	}
	resilient, runeSource, runeWarnings := resilientItemBonus(character)
	flatBonus, flatSources, flatWarnings := staticItemBonus(character, save)
	bonusKind := pf2e.ModifierItem
	if resilient > flatBonus {
		flatBonus = resilient
		flatSources = []string{runeSource}
		bonusKind = pf2e.ModifierResilient
	}
	if flatBonus != 0 {
		stat.Value = intPointer(*stat.Value + flatBonus)
		stat.Breakdown = append(stat.Breakdown, pf2e.ModifierBreakdownDto{
			Source: strings.Join(flatSources, ", "), Kind: bonusKind, Value: flatBonus,
		})
	}
	stat.Warnings = mergeWarnings(stat.Warnings, runeWarnings, flatWarnings)
	if len(stat.Warnings) > 0 {
		stat.Status = pf2e.CalculationIncomplete
		stat.Value = nil
	}
	return stat
}

func calculateAC(character pf2e.CharacterDto, ranks ProficiencyRanks) pf2e.DerivedStatDto {
	armor, warning := equippedArmor(character)
	defenseKey := "unarmored"
	baseArmorBonus, potency, dexCap := 0, 0, 99
	breakdown := []pf2e.ModifierBreakdownDto{{Source: "PF2e base AC", Kind: pf2e.ModifierBase, Value: 10}}
	warnings := append([]string{}, ranks.Warnings...)
	if warning != "" {
		warnings = append(warnings, warning)
	}
	if armor != nil {
		if armor.Armor == nil || armor.Armor.Category == nil || armor.Armor.ACBonus == nil || armor.Armor.DexCap == nil {
			return incompleteWithWarnings("Equipped armor is missing category, AC bonus or Dexterity cap", warnings)
		}
		defenseKey = *armor.Armor.Category
		baseArmorBonus = *armor.Armor.ACBonus
		dexCap = *armor.Armor.DexCap
		if armor.Runes != nil && armor.Runes.Potency != nil {
			potency = *armor.Runes.Potency
		}
	}
	rank, ok := ranks.Defenses[defenseKey]
	if !ok {
		return incompleteWithWarnings("Missing "+defenseKey+" armor proficiency rank", warnings)
	}
	level, levelOK := characterLevel(character)
	if rank.Rank > 0 && !levelOK {
		return incompleteWithWarnings("Missing character level for armor proficiency", warnings)
	}
	proficiency := proficiencyBonus(level, rank.Rank)
	breakdown = append(breakdown, pf2e.ModifierBreakdownDto{Source: "class.defenses."+defenseKey, Kind: pf2e.ModifierProficiency, Value: proficiency})
	dex, dexOK := abilityModifier(character, "dex")
	if !dexOK {
		return incompleteWithWarnings("Missing DEX ability modifier", warnings)
	}
	if dex > dexCap {
		dex = dexCap
	}
	breakdown = append(breakdown, pf2e.ModifierBreakdownDto{Source: "ability.dex (armor cap)", Kind: pf2e.ModifierAbility, Value: dex})
	armorItemBonus := baseArmorBonus + potency
	itemBonus, sources, itemWarnings := staticItemBonus(character, "ac")
	selectedItemBonus := armorItemBonus
	if itemBonus > selectedItemBonus {
		selectedItemBonus = itemBonus
		breakdown = append(breakdown, pf2e.ModifierBreakdownDto{Source: strings.Join(sources, ", "), Kind: pf2e.ModifierItem, Value: itemBonus})
	} else if armor != nil {
		breakdown = append(breakdown, pf2e.ModifierBreakdownDto{Source: "Item "+armor.Name+" armor", Kind: pf2e.ModifierArmor, Value: baseArmorBonus})
		if potency != 0 {
			breakdown = append(breakdown, pf2e.ModifierBreakdownDto{Source: "Item "+armor.Name+" potency rune", Kind: pf2e.ModifierRune, Value: potency})
		}
	}
	value := 10 + selectedItemBonus + proficiency + dex
	warnings = mergeWarnings(warnings, itemWarnings)
	return totalStat(value, breakdown, warnings)
}

func equippedArmor(character pf2e.CharacterDto) (*pf2e.CharacterItemDto, string) {
	var equipped *pf2e.CharacterItemDto
	for i := range character.Items {
		item := &character.Items[i]
		if item.Type != "armor" { continue }
		if item.Equipped == nil || (item.Equipped.InSlot == nil && item.Equipped.CarryType == nil) {
			return nil, "Armor equipment state is unknown for "+item.Name
		}
		worn := (item.Equipped.InSlot != nil && *item.Equipped.InSlot) ||
			(item.Equipped.CarryType != nil && *item.Equipped.CarryType == "worn")
		if !worn { continue }
		if equipped != nil { return nil, "More than one armor item is marked as worn" }
		equipped = item
	}
	return equipped, ""
}

func resilientItemBonus(character pf2e.CharacterDto) (int, string, []string) {
	best := 0
	source := ""
	var warnings []string
	for _, item := range character.Items {
		if item.Type != "armor" || item.Runes == nil || item.Runes.Resilient == nil { continue }
		if item.Equipped == nil || (item.Equipped.InSlot == nil && item.Equipped.CarryType == nil) {
			warnings = append(warnings, "Armor equipment state is unknown for "+item.Name+" resilient rune")
			continue
		}
		worn := (item.Equipped.InSlot != nil && *item.Equipped.InSlot) ||
			(item.Equipped.CarryType != nil && *item.Equipped.CarryType == "worn")
		if !worn { continue }
		if *item.Runes.Resilient > best {
			best = *item.Runes.Resilient
			source = "Item "+item.Name+" resilient rune"
		}
	}
	return best, source, warnings
}

func staticItemBonus(character pf2e.CharacterDto, selector string) (int, []string, []string) {
	best := 0
	var sources, warnings []string
	for _, item := range character.Items {
		for _, rule := range item.RuleElements {
			if rule.Key != "FlatModifier" || rule.ModifierType != "item" || !selectorMatches(rule.Selector, selector) {
				continue
			}
			if hasPredicate(rule.Predicate) {
				warnings = append(warnings, "Conditional item bonus is not evaluated: "+item.Name)
				continue
			}
			if !alwaysActiveItem(item.Type) && !itemEquipped(item) {
				if item.Equipped == nil {
					warnings = append(warnings, "Equipment state is unknown for item bonus: "+item.Name)
				}
				continue
			}
			var value int
			if json.Unmarshal(rule.Value, &value) != nil || value < 0 {
				warnings = append(warnings, "Non-static or negative item bonus is not evaluated: "+item.Name)
				continue
			}
			if value > best {
				best = value
				sources = []string{"Item "+item.Name}
			} else if value == best && value > 0 {
				sources = append(sources, "Item "+item.Name)
			}
		}
	}
	return best, sources, warnings
}

func selectorMatches(raw json.RawMessage, selector string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == selector || (selector == "fortitude" || selector == "reflex" || selector == "will") && single == "saving-throw"
	}
	var values []string
	if json.Unmarshal(raw, &values) != nil { return false }
	for _, value := range values {
		if value == selector || (selector == "fortitude" || selector == "reflex" || selector == "will") && value == "saving-throw" {
			return true
		}
	}
	return false
}

func alwaysActiveItem(itemType string) bool {
	switch itemType {
	case "feat", "action", "class", "ancestry", "heritage", "background":
		return true
	default:
		return false
	}
}

func itemEquipped(item pf2e.CharacterItemDto) bool {
	if item.Equipped == nil { return false }
	return (item.Equipped.InSlot != nil && *item.Equipped.InSlot) ||
		(item.Equipped.CarryType != nil && (*item.Equipped.CarryType == "worn" || *item.Equipped.CarryType == "held"))
}

func skillAbility(skill string) string {
	if strings.HasPrefix(skill, "lore-") || strings.HasSuffix(skill, "-lore") { return "int" }
	abilities := map[string]string{
		"acrobatics": "dex", "arcana": "int", "athletics": "str", "crafting": "int",
		"deception": "cha", "diplomacy": "cha", "intimidation": "cha", "medicine": "wis",
		"nature": "wis", "occultism": "int", "performance": "cha", "religion": "wis",
		"society": "int", "stealth": "dex", "survival": "wis", "thievery": "dex",
	}
	return abilities[skill]
}

func abilityModifier(character pf2e.CharacterDto, ability string) (int, bool) {
	entry, ok := character.Statistics.Abilities[ability]
	if !ok { return 0, false }
	if entry.Modifier != nil { return *entry.Modifier, true }
	if entry.Score != nil {
		delta := *entry.Score - 10
		modifier := delta / 2
		if delta < 0 && delta%2 != 0 { modifier-- }
		return modifier, true
	}
	return 0, false
}

func characterLevel(character pf2e.CharacterDto) (int, bool) {
	if character.Progression.Level == nil { return 0, false }
	return *character.Progression.Level, true
}

func proficiencyBonus(level, rank int) int {
	if rank <= 0 { return 0 }
	return level + 2 + (rank-1)*2
}

func totalStat(value int, breakdown []pf2e.ModifierBreakdownDto, warnings []string) pf2e.DerivedStatDto {
	status := pf2e.CalculationComplete
	if len(warnings) > 0 { status = pf2e.CalculationIncomplete }
	if status != pf2e.CalculationComplete {
		return pf2e.DerivedStatDto{Status: status, Breakdown: breakdown, Warnings: warnings}
	}
	return pf2e.DerivedStatDto{Value: intPointer(value), Status: status, Breakdown: breakdown}
}

func unavailable(warning string) pf2e.DerivedStatDto {
	return pf2e.DerivedStatDto{Status: pf2e.CalculationUnavailable, Warnings: []string{warning}}
}

func incomplete(warning string) pf2e.DerivedStatDto {
	return pf2e.DerivedStatDto{Status: pf2e.CalculationIncomplete, Warnings: []string{warning}}
}

func incompleteWithWarnings(warning string, warnings []string) pf2e.DerivedStatDto {
	return pf2e.DerivedStatDto{Status: pf2e.CalculationIncomplete, Warnings: mergeWarnings(warnings, []string{warning})}
}

func mergeWarnings(groups ...[]string) []string {
	var result []string
	for _, group := range groups {
		for _, warning := range group {
			if warning == "" { continue }
			found := false
			for _, existing := range result { if existing == warning { found = true; break } }
			if !found { result = append(result, warning) }
		}
	}
	return result
}

func intPointer(value int) *int { return &value }

