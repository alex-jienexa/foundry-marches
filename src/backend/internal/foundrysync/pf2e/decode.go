package pf2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrInvalidPayload   = errors.New("invalid Foundry payload")
	ErrMissingActorID   = errors.New("Foundry Actor is missing _id")
	ErrMissingActorType = errors.New("Foundry Actor is missing type")
	ErrMissingItems     = errors.New("Foundry Actor is missing a valid items array")
)

// DecodeRelayResponse unwraps the Relay entity-result envelope and decodes
// its data. Bare Actor JSON is also accepted for fixture/import workflows.
func DecodeRelayResponse(payload []byte) (*DecodedActor, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("%w: decode response: %v", ErrInvalidPayload, err)
	}
	if data, ok := envelope["data"]; ok && len(data) > 0 && !bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		var kind string
		_ = json.Unmarshal(envelope["type"], &kind)
		if kind != "" && kind != "entity-result" {
			return nil, fmt.Errorf("%w: unexpected Relay response type %q", ErrInvalidPayload, kind)
		}
		return DecodeActor(data)
	}
	return DecodeActor(payload)
}

// DecodeActor maps a complete Actor document into the application DTO.
// RawSnapshot retains every source field, including fields unknown to this version.
func DecodeActor(payload []byte) (*DecodedActor, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("%w: decode Actor: %v", ErrInvalidPayload, err)
	}
	var actor FoundryActorDto
	if err := json.Unmarshal(payload, &actor); err != nil {
		return nil, fmt.Errorf("%w: decode Actor fields: %v", ErrInvalidPayload, err)
	}
	if actor.ID == "" {
		return nil, ErrMissingActorID
	}
	if actor.Type == "" {
		return nil, ErrMissingActorType
	}

	itemsRaw, ok := root["items"]
	if !ok || len(itemsRaw) == 0 || bytes.Equal(bytes.TrimSpace(itemsRaw), []byte("null")) {
		return nil, ErrMissingItems
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(itemsRaw, &rawItems); err != nil {
		return nil, fmt.Errorf("%w: Actor.items must be an array: %v", ErrMissingItems, err)
	}

	normalizedItems := make([]CharacterItemDto, 0, len(rawItems))
	actor.Items = make([]FoundryItemDto, 0, len(rawItems))
	seenIDs := make(map[string]struct{}, len(rawItems))
	for i, raw := range rawItems {
		item, err := DecodeItem(raw)
		if err != nil {
			return nil, fmt.Errorf("Actor.items[%d]: %w", i, err)
		}
		if _, exists := seenIDs[item.Foundry.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate Actor Item _id %q", ErrInvalidPayload, item.Foundry.ID)
		}
		seenIDs[item.Foundry.ID] = struct{}{}
		actor.Items = append(actor.Items, item.Foundry)
		normalizedItems = append(normalizedItems, item.Character)
	}
	enrichSpellcastingItems(normalizedItems, actor.Items)

	system := actor.System
	details := objectAt(system, "details")
	xp := objectAt(details, "xp")
	hp := objectAt(objectAt(system, "attributes"), "hp")
	hero := objectAt(objectAt(system, "resources"), "heroPoints")
	bio := objectAt(details, "biography")
	migration := objectAt(system, "_migration")

	character := CharacterDto{
		SchemaVersion: 1,
		Identity: CharacterIdentityDto{
			ID: actor.ID, UUID: "Actor." + actor.ID, Name: actor.Name,
			Type: actor.Type, Image: actor.Image,
		},
		Source: CharacterSourceDto{
			Platform: "foundry-vtt", CoreVersion: actor.Stats.CoreVersion,
			SystemID: actor.Stats.SystemID, SystemVersion: actor.Stats.SystemVersion,
			MigrationVersion: floatAt(migration, "version"),
		},
		Progression: CharacterProgressionDto{
			Level: intAt(details, "level", "value"),
			Experience: ExperienceDto{
				Value: intAt(xp, "value"), Min: intAt(xp, "min"), Max: intAt(xp, "max"),
			},
		},
		Attributes: decodeAttributes(system, hp, actor.Items, intAt(details, "level", "value")),
		Statistics: decodeStatistics(system, actor.Items, intAt(details, "level", "value")),
		Resources:  CharacterResourcesDto{HeroPoints: decodeResource(hero)},
		Skills:     decodeSkillsWithTraining(system["skills"], actor.Items),
		Items:      normalizedItems,
		Biography:  decodeBiography(bio, objectAt(details, "languages"), details),
	}
	character.Build = buildReferences(normalizedItems)
	character.Build.ClassProficiencies = decodeClassProficiencies(actor.Items)
	character.Build.AncestryFeatures = decodeAncestryFeatureReferences(actor.Items)
	character.Build.KeyAbility = decodeKeyAbility(details, actor.Items)
	logActorDecodeDiagnostics(actor, character, itemsRaw, actor.Items)

	return &DecodedActor{
		Character:   character,
		RawSnapshot: cloneRaw(payload),
	}, nil
}

// DecodeItemResponse unwraps a Relay entity-result envelope when present,
// then decodes the returned Item document. Bare Item JSON is also accepted.
func DecodeItemResponse(payload []byte) (*DecodedItem, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("%w: decode Item response: %v", ErrInvalidPayload, err)
	}
	if data, ok := envelope["data"]; ok && len(data) > 0 && !bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		var kind string
		_ = json.Unmarshal(envelope["type"], &kind)
		if kind != "" && kind != "entity-result" {
			return nil, fmt.Errorf("%w: unexpected Relay Item response type %q", ErrInvalidPayload, kind)
		}
		return DecodeItem(data)
	}
	return DecodeItem(payload)
}

// DecodeItem returns both a normalized projection and the source Item document.
func DecodeItem(payload []byte) (*DecodedItem, error) {
	var item FoundryItemDto
	if err := json.Unmarshal(payload, &item); err != nil {
		return nil, fmt.Errorf("%w: decode Item: %v", ErrInvalidPayload, err)
	}
	if item.ID == "" {
		return nil, fmt.Errorf("%w: Item is missing _id", ErrInvalidPayload)
	}
	if item.Type == "" {
		return nil, fmt.Errorf("%w: Item %s is missing type", ErrInvalidPayload, item.ID)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("%w: decode Item object: %v", ErrInvalidPayload, err)
	}
	system := objectAt(root, "system")
	normalized := CharacterItemDto{
		ID: item.ID, Name: item.Name, Type: item.Type, Image: item.Image,
		Slug: stringAt(system, "slug"), Category: categoryForItem(item.Type, system),
		Skill: decodeSkillAssociation(system), Cantrip: isCantripSpell(system),
		SystemCategory:      stringAt(system, "category"),
		ActionType:          stringAt(objectAt(system, "actionType"), "value"),
		Quantity:            intAt(system, "quantity"),
		Level:               firstInt(intAt(system, "level", "value"), intAt(system, "level")),
		Equipped:            decodeEquipped(objectAt(system, "equipped")),
		ContainerID:         stringAt(system, "containerId"),
		Usage:               stringAt(objectAt(system, "usage"), "value"),
		SpellcastingEntryID: firstString(stringAt(system, "location"), stringAt(objectAt(system, "location"), "value")),
		Prepared:            boolAt(system, "prepared"),
		Armor:               decodeArmorData(system),
		Runes:               decodeItemRunes(system),
		CompendiumSource:    stringAt(objectAt(root, "_stats"), "compendiumSource"),
		RuleElements:        decodeRuleElements(system, item.Flags),
	}
	return &DecodedItem{
		Foundry: item, Character: normalized, RawSnapshot: cloneRaw(payload),
	}, nil
}

func isCantripSpell(system map[string]json.RawMessage) bool {
	category := firstString(
		stringAt(objectAt(system, "category"), "value"),
		stringAt(system, "category"),
	)
	if category != nil && strings.EqualFold(strings.TrimSpace(*category), "cantrip") {
		return true
	}
	for _, trait := range stringsAt(objectAt(system, "traits"), "value") {
		if strings.EqualFold(strings.TrimSpace(trait), "cantrip") {
			return true
		}
	}
	return false
}

func categoryForItem(kind string, system map[string]json.RawMessage) CharacterItemCategory {
	category := categoryFor(kind)
	if category == ItemCategoryOther && (system["quantity"] != nil || system["bulk"] != nil || system["equipped"] != nil || system["usage"] != nil || system["containerId"] != nil) {
		return ItemCategoryEquipment
	}
	return category
}

func categoryFor(kind string) CharacterItemCategory {
	switch kind {
	case "weapon", "armor", "consumable", "treasure", "equipment", "backpack", "shield":
		return ItemCategoryEquipment
	case "feat", "action":
		return ItemCategoryAbility
	case "ancestry", "heritage", "background", "class":
		return ItemCategoryIdentity
	default:
		return ItemCategoryOther
	}
}

func buildReferences(items []CharacterItemDto) CharacterBuildDto {
	var result CharacterBuildDto
	for i := range items {
		item := items[i]
		ref := &CharacterItemReferenceDto{ID: item.ID, Name: item.Name, Type: item.Type, Slug: item.Slug, Image: item.Image}
		switch item.Type {
		case "ancestry":
			result.Ancestry = ref
		case "heritage":
			result.Heritage = ref
		case "background":
			result.Background = ref
		case "class":
			result.Class = ref
		}
	}
	return result
}

func decodeHP(raw map[string]json.RawMessage) *HitPointsDto {
	if raw == nil {
		return nil
	}
	return &HitPointsDto{Current: intAt(raw, "value"), Temp: intAt(raw, "temp"), Max: intAt(raw, "max")}
}

func decodeAttributes(system map[string]json.RawMessage, hpRaw map[string]json.RawMessage, items []FoundryItemDto, level *int) CharacterAttributesDto {
	hp := decodeHP(hpRaw)
	if hp == nil {
		// An absent Actor HP object does not block derivation when enough
		// independent inputs exist, but must remain nil when no result can
		// be derived.
		logCharacterDiagnostic("hp source=actor object=<absent>; attempting partial derivation")
		maxHP := deriveMaximumHP(system, items, level)
		if maxHP == nil {
			return CharacterAttributesDto{
				HP:    nil,
				Dying: decodeDying(objectAt(objectAt(system, "attributes"), "dying")),
			}
		}
		hp = &HitPointsDto{Max: maxHP}
	} else if hp.Max == nil {
		logCharacterDiagnostic("hp source=actor max=<absent>; attempting partial derivation")
		hp.Max = deriveMaximumHP(system, items, level)
	} else {
		logCharacterDiagnostic("hp source=actor max=%s current=%s temp=%s", diagnosticIntPointer(hp.Max), diagnosticIntPointer(hp.Current), diagnosticIntPointer(hp.Temp))
	}
	attributes := objectAt(system, "attributes")
	return CharacterAttributesDto{
		HP:    hp,
		Dying: decodeDying(objectAt(attributes, "dying")),
	}
}

func decodeDying(raw map[string]json.RawMessage) *DyingConditionDto {
	if raw == nil {
		return nil
	}
	return &DyingConditionDto{Current: intAt(raw, "value"), Max: intAt(raw, "max")}
}

func characterDiagnosticsEnabled() bool {
	value := strings.TrimSpace(os.Getenv("FOUNDRY_CHARACTER_DIAGNOSTICS"))
	return strings.EqualFold(value, "1") || strings.EqualFold(value, "true")
}

func logCharacterDiagnostic(format string, args ...any) {
	if characterDiagnosticsEnabled() {
		log.Printf("[character-diagnostics] "+format, args...)
	}
}

func logActorDecodeDiagnostics(actor FoundryActorDto, character CharacterDto, rawItems json.RawMessage, items []FoundryItemDto) {
	if !characterDiagnosticsEnabled() {
		return
	}
	var className, ancestryName, backgroundName string
	var classHP, ancestryHP *int
	var classItem, ancestryItem, backgroundItem *FoundryItemDto
	for _, item := range items {
		switch item.Type {
		case "class":
			className, classHP, classItem = item.Name, intAt(item.System, "hp"), &item
		case "ancestry":
			ancestryName, ancestryHP, ancestryItem = item.Name, intAt(item.System, "hp"), &item
		case "background":
			backgroundName, backgroundItem = item.Name, &item
		}
	}
	system := actor.System
	details := objectAt(system, "details")
	xp := objectAt(details, "xp")
	attributes := objectAt(system, "attributes")
	abilities := objectAt(system, "abilities")
	buildAttributes := objectAt(objectAt(system, "build"), "attributes")
	logCharacterDiagnostic(
		"decode actor_uuid=%q name=%q type=%q items=%d raw_fields={details:%t,details.level:%t,details.xp.value:%t,details.xp.max:%t,attributes:%t,attributes.hp:%t,abilities:%t,build.attributes.boosts:%t,ancestry.boosts:%t,ancestry.voluntary:%t,background.boosts:%t,class.boosts:%t,class.keyAbility:%t} decoded={level:%s,xp:%s/%s,hp:%s/%s,class:%q,class_hp:%s,ancestry:%q,ancestry_hp:%s,background:%q,ability_keys:%v,ability_raw={str:%s,dex:%s,con:%s,int:%s,wis:%s,cha:%s}}",
		"Actor."+actor.ID, actor.Name, actor.Type, len(items),
		system["details"] != nil, details["level"] != nil, xp["value"] != nil, xp["max"] != nil,
		system["attributes"] != nil, attributes["hp"] != nil, abilities != nil, buildAttributes["boosts"] != nil,
		ancestryItem != nil && ancestryItem.System["boosts"] != nil, ancestryItem != nil && ancestryItem.System["voluntary"] != nil,
		backgroundItem != nil && backgroundItem.System["boosts"] != nil, classItem != nil && classItem.System["boosts"] != nil,
		classItem != nil && classItem.System["keyAbility"] != nil,
		diagnosticIntPointer(character.Progression.Level), diagnosticIntPointer(character.Progression.Experience.Value), diagnosticIntPointer(character.Progression.Experience.Max),
		diagnosticIntPointer(hpCurrentValue(character.Attributes.HP)), diagnosticIntPointer(hpMaxValue(character.Attributes.HP)),
		className, diagnosticIntPointer(classHP), ancestryName, diagnosticIntPointer(ancestryHP), backgroundName, rawObjectKeys(abilities),
		rawJSONValue(abilities["str"]), rawJSONValue(abilities["dex"]), rawJSONValue(abilities["con"]),
		rawJSONValue(abilities["int"]), rawJSONValue(abilities["wis"]), rawJSONValue(abilities["cha"]),
	)
	_ = rawItems
}

func diagnosticIntPointer(value *int) string {
	if value == nil {
		return "<nil>"
	}
	return strconv.Itoa(*value)
}

func rawJSONValue(value json.RawMessage) string {
	if len(value) == 0 {
		return "<absent>"
	}
	return strings.TrimSpace(string(value))
}

func rawObjectKeys(raw map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func hpCurrentValue(hp *HitPointsDto) *int {
	if hp == nil {
		return nil
	}
	return hp.Current
}

func hpMaxValue(hp *HitPointsDto) *int {
	if hp == nil {
		return nil
	}
	return hp.Max
}

func deriveMaximumHP(system map[string]json.RawMessage, items []FoundryItemDto, level *int) *int {
	levelValue := 0
	if level != nil && *level > 0 {
		levelValue = *level
	}
	var classHP, ancestryHP *int
	for _, item := range items {
		switch item.Type {
		case "class":
			classHP = intAt(item.System, "hp")
		case "ancestry":
			ancestryHP = intAt(item.System, "hp")
		}
	}

	// Keep every known term. An absent class, ancestry, level or CON value
	// removes only that term from the formula instead of cancelling the result.
	maximum := 0
	if ancestryHP != nil {
		maximum += *ancestryHP
	}
	if classHP != nil && levelValue > 0 {
		maximum += *classHP * levelValue
	}
	if conModifier, ok := abilityModifierFromActor(system, items, "con", levelValue); ok && levelValue > 0 {
		maximum += conModifier * levelValue
	}
	hpModifier, _ := calculateHPModifiers(items, levelValue)
	maximum += hpModifier

	logCharacterDiagnostic(
		"hp partial formula: ancestry_hp=%s class_hp=%s level=%d con_modifier_source_available=%t item_hp_modifiers=%d => maximum_hp=%d",
		diagnosticIntPointer(ancestryHP), diagnosticIntPointer(classHP), levelValue,
		abilityModifierAvailable(system, items, levelValue), hpModifier, maximum,
	)
	if maximum < 1 {
		return nil
	}
	return &maximum
}

func abilityModifierAvailable(system map[string]json.RawMessage, items []FoundryItemDto, level int) bool {
	_, ok := abilityModifierFromActor(system, items, "con", level)
	return ok
}

func calculateHPModifiers(items []FoundryItemDto, level int) (int, bool) {
	total := 0
	for _, item := range items {
		var rules []map[string]json.RawMessage
		if json.Unmarshal(item.System["rules"], &rules) != nil {
			continue
		}
		for _, rule := range rules {
			switch stringValue(rule, "key") {
			case "ActiveEffectLike":
				// Unsupported effects are ignored individually. They do not
				// invalidate supported FlatModifiers from other items.
				continue
			case "FlatModifier":
				if !ruleSelectsHP(rule) {
					continue
				}
				modifierType := stringValue(rule, "type")
				if modifierType != "" && modifierType != "untyped" {
					continue
				}
				predicate := strings.TrimSpace(string(rule["predicate"]))
				if predicate != "" && predicate != "null" && predicate != "[]" && predicate != "{}" {
					continue
				}
				value, ok := simpleHPModifierValue(rule["value"], level)
				if ok {
					total += value
				}
			}
		}
	}
	return total, true
}

func ruleSelectsHP(rule map[string]json.RawMessage) bool {
	if selector := stringAt(rule, "selector"); selector != nil {
		switch strings.ToLower(strings.TrimSpace(*selector)) {
		case "hp", "hp-per-level", "hit-points":
			return true
		}
	}
	var selectors []string
	if json.Unmarshal(rule["selector"], &selectors) == nil {
		for _, selector := range selectors {
			switch strings.ToLower(strings.TrimSpace(selector)) {
			case "hp", "hp-per-level", "hit-points":
				return true
			}
		}
	}
	return false
}

func simpleHPModifierValue(raw json.RawMessage, level int) (int, bool) {
	var value int
	if json.Unmarshal(raw, &value) == nil {
		return value, true
	}
	var expression string
	if json.Unmarshal(raw, &expression) != nil {
		return 0, false
	}
	expression = strings.ReplaceAll(strings.TrimSpace(expression), " ", "")
	if expression == "@actor.level" {
		return level, true
	}
	if strings.HasPrefix(expression, "@actor.level*") {
		multiplier, err := strconv.Atoi(strings.TrimPrefix(expression, "@actor.level*"))
		if err != nil {
			return 0, false
		}
		return level * multiplier, true
	}
	if strings.HasSuffix(expression, "*@actor.level") {
		multiplier, err := strconv.Atoi(strings.TrimSuffix(expression, "*@actor.level"))
		if err != nil {
			return 0, false
		}
		return level * multiplier, true
	}
	value, err := strconv.Atoi(expression)
	return value, err == nil
}

func abilityModifierFromActor(system map[string]json.RawMessage, items []FoundryItemDto, ability string, level int) (int, bool) {
	entry := objectAt(objectAt(system, "abilities"), ability)
	if entry != nil {
		if modifier := firstInt(intAt(entry, "mod"), intAt(entry, "modifier")); modifier != nil {
			return *modifier, true
		}
		if score := firstInt(intAt(entry, "value"), intAt(entry, "score")); score != nil {
			return abilityModifierFromScore(*score), true
		}
	}
	scores, ok := reconstructAbilityScores(system, items, &level)
	if !ok {
		return 0, false
	}
	score, ok := scores[ability]
	if !ok {
		return 0, false
	}
	return abilityModifierFromScore(score), true
}

func reconstructedAbilityScore(system map[string]json.RawMessage, items []FoundryItemDto, ability string, level int) (int, bool) {
	if !isAbilityKey(ability) || level < 1 {
		return 0, false
	}

	// Start from the PF2e baseline and apply each known source independently.
	// Missing class/background/ancestry data or unresolved choices must not
	// erase boosts that were already established by other sources.
	score := 10
	for _, item := range items {
		switch item.Type {
		case "ancestry":
			if alternateBoosts, alternate := alternateAncestryBoosts(item.System); alternate {
				// A non-null alternateAncestryBoosts switches Foundry to the
				// two-free-boost ancestry variant. Standard boosts and ancestry
				// flaws are ignored; only selections actually present in the
				// alternate array are applied.
				for _, selected := range alternateBoosts {
					if selected == ability {
						score = applyAbilityBoost(score)
					}
				}
			} else {
				for range selectedBoostCount(objectAt(item.System, "boosts"), ability) {
					score = applyAbilityBoost(score)
				}
				flaws := selectedBoostCount(objectAt(item.System, "flaws"), ability)
				score -= flaws * 2
			}
			voluntary := objectAt(item.System, "voluntary")
			if selected := stringAt(voluntary, "boost"); selected != nil && *selected == ability {
				score = applyAbilityBoost(score)
			}
			score -= stringArrayCount(voluntary, "flaws", ability) * 2
		case "background":
			for range selectedBoostCount(objectAt(item.System, "boosts"), ability) {
				score = applyAbilityBoost(score)
			}
		case "class":
			classBoosts := objectAt(item.System, "boosts")
			if len(classBoosts) > 0 {
				for range selectedBoostCount(classBoosts, ability) {
					score = applyAbilityBoost(score)
				}
			} else {
				keyAbility := objectAt(item.System, "keyAbility")
				selected := stringAt(keyAbility, "selected")
				if selected != nil && *selected == ability {
					score = applyAbilityBoost(score)
				} else if selected == nil && len(stringsAt(keyAbility, "value")) == 1 && stringArrayCount(keyAbility, "value", ability) == 1 {
					score = applyAbilityBoost(score)
				}
			}
		}
	}

	// Foundry stores gradual ability boosts under their individual level
	// keys (for example 1, 2, 3, ...). Apply every persisted selection,
	// including choices recorded for levels above the character's current
	// level. This keeps the reconstructed score aligned with the configured
	// gradual-boost variant instead of assuming the standard 1/5/9 cadence.
	boosts := objectAt(objectAt(objectAt(system, "build"), "attributes"), "boosts")
	boostLevels := make([]int, 0, len(boosts))
	for levelKey := range boosts {
		boostLevel, err := strconv.Atoi(levelKey)
		if err != nil || boostLevel < 1 {
			continue
		}
		boostLevels = append(boostLevels, boostLevel)
	}
	sort.Ints(boostLevels)
	for _, boostLevel := range boostLevels {
		var selected []string
		if json.Unmarshal(boosts[strconv.Itoa(boostLevel)], &selected) != nil {
			continue
		}
		for _, selectedAbility := range selected {
			if selectedAbility == ability {
				score = applyAbilityBoost(score)
			}
		}
	}
	return score, true
}

func applyAbilityBoost(score int) int {
	if score >= 18 {
		return score + 1
	}
	return score + 2
}

func alternateAncestryBoosts(system map[string]json.RawMessage) ([]string, bool) {
	raw, exists := system["alternateAncestryBoosts"]
	if !exists || len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, false
	}

	var selected []string
	if json.Unmarshal(raw, &selected) != nil {
		// Presence of a non-null value still enables the alternate mode.
		// A malformed selection contributes no boosts, but must not cause
		// standard ancestry boosts/flaws to be applied as a fallback.
		return nil, true
	}

	valid := selected[:0]
	for _, ability := range selected {
		if isAbilityKey(ability) {
			valid = append(valid, ability)
		}
	}
	return valid, true
}

func selectedBoostCount(raw map[string]json.RawMessage, ability string) int {
	count := 0
	for _, value := range raw {
		var entry map[string]json.RawMessage
		if json.Unmarshal(value, &entry) != nil {
			continue
		}
		if selectedBoost(entry) == ability {
			count++
		}
	}
	return count
}

func stringArrayCount(raw map[string]json.RawMessage, key, value string) int {
	var values []string
	if raw == nil || json.Unmarshal(raw[key], &values) != nil {
		return 0
	}
	count := 0
	for _, item := range values {
		if item == value {
			count++
		}
	}
	return count
}

func decodeKeyAbility(details map[string]json.RawMessage, items []FoundryItemDto) *string {
	for _, item := range items {
		if item.Type != "class" {
			continue
		}
		keyAbility := objectAt(item.System, "keyAbility")
		if selected := stringAt(keyAbility, "selected"); selected != nil && *selected != "" {
			return selected
		}
		values := stringsAt(keyAbility, "value")
		if len(values) == 1 && values[0] != "" {
			value := values[0]
			return &value
		}
		return nil
	}
	return stringAt(objectAt(details, "keyability"), "value")
}

func enrichSpellcastingItems(items []CharacterItemDto, foundryItems []FoundryItemDto) {
	type spellcastingEntryInfo struct {
		item           FoundryItemDto
		tradition      *string
		preparedType   *string
		preparedRanks  map[string][]int
		slotsAvailable bool
	}
	entries := make(map[string]spellcastingEntryInfo)
	sources := make(map[string]FoundryItemDto)
	for _, item := range foundryItems {
		sources[item.ID] = item
		if item.Type != "spellcastingEntry" {
			continue
		}
		preparedType := stringAt(objectAt(item.System, "prepared"), "value")
		if category := firstString(
			stringAt(objectAt(item.System, "category"), "value"),
			stringAt(item.System, "category"),
		); category != nil && strings.EqualFold(*category, "focus") {
			preparedType = category
		}
		info := spellcastingEntryInfo{
			item:          item,
			tradition:     firstString(stringAt(objectAt(item.System, "tradition"), "value"), stringAt(item.System, "tradition")),
			preparedType:  preparedType,
			preparedRanks: make(map[string][]int),
		}
		if preparedType != nil && *preparedType == "prepared" {
			slots := objectAt(item.System, "slots")
			if slots != nil {
				info.slotsAvailable = true
				for rank := 0; rank <= 11; rank++ {
					group := objectAt(slots, "slot"+strconv.Itoa(rank))
					if group == nil {
						continue
					}
					var prepared []json.RawMessage
					if json.Unmarshal(group["prepared"], &prepared) != nil {
						continue
					}
					for _, slot := range prepared {
						var slotData map[string]json.RawMessage
						if json.Unmarshal(slot, &slotData) != nil {
							continue
						}
						if spellID := stringAt(slotData, "id"); spellID != nil && *spellID != "" {
							info.preparedRanks[*spellID] = append(info.preparedRanks[*spellID], rank)
						}
					}
				}
			}
		}
		entries[item.ID] = info
	}
	for i := range items {
		if items[i].Type != "spell" {
			continue
		}
		source := sources[items[i].ID]
		entryID := items[i].SpellcastingEntryID
		if entryID == nil {
			entryID = firstString(stringAt(source.System, "location"), stringAt(objectAt(source.System, "location"), "value"))
		}
		if entryID != nil {
			items[i].SpellcastingEntryID = entryID
			if entry, ok := entries[*entryID]; ok {
				items[i].Tradition = entry.tradition
				items[i].SpellcastingType = entry.preparedType
				if entry.item.Name != "" {
					entryName := entry.item.Name
					items[i].SpellList = &entryName
				}
				if entry.preparedType != nil && *entry.preparedType == "prepared" && entry.slotsAvailable {
					ranks := append([]int(nil), entry.preparedRanks[items[i].ID]...)
					sort.Ints(ranks)
					items[i].PreparedRanks = ranks
					prepared := len(ranks) > 0
					items[i].Prepared = &prepared
				}
			}
		}
		if items[i].Tradition == nil {
			items[i].Tradition = firstString(
				stringAt(objectAt(source.System, "tradition"), "value"),
				stringAt(source.System, "tradition"),
			)
		}
		if items[i].SpellList == nil {
			items[i].SpellList = firstString(
				stringAt(objectAt(source.System, "spellList"), "value"),
				stringAt(source.System, "spellList"),
			)
		}
		if items[i].Prepared == nil {
			items[i].Prepared = firstBool(
				boolAt(source.System, "prepared"),
				boolAt(objectAt(source.System, "prepared"), "value"),
			)
		}
	}
}

func firstBool(values ...*bool) *bool {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func decodeResource(raw map[string]json.RawMessage) *ResourceValueDto {
	if raw == nil {
		return nil
	}
	return &ResourceValueDto{Value: intAt(raw, "value"), Max: intAt(raw, "max")}
}

func decodeSkills(raw json.RawMessage) CharacterSkillsDto {
	standard := make(map[string]CharacterSkillDto)
	lore := make(map[string]CharacterSkillDto)
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return CharacterSkillsDto{Standard: standard, Lore: lore}
	}
	var entries map[string]json.RawMessage
	if json.Unmarshal(raw, &entries) != nil {
		return CharacterSkillsDto{Standard: standard, Lore: lore}
	}
	for key, data := range entries {
		var entry map[string]json.RawMessage
		if json.Unmarshal(data, &entry) != nil {
			continue
		}
		skill := CharacterSkillDto{Rank: intAt(entry, "rank")}
		if strings.HasPrefix(strings.ToLower(key), "lore-") {
			lore[key] = skill
		} else {
			standard[key] = skill
		}
	}
	return CharacterSkillsDto{Standard: standard, Lore: lore}
}

func decodeSkillsWithTraining(raw json.RawMessage, items []FoundryItemDto) CharacterSkillsDto {
	skills := decodeSkills(raw)
	for _, item := range items {
		trained := objectAt(item.System, "trainedSkills")
		if item.Type == "class" || item.Type == "background" || trained != nil {
			for _, slug := range stringsAt(trained, "value") {
				mergeSkillRank(&skills, slug, 1)
			}
			for _, lore := range stringsAt(trained, "lore") {
				mergeSkillRank(&skills, loreSlug(lore), 1)
			}
		}
		mergeSkillRuleRanks(&skills, item)
	}
	return skills
}

func mergeSkillRank(skills *CharacterSkillsDto, slug string, rank int) {
	slug = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(slug, "_", "-")))
	if slug == "" || rank < 0 {
		return
	}
	if strings.HasPrefix(slug, "lore-") {
		current := skills.Lore[slug]
		if current.Rank == nil || *current.Rank < rank {
			value := rank
			current.Rank = &value
			skills.Lore[slug] = current
		}
		return
	}
	current := skills.Standard[slug]
	if current.Rank == nil || *current.Rank < rank {
		value := rank
		current.Rank = &value
		skills.Standard[slug] = current
	}
}

func loreSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimSuffix(value, " lore")
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "-")
	return "lore-" + value
}

// mergeSkillRuleRanks reads only unconditional ActiveEffectLike upgrades that
// target a concrete system.skills.<slug>.rank path. It does not execute rules.
func mergeSkillRuleRanks(skills *CharacterSkillsDto, item FoundryItemDto) {
	var rules []json.RawMessage
	if json.Unmarshal(item.System["rules"], &rules) != nil {
		return
	}
	selections := make(map[string]string)
	for _, raw := range rules {
		var rule map[string]json.RawMessage
		if json.Unmarshal(raw, &rule) != nil || stringValue(rule, "key") != "ChoiceSet" {
			continue
		}
		flag, selection := stringValue(rule, "flag"), stringValue(rule, "selection")
		if flag != "" && selection != "" {
			selections[flag] = selection
		}
	}
	for _, raw := range rules {
		var rule map[string]json.RawMessage
		if json.Unmarshal(raw, &rule) != nil ||
			stringValue(rule, "key") != "ActiveEffectLike" ||
			stringValue(rule, "mode") != "upgrade" {
			continue
		}
		if len(rule["predicate"]) > 0 && !bytes.Equal(bytes.TrimSpace(rule["predicate"]), []byte("null")) &&
			!bytes.Equal(bytes.TrimSpace(rule["predicate"]), []byte("[]")) {
			continue
		}
		path := stringValue(rule, "path")
		const marker = "{item|flags.system.rulesSelections."
		if start := strings.Index(path, marker); start >= 0 {
			end := strings.Index(path[start:], "}")
			if end < 0 {
				continue
			}
			flag := path[start+len(marker) : start+end]
			selection := selections[flag]
			if selection == "" {
				selectionRaw := ruleSelection(item.Flags, flag)
				_ = json.Unmarshal(selectionRaw, &selection)
			}
			if selection == "" {
				continue
			}
			path = strings.Replace(path, path[start:start+end+1], selection, 1)
		}
		if !strings.HasPrefix(path, "system.skills.") || !strings.HasSuffix(path, ".rank") {
			continue
		}
		slug := strings.TrimSuffix(strings.TrimPrefix(path, "system.skills."), ".rank")
		if slug == "" || strings.Contains(slug, ".") {
			continue
		}
		var rank int
		if json.Unmarshal(rule["value"], &rank) == nil {
			mergeSkillRank(skills, slug, rank)
		}
	}
}

func decodeSkillAssociation(system map[string]json.RawMessage) *string {
	prerequisites := objectAt(system, "prerequisites")
	var entries []map[string]json.RawMessage
	if prerequisites == nil || json.Unmarshal(prerequisites["value"], &entries) != nil {
		return nil
	}
	known := []struct {
		slug  string
		terms []string
	}{
		{"acrobatics", []string{"acrobatics", "акробатик"}},
		{"arcana", []string{"arcana", "аркан"}},
		{"athletics", []string{"athletics", "атлетик"}},
		{"crafting", []string{"crafting", "ремесл"}},
		{"deception", []string{"deception", "обман"}},
		{"diplomacy", []string{"diplomacy", "дипломат"}},
		{"intimidation", []string{"intimidation", "запугив"}},
		{"medicine", []string{"medicine", "медицин"}},
		{"nature", []string{"nature", "природ"}},
		{"occultism", []string{"occultism", "оккульт"}},
		{"performance", []string{"performance", "выступлен"}},
		{"religion", []string{"religion", "религи"}},
		{"society", []string{"society", "обществ"}},
		{"stealth", []string{"stealth", "скрытност"}},
		{"survival", []string{"survival", "выживан"}},
		{"thievery", []string{"thievery", "воровств"}},
	}
	for _, entry := range entries {
		value := stringValue(entry, "value")
		value = strings.ToLower(value)
		for _, candidate := range known {
			for _, term := range candidate.terms {
				if strings.Contains(value, term) {
					slug := candidate.slug
					return &slug
				}
			}
		}
	}
	return nil
}

func decodeBiography(raw, languages map[string]json.RawMessage, details map[string]json.RawMessage) CharacterBiographyDto {
	if raw == nil {
		raw = map[string]json.RawMessage{}
	}
	return CharacterBiographyDto{
		Appearance: stringAt(raw, "appearance"), Backstory: stringAt(raw, "backstory"),
		BirthPlace: stringAt(raw, "birthPlace"), Attitude: stringAt(raw, "attitude"),
		Beliefs: stringAt(raw, "beliefs"), Anathema: stringsAt(raw, "anathema"),
		Edicts: stringsAt(raw, "edicts"), Likes: stringAt(raw, "likes"),
		Dislikes: stringAt(raw, "dislikes"), Catchphrases: stringAt(raw, "catchphrases"),
		CampaignNotes: stringAt(raw, "campaignNotes"), Allies: stringAt(raw, "allies"),
		Enemies: stringAt(raw, "enemies"), Organizations: stringAt(raw, "organizations"),
		Visibility:      boolMapAt(raw, "visibility"),
		Age:             stringAt(objectAt(details, "age"), "value"),
		Height:          stringAt(objectAt(details, "height"), "value"),
		Weight:          stringAt(objectAt(details, "weight"), "value"),
		Gender:          stringAt(objectAt(details, "gender"), "value"),
		Ethnicity:       stringAt(objectAt(details, "ethnicity"), "value"),
		Nationality:     stringAt(objectAt(details, "nationality"), "value"),
		Languages:       stringsAt(languages, "value"),
		LanguageDetails: stringAt(languages, "details"),
	}
}

func decodeEquipped(raw map[string]json.RawMessage) *CharacterEquippedDto {
	if raw == nil {
		return nil
	}
	return &CharacterEquippedDto{
		CarryType: stringAt(raw, "carryType"), HandsHeld: intAt(raw, "handsHeld"),
		InSlot: boolAt(raw, "inSlot"), Invested: boolAt(raw, "invested"),
	}
}

func objectAt(raw map[string]json.RawMessage, key string) map[string]json.RawMessage {
	if raw == nil {
		return nil
	}
	var value map[string]json.RawMessage
	if json.Unmarshal(raw[key], &value) != nil {
		return nil
	}
	return value
}

func intAt(raw map[string]json.RawMessage, path ...string) *int {
	if raw == nil || len(path) == 0 {
		return nil
	}
	current := raw
	for _, key := range path[:len(path)-1] {
		current = objectAt(current, key)
		if current == nil {
			return nil
		}
	}
	data := current[path[len(path)-1]]
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	var number int
	if json.Unmarshal(data, &number) == nil {
		return &number
	}
	var numeric float64
	if json.Unmarshal(data, &numeric) == nil && numeric == float64(int(numeric)) {
		number = int(numeric)
		return &number
	}
	return nil
}

func floatAt(raw map[string]json.RawMessage, key string) *float64 {
	if raw == nil || len(raw[key]) == 0 || bytes.Equal(bytes.TrimSpace(raw[key]), []byte("null")) {
		return nil
	}
	var value float64
	if json.Unmarshal(raw[key], &value) != nil {
		return nil
	}
	return &value
}

func stringAt(raw map[string]json.RawMessage, key string) *string {
	if raw == nil || len(raw[key]) == 0 || bytes.Equal(bytes.TrimSpace(raw[key]), []byte("null")) {
		return nil
	}
	var value string
	if json.Unmarshal(raw[key], &value) != nil {
		return nil
	}
	return &value
}

func boolAt(raw map[string]json.RawMessage, key string) *bool {
	if raw == nil || len(raw[key]) == 0 || bytes.Equal(bytes.TrimSpace(raw[key]), []byte("null")) {
		return nil
	}
	var value bool
	if json.Unmarshal(raw[key], &value) != nil {
		return nil
	}
	return &value
}

func stringsAt(raw map[string]json.RawMessage, key string) []string {
	var values []string
	if raw == nil || json.Unmarshal(raw[key], &values) != nil {
		return nil
	}
	return values
}

func boolMapAt(raw map[string]json.RawMessage, key string) map[string]bool {
	var result map[string]bool
	if raw == nil || json.Unmarshal(raw[key], &result) != nil {
		return nil
	}
	return result
}

func firstString(values ...*string) *string {
	for _, value := range values {
		if value != nil && *value != "" {
			return value
		}
	}
	return nil
}

func firstInt(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func cloneRaw(value []byte) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

// decodeStatistics prefers prepared Actor values. If those are absent, it
// reconstructs scores only when all ancestry, background, class and level-boost
// selections required at the character's current level are present.
func decodeStatistics(system map[string]json.RawMessage, items []FoundryItemDto, level *int) CharacterStatisticsInputDto {
	abilities := make(map[string]AbilityScoreDto)
	for _, key := range []string{"str", "dex", "con", "int", "wis", "cha"} {
		entry := objectAt(objectAt(system, "abilities"), key)
		if entry == nil {
			continue
		}
		score := firstInt(intAt(entry, "value"), intAt(entry, "score"))
		modifier := firstInt(intAt(entry, "mod"), intAt(entry, "modifier"))
		ability := AbilityScoreDto{Score: score, Modifier: modifier}
		if ability.Score != nil && ability.Modifier == nil {
			ability.Modifier = intPointerValue(abilityModifierFromScore(*ability.Score))
			ability.Derived = true
		}
		abilities[key] = ability
	}
	derived := false
	for _, ability := range abilities {
		if ability.Derived {
			derived = true
			break
		}
	}
	reconstructed := make(map[string]AbilityScoreDto)
	if scores, ok := reconstructAbilityScores(system, items, level); ok {
		for key, score := range scores {
			if current, exists := abilities[key]; exists && (current.Score != nil || current.Modifier != nil) {
				continue
			}
			reconstructed[key] = AbilityScoreDto{
				Score: intPointerValue(score), Modifier: intPointerValue(abilityModifierFromScore(score)), Derived: true,
			}
			derived = true
		}
	}
	proficiencies := objectAt(system, "proficiencies")
	perceptionRaw := objectAt(proficiencies, "perception")
	perceptionSource := "actor.proficiencies.perception"
	if perceptionRaw == nil {
		perceptionRaw = objectAt(objectAt(system, "attributes"), "perception")
		perceptionSource = "actor.attributes.perception"
	}
	perception := proficiencyAt(perceptionRaw, perceptionSource)
	savesRaw := objectAt(proficiencies, "savingThrows")
	if savesRaw == nil {
		savesRaw = objectAt(proficiencies, "saves")
	}
	savesSource := "actor.proficiencies.savingThrows"
	if savesRaw == nil {
		savesRaw = objectAt(system, "saves")
		savesSource = "actor.saves"
	}
	defensesRaw := objectAt(proficiencies, "defenses")
	defensesSource := "actor.proficiencies.defenses"
	if defensesRaw == nil {
		defensesRaw = objectAt(system, "defenses")
		defensesSource = "actor.defenses"
	}
	return CharacterStatisticsInputDto{
		Abilities: abilities, ReconstructedAbilities: reconstructed, AbilitiesDerived: derived,
		Proficiencies: CharacterProficienciesDto{
			Perception: perception,
			Saves:      proficiencyMap(savesRaw, savesSource),
			Defenses:   proficiencyMap(defensesRaw, defensesSource),
		},
	}
}

func reconstructAbilityScores(system map[string]json.RawMessage, items []FoundryItemDto, level *int) (map[string]int, bool) {
	if level == nil || *level < 1 {
		return nil, false
	}
	result := make(map[string]int, 6)
	for _, ability := range []string{"str", "dex", "con", "int", "wis", "cha"} {
		score, _ := reconstructedAbilityScore(system, items, ability, *level)
		result[ability] = score
	}
	return result, true
}

func hasSelectedBoosts(raw map[string]json.RawMessage) bool {
	for _, value := range raw {
		var entry map[string]json.RawMessage
		if json.Unmarshal(value, &entry) != nil {
			continue
		}
		if selected := stringAt(entry, "selected"); selected != nil && *selected != "" {
			return true
		}
	}
	return false
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

// completeSelectedBoosts is retained for callers that need to validate a fully
// selected boost group. Reconstruction itself deliberately does not use it:
// unresolved choices are skipped independently rather than aborting all scores.
func completeSelectedBoosts(raw map[string]json.RawMessage) bool {
	if raw == nil {
		return true
	}
	for _, value := range raw {
		var entry map[string]json.RawMessage
		if json.Unmarshal(value, &entry) != nil {
			continue
		}
		var choices []string
		if len(entry["value"]) > 0 && json.Unmarshal(entry["value"], &choices) != nil {
			continue
		}
		if len(choices) == 0 {
			continue
		}
		selected := selectedBoost(entry)
		if selected == "" {
			return false
		}
	}
	return true
}

// selectedBoost returns the explicit selection, or the only legal choice when
// Foundry has not materialized selected for a mandatory single-option boost.
// Ambiguous and empty choices intentionally contribute nothing.
func selectedBoost(entry map[string]json.RawMessage) string {
	if selected := stringAt(entry, "selected"); selected != nil && isAbilityKey(*selected) {
		var choices []string
		if len(entry["value"]) == 0 || json.Unmarshal(entry["value"], &choices) != nil || len(choices) == 0 || containsString(choices, *selected) {
			return *selected
		}
		return ""
	}
	var choices []string
	if json.Unmarshal(entry["value"], &choices) == nil && len(choices) == 1 && isAbilityKey(choices[0]) {
		return choices[0]
	}
	return ""
}

func abilityModifierFromScore(score int) int {
	delta := score - 10
	modifier := delta / 2
	if delta < 0 && delta%2 != 0 {
		modifier--
	}
	return modifier
}

func intPointerValue(value int) *int { return &value }

func isAbilityKey(value string) bool {
	switch value {
	case "str", "dex", "con", "int", "wis", "cha":
		return true
	default:
		return false
	}
}

func proficiencyAt(raw map[string]json.RawMessage, source string) *ProficiencyDto {
	if raw == nil {
		return nil
	}
	rank := intAt(raw, "rank")
	if rank == nil {
		return nil
	}
	return &ProficiencyDto{Rank: rank, Source: source}
}

func proficiencyMap(raw map[string]json.RawMessage, source string) map[string]ProficiencyDto {
	result := make(map[string]ProficiencyDto)
	for key, value := range raw {
		var entry map[string]json.RawMessage
		if json.Unmarshal(value, &entry) != nil {
			continue
		}
		if rank := intAt(entry, "rank"); rank != nil {
			result[key] = ProficiencyDto{Rank: rank, Source: source + "." + key}
		}
	}
	return result
}

func decodeClassProficiencies(items []FoundryItemDto) *ClassProficiencyProfileDto {
	for _, item := range items {
		if item.Type != "class" {
			continue
		}
		trainedSkills := objectAt(item.System, "trainedSkills")
		return &ClassProficiencyProfileDto{
			PerceptionRank:   intAt(item.System, "perception"),
			SavingThrows:     intMapAt(objectAt(item.System, "savingThrows")),
			Defenses:         intMapAt(objectAt(item.System, "defenses")),
			Attacks:          intMapAt(objectAt(item.System, "attacks")),
			SpellcastingRank: intAt(item.System, "spellcasting"),
			TrainedSkills:    stringsAt(trainedSkills, "value"),
			AdditionalSkills: intAt(trainedSkills, "additional"),
			Features:         decodeClassFeatureReferences(objectAt(item.System, "items")),
			RuleElements:     decodeRuleElements(item.System, item.Flags),
		}
	}
	return nil
}

// decodeClassFeatureReferences retains links from the class progression table.
// The linked compendium documents must be fetched separately before their rules
// can be interpreted; names and levels alone never imply a proficiency upgrade.
func decodeAncestryFeatureReferences(items []FoundryItemDto) []ClassFeatureReferenceDto {
	for _, item := range items {
		if item.Type == "ancestry" {
			return decodeClassFeatureReferences(objectAt(item.System, "items"))
		}
	}
	return nil
}

func decodeClassFeatureReferences(raw map[string]json.RawMessage) []ClassFeatureReferenceDto {
	if raw == nil {
		return nil
	}
	features := make([]ClassFeatureReferenceDto, 0, len(raw))
	for _, value := range raw {
		var entry map[string]json.RawMessage
		if json.Unmarshal(value, &entry) != nil {
			continue
		}
		uuid := stringAt(entry, "uuid")
		name := stringAt(entry, "name")
		if uuid == nil || *uuid == "" || name == nil || *name == "" {
			continue
		}
		features = append(features, ClassFeatureReferenceDto{
			UUID: *uuid, Name: *name, Level: intAt(entry, "level"), Image: stringAt(entry, "img"),
		})
	}
	sort.Slice(features, func(i, j int) bool {
		if features[i].Level == nil {
			return false
		}
		if features[j].Level == nil {
			return true
		}
		if *features[i].Level != *features[j].Level {
			return *features[i].Level < *features[j].Level
		}
		return features[i].UUID < features[j].UUID
	})
	return features
}

// decodeRuleElements copies a deliberately small, declarative projection.
// Raw retains unsupported fields; no Rule Element expressions are evaluated.
func decodeRuleElements(system, flags map[string]json.RawMessage) []RuleElementDto {
	if system == nil {
		return nil
	}
	var rawRules []json.RawMessage
	if json.Unmarshal(system["rules"], &rawRules) != nil {
		return nil
	}
	rules := make([]RuleElementDto, 0, len(rawRules))
	for _, raw := range rawRules {
		var entry map[string]json.RawMessage
		if json.Unmarshal(raw, &entry) != nil {
			continue
		}
		flag := stringValue(entry, "flag")
		selection := cloneRaw(entry["selection"])
		if len(selection) == 0 && flag != "" {
			selection = ruleSelection(flags, flag)
		}
		rule := RuleElementDto{
			Key:          stringValue(entry, "key"),
			Mode:         stringValue(entry, "mode"),
			Path:         stringValue(entry, "path"),
			Flag:         stringValue(entry, "flag"),
			Value:        cloneRaw(entry["value"]),
			Selection:    selection,
			Predicate:    cloneRaw(entry["predicate"]),
			Selector:     cloneRaw(entry["selector"]),
			ModifierType: stringValue(entry, "type"),
			Raw:          cloneRaw(raw),
		}
		if rule.Key != "" {
			rules = append(rules, rule)
		}
	}
	return rules
}

func ruleSelection(flags map[string]json.RawMessage, flag string) json.RawMessage {
	for _, namespace := range []string{"system", "pf2e"} {
		selections := objectAt(objectAt(flags, namespace), "rulesSelections")
		if len(selections[flag]) > 0 {
			return cloneRaw(selections[flag])
		}
	}
	return nil
}

func stringValue(raw map[string]json.RawMessage, key string) string {
	value := stringAt(raw, key)
	if value == nil {
		return ""
	}
	return *value
}

func intMapAt(raw map[string]json.RawMessage) map[string]int {
	result := make(map[string]int)
	for key, value := range raw {
		var number int
		if json.Unmarshal(value, &number) == nil {
			result[key] = number
		}
	}
	return result
}

func decodeArmorData(system map[string]json.RawMessage) *ArmorItemDataDto {
	if system == nil {
		return nil
	}
	ac := intAt(system, "acBonus")
	dexCap := intAt(system, "dexCap")
	strength := intAt(system, "strength")
	if ac == nil && dexCap == nil && strength == nil {
		return nil
	}
	return &ArmorItemDataDto{Category: stringAt(system, "category"), ACBonus: ac, DexCap: dexCap, Strength: strength}
}

func decodeItemRunes(system map[string]json.RawMessage) *ItemRuneDataDto {
	raw := objectAt(system, "runes")
	if raw == nil {
		return nil
	}
	return &ItemRuneDataDto{
		Potency:   intAt(raw, "potency"),
		Resilient: intAt(raw, "resilient"),
		Property:  stringsAt(raw, "property"),
	}
}
