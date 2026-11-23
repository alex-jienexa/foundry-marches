package pf2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
		Statistics: decodeStatistics(system, intAt(details, "level", "value"), actor.Items),
		Resources:  CharacterResourcesDto{HeroPoints: decodeResource(hero)},
		Skills:     decodeSkills(system["skills"]),
		Items:      normalizedItems,
		Biography:  decodeBiography(bio, objectAt(details, "languages"), details),
	}
	character.Build = buildReferences(normalizedItems)
	character.Build.ClassProficiencies = decodeClassProficiencies(actor.Items)
	character.Build.KeyAbility = decodeKeyAbility(details, actor.Items)

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
	if hp != nil && hp.Max == nil {
		hp.Max = deriveMaximumHP(system, items, level)
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

func deriveMaximumHP(system map[string]json.RawMessage, items []FoundryItemDto, level *int) *int {
	if level == nil || *level < 1 {
		return nil
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
	if classHP == nil || ancestryHP == nil {
		return nil
	}
	if hasUnsupportedHPModifier(items) {
		return nil
	}
	conModifier, ok := abilityModifierFromActor(system, items, "con", *level)
	if !ok {
		return nil
	}
	maximum := (*classHP+conModifier)**level + *ancestryHP
	if maximum < 1 {
		return nil
	}
	return &maximum
}

func hasUnsupportedHPModifier(items []FoundryItemDto) bool {
	for _, item := range items {
		var rules []map[string]json.RawMessage
		if json.Unmarshal(item.System["rules"], &rules) != nil {
			continue
		}
		for _, rule := range rules {
			key := stringAt(rule, "key")
			path := stringAt(rule, "path")
			selector := stringAt(rule, "selector")
			if key != nil && *key == "ActiveEffectLike" && path != nil &&
				(strings.Contains(*path, "system.abilities.con") || strings.Contains(*path, "system.attributes.hp")) {
				return true
			}
			if key != nil && *key == "FlatModifier" && selector != nil &&
				(strings.Contains(*selector, "hp") || strings.Contains(*selector, "hit-points")) {
				return true
			}
		}
	}
	return false
}

func abilityModifierFromActor(system map[string]json.RawMessage, items []FoundryItemDto, ability string, level int) (int, bool) {
	entry := objectAt(objectAt(system, "abilities"), ability)
	if entry != nil {
		if modifier := firstInt(intAt(entry, "mod"), intAt(entry, "modifier")); modifier != nil {
			return *modifier, true
		}
		if score := firstInt(intAt(entry, "value"), intAt(entry, "score")); score != nil {
			return (*score - 10) / 2, true
		}
	}
	score, ok := reconstructedAbilityScore(system, items, ability, level)
	if !ok {
		return 0, false
	}
	return (score - 10) / 2, true
}

func reconstructedAbilityScore(system map[string]json.RawMessage, items []FoundryItemDto, ability string, level int) (int, bool) {
	hasAncestry, hasBackground, hasClass := false, false, false
	boosts := 0
	flaws := 0
	for _, item := range items {
		switch item.Type {
		case "ancestry":
			hasAncestry = true
			boosts += selectedBoostCount(objectAt(item.System, "boosts"), ability)
			flaws += selectedBoostCount(objectAt(item.System, "flaws"), ability)
			voluntary := objectAt(item.System, "voluntary")
			flaws += stringArrayCount(voluntary, "flaws", ability)
		case "background":
			hasBackground = true
			boosts += selectedBoostCount(objectAt(item.System, "boosts"), ability)
		case "class":
			hasClass = true
			keyAbility := objectAt(item.System, "keyAbility")
			selected := stringAt(keyAbility, "selected")
			if selected != nil && *selected == ability {
				boosts++
			} else if selected == nil && stringArrayCount(keyAbility, "value", ability) == 1 && len(stringsAt(keyAbility, "value")) == 1 {
				boosts++
			}
		}
	}
	if !hasAncestry || !hasBackground || !hasClass {
		return 0, false
	}
	build := objectAt(system, "build")
	attributes := objectAt(build, "attributes")
	rawBoosts := objectAt(attributes, "boosts")
	if rawBoosts == nil {
		return 0, false
	}
	for levelKey, raw := range rawBoosts {
		boostLevel, err := strconv.Atoi(levelKey)
		if err != nil || boostLevel > level {
			continue
		}
		var selected []string
		if json.Unmarshal(raw, &selected) == nil {
			for _, value := range selected {
				if value == ability {
					boosts++
				}
			}
		}
	}
	score := 10
	for range boosts {
		if score >= 18 {
			score++
		} else {
			score += 2
		}
	}
	score -= flaws * 2
	return score, true
}

func selectedBoostCount(raw map[string]json.RawMessage, ability string) int {
	count := 0
	for _, value := range raw {
		var entry map[string]json.RawMessage
		if json.Unmarshal(value, &entry) != nil {
			continue
		}
		if selected := stringAt(entry, "selected"); selected != nil && *selected == ability {
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
	entries := make(map[string]FoundryItemDto)
	sources := make(map[string]FoundryItemDto)
	for _, item := range foundryItems {
		sources[item.ID] = item
		if item.Type == "spellcastingEntry" {
			entries[item.ID] = item
		}
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
				tradition := firstString(
					stringAt(objectAt(entry.System, "tradition"), "value"),
					stringAt(entry.System, "tradition"),
				)
				items[i].Tradition = tradition
				entryName := entry.Name
				if entryName != "" {
					items[i].SpellList = &entryName
				}
				prepared := firstBool(boolAt(entry.System, "prepared"), boolAt(objectAt(entry.System, "prepared"), "value"))
				if prepared != nil {
					items[i].Prepared = prepared
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

// decodeStatistics reads explicit Actor-level values and reconstructs ability
// scores only from complete, saved PF2e boost selections.
func decodeStatistics(system map[string]json.RawMessage, level *int, actorItems []FoundryItemDto) CharacterStatisticsInputDto {
	abilities := make(map[string]AbilityScoreDto)
	for _, key := range []string{"str", "dex", "con", "int", "wis", "cha"} {
		entry := objectAt(objectAt(system, "abilities"), key)
		if entry == nil {
			continue
		}
		score := firstInt(intAt(entry, "value"), intAt(entry, "score"))
		modifier := firstInt(intAt(entry, "mod"), intAt(entry, "modifier"))
		abilities[key] = AbilityScoreDto{Score: score, Modifier: modifier}
	}
	if derived := decodeAbilityScoresFromBoosts(system, level, actorItems); len(derived) > 0 {
		for key, value := range derived {
			if _, exists := abilities[key]; !exists {
				abilities[key] = value
			}
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
		Abilities: abilities,
		Proficiencies: CharacterProficienciesDto{
			Perception: perception,
			Saves:      proficiencyMap(savesRaw, savesSource),
			Defenses:   proficiencyMap(defensesRaw, defensesSource),
		},
	}
}

// decodeAbilityScoresFromBoosts reconstructs remaster ability scores only when
// the Actor snapshot contains the complete level-1 boost selection. It does not
// execute Rule Elements or infer choices absent from the saved build data.
func decodeAbilityScoresFromBoosts(system map[string]json.RawMessage, level *int, actorItems []FoundryItemDto) map[string]AbilityScoreDto {
	if level == nil {
		return nil
	}
	buildAttributes := objectAt(objectAt(system, "build"), "attributes")
	if buildAttributes == nil {
		return nil
	}
	var boosts map[string][]string
	if json.Unmarshal(buildAttributes["boosts"], &boosts) != nil {
		return nil
	}
	requiredRounds := []int{1}
	for _, round := range []int{5, 10, 15, 20} {
		if *level >= round {
			requiredRounds = append(requiredRounds, round)
		}
	}
	for _, round := range requiredRounds {
		selected, ok := boosts[strconv.Itoa(round)]
		if !ok || len(selected) != 4 {
			return nil
		}
		for _, key := range selected {
			if !isAbilityKey(key) {
				return nil
			}
		}
	}
	scores := map[string]int{"str": 10, "dex": 10, "con": 10, "int": 10, "wis": 10, "cha": 10}
	flaws := make(map[string][]string)
	if raw := buildAttributes["flaws"]; len(raw) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if json.Unmarshal(raw, &flaws) != nil {
			var levelOneFlaws []string
			if json.Unmarshal(raw, &levelOneFlaws) != nil {
				return nil
			}
			flaws["1"] = append(flaws["1"], levelOneFlaws...)
		}
	}
	// Ancestry flaw choices are stored on the Actor's ancestry Item rather
	// than in system.abilities; apply only an explicit selected value.
	for _, item := range actorItems {
		if item.Type != "ancestry" {
			continue
		}
		for _, rawSelection := range objectAt(item.System, "flaws") {
			var selection map[string]json.RawMessage
			if json.Unmarshal(rawSelection, &selection) != nil {
				return nil
			}
			selected := stringAt(selection, "selected")
			if selected != nil && *selected != "" {
				flaws["1"] = append(flaws["1"], *selected)
			}
		}
		if voluntary := objectAt(item.System, "voluntary"); voluntary != nil {
			flaws["1"] = append(flaws["1"], stringsAt(voluntary, "flaws")...)
		}
	}
	rounds := make([]int, 0, len(boosts))
	for rawLevel := range boosts {
		roundLevel, err := strconv.Atoi(rawLevel)
		if err != nil || (roundLevel != 1 && roundLevel != 5 && roundLevel != 10 && roundLevel != 15 && roundLevel != 20) {
			return nil
		}
		if roundLevel <= *level {
			rounds = append(rounds, roundLevel)
		}
	}
	sort.Ints(rounds)
	for _, roundLevel := range rounds {
		levelKey := strconv.Itoa(roundLevel)
		seen := make(map[string]struct{})
		for _, key := range flaws[levelKey] {
			if !isAbilityKey(key) {
				return nil
			}
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			scores[key] -= 2
		}
		clear(seen)
		for _, key := range boosts[levelKey] {
			if !isAbilityKey(key) {
				return nil
			}
			if _, duplicate := seen[key]; duplicate {
				return nil
			}
			seen[key] = struct{}{}
			if scores[key] >= 18 {
				scores[key]++
			} else {
				scores[key] += 2
			}
		}
	}
	result := make(map[string]AbilityScoreDto, len(scores))
	for key, score := range scores {
		modifier := int(math.Floor(float64(score-10) / 2))
		scoreValue, modifierValue := score, modifier
		result[key] = AbilityScoreDto{Score: &scoreValue, Modifier: &modifierValue}
	}
	return result
}

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
