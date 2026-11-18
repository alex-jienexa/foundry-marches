package pf2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
		Attributes: CharacterAttributesDto{HP: decodeHP(hp)},
		Statistics: decodeStatistics(system),
		Resources:  CharacterResourcesDto{HeroPoints: decodeResource(hero)},
		Skills:     decodeSkills(system["skills"]),
		Items:      normalizedItems,
		Biography:  decodeBiography(bio, objectAt(details, "languages"), details),
	}
	character.Build = buildReferences(normalizedItems)
	character.Build.ClassProficiencies = decodeClassProficiencies(actor.Items)
	character.Build.KeyAbility = stringAt(objectAt(details, "keyability"), "value")

	return &DecodedActor{
		Character:   character,
		RawSnapshot: cloneRaw(payload),
	}, nil
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
		Slug: stringAt(system, "slug"), Category: categoryFor(item.Type),
		Quantity:         intAt(system, "quantity"),
		Level:            firstInt(intAt(system, "level", "value"), intAt(system, "level")),
		Equipped:         decodeEquipped(objectAt(system, "equipped")),
		Armor:            decodeArmorData(system),
		Runes:            decodeItemRunes(system),
		CompendiumSource: stringAt(objectAt(root, "_stats"), "compendiumSource"),
		RuleElements:     decodeRuleElements(system, item.Flags),
	}
	return &DecodedItem{
		Foundry: item, Character: normalized, RawSnapshot: cloneRaw(payload),
	}, nil
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

// decodeStatistics reads only explicit Actor-level values. In particular, it
// does not infer abilities from boosts or current ranks from class progression.
func decodeStatistics(system map[string]json.RawMessage) CharacterStatisticsInputDto {
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
		if features[i].Level == nil { return false }
		if features[j].Level == nil { return true }
		if *features[i].Level != *features[j].Level { return *features[i].Level < *features[j].Level }
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
			Key:       stringValue(entry, "key"),
			Mode:      stringValue(entry, "mode"),
			Path:      stringValue(entry, "path"),
			Flag:      stringValue(entry, "flag"),
			Value:     cloneRaw(entry["value"]),
			Selection: selection,
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
	if value == nil { return "" }
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
