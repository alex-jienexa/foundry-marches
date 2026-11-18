package pf2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrInvalidPayload = errors.New("invalid Foundry payload")
	ErrMissingActorID = errors.New("Foundry Actor is missing _id")
	ErrMissingActorType = errors.New("Foundry Actor is missing type")
	ErrMissingItems = errors.New("Foundry Actor is missing a valid items array")
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
// The original JSON and every Item document are retained byte-for-byte.
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
	actor.Raw = cloneRaw(payload)
	actor.Items = make([]FoundryItemDto, 0, len(rawItems))
	normalizedItems := make([]CharacterItemDto, 0, len(rawItems))
	for i, raw := range rawItems {
		item, err := DecodeItem(raw)
		if err != nil {
			return nil, fmt.Errorf("Actor.items[%d]: %w", i, err)
		}
		actor.Items = append(actor.Items, item.raw)
		normalizedItems = append(normalizedItems, item.normalized)
	}

	system := actor.System
	details := objectAt(system, "details")
	xp := objectAt(details, "xp")
	hp := objectAt(objectAt(system, "attributes"), "hp")
	hero := objectAt(objectAt(system, "resources"), "heroPoints")
	bio := objectAt(details, "biography")

	character := CharacterDto{
		SchemaVersion: 1,
		Identity: CharacterIdentityDto{
			ID: actor.ID, UUID: "Actor." + actor.ID, Name: actor.Name,
			Type: actor.Type, Image: actor.Image,
		},
		Source: CharacterSourceDto{
			Platform: "foundry-vtt", CoreVersion: actor.Stats.CoreVersion,
			SystemID: actor.Stats.SystemID, SystemVersion: actor.Stats.SystemVersion,
		},
		Progression: CharacterProgressionDto{
			Level: intAt(details, "level", "value"),
			Experience: ExperienceDto{
				Value: intAt(xp, "value"), Min: intAt(xp, "min"), Max: intAt(xp, "max"),
			},
		},
		Attributes: CharacterAttributesDto{
			HP: decodeHP(hp),
			Abilities: decodeAbilities(system["abilities"]),
			AC: intAt(objectAt(system, "attributes"), "ac", "value"),
			Perception: intAt(objectAt(system, "perception"), "value"),
		},
		Resources: CharacterResourcesDto{HeroPoints: decodeResource(hero)},
		Skills: decodeSkills(system["skills"]),
		Items: normalizedItems,
		Biography: decodeBiography(bio),
	}
	character.Build = buildReferences(normalizedItems)

	return &DecodedActor{
		Character: character, RawActor: cloneRaw(payload), RawItems: cloneRawSlice(rawItems),
	}, nil
}

type decodedItem struct {
	raw        FoundryItemDto
	normalized CharacterItemDto
}

// DecodeItem preserves the full raw Item.system object. Unknown Item types
// are valid and receive category "other".
func DecodeItem(payload []byte) (decodedItem, error) {
	var item FoundryItemDto
	if err := json.Unmarshal(payload, &item); err != nil {
		return decodedItem{}, fmt.Errorf("%w: decode Item: %v", ErrInvalidPayload, err)
	}
	if item.ID == "" {
		return decodedItem{}, fmt.Errorf("%w: Item is missing _id", ErrInvalidPayload)
	}
	if item.Type == "" {
		return decodedItem{}, fmt.Errorf("%w: Item %s is missing type", ErrInvalidPayload, item.ID)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return decodedItem{}, fmt.Errorf("%w: decode Item object: %v", ErrInvalidPayload, err)
	}
	item.Raw = cloneRaw(payload)
	system := objectAt(root, "system")
	equipment := decodeEquipment(objectAt(system, "equipped"))
	return decodedItem{
		raw: item,
		normalized: CharacterItemDto{
			ID: item.ID, Name: item.Name, Type: item.Type, Image: item.Image,
			Slug: stringAt(system, "slug"), Category: categoryFor(item.Type),
			Quantity: intAt(system, "quantity"),
			Level: firstInt(intAt(system, "level", "value"), intAt(system, "level")),
			Equipment: equipment,
			CompendiumSource: stringAt(objectAt(root, "_stats"), "compendiumSource"),
			Details: cloneRawMap(system),
		},
	}, nil
}

func categoryFor(kind string) CharacterItemCategory {
	switch kind {
	case "weapon", "armor", "consumable", "treasure", "equipment", "backpack", "shield":
		return ItemCategoryEquipment
	case "feat", "action", "effect":
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
		case "ancestry": result.Ancestry = ref
		case "heritage": result.Heritage = ref
		case "background": result.Background = ref
		case "class": result.Class = ref
		}
	}
	return result
}

func decodeHP(raw map[string]json.RawMessage) *HitPointsDto {
	if raw == nil { return nil }
	return &HitPointsDto{Current: intAt(raw, "value"), Temp: intAt(raw, "temp"), Max: intAt(raw, "max")}
}

func decodeResource(raw map[string]json.RawMessage) *ResourceValueDto {
	if raw == nil { return nil }
	return &ResourceValueDto{Value: intAt(raw, "value"), Max: intAt(raw, "max")}
}

func decodeAbilities(raw json.RawMessage) *AbilityScoresDto {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) { return nil }
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil { return nil }
	return &AbilityScoresDto{
		Strength: abilityValue(obj, "str"), Dexterity: abilityValue(obj, "dex"),
		Constitution: abilityValue(obj, "con"), Intelligence: abilityValue(obj, "int"),
		Wisdom: abilityValue(obj, "wis"), Charisma: abilityValue(obj, "cha"),
	}
}

func abilityValue(raw map[string]json.RawMessage, key string) *int {
	value := objectAt(raw, key)
	if v := intAt(value, "value"); v != nil { return v }
	if len(value) == 0 { return nil }
	var direct int
	if json.Unmarshal(value[key], &direct) == nil { return &direct }
	return nil
}

func decodeSkills(raw json.RawMessage) map[string]CharacterSkillDto {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) { return map[string]CharacterSkillDto{} }
	var skills map[string]json.RawMessage
	if json.Unmarshal(raw, &skills) != nil { return map[string]CharacterSkillDto{} }
	result := make(map[string]CharacterSkillDto, len(skills))
	for name, data := range skills {
		var entry map[string]json.RawMessage
		if json.Unmarshal(data, &entry) != nil { continue }
		result[name] = CharacterSkillDto{Rank: intAt(entry, "rank")}
	}
	return result
}

func decodeBiography(raw map[string]json.RawMessage) CharacterBiographyDto {
	if raw == nil { return CharacterBiographyDto{} }
	return CharacterBiographyDto{
		Appearance: stringAt(raw, "appearance"), Backstory: stringAt(raw, "backstory"),
		BirthPlace: stringAt(raw, "birthPlace"), Attitude: stringAt(raw, "attitude"),
		Beliefs: stringAt(raw, "beliefs"), Anathema: stringsAt(raw, "anathema"),
		Edicts: stringsAt(raw, "edicts"), Likes: stringAt(raw, "likes"),
		Dislikes: stringAt(raw, "dislikes"), Catchphrases: stringAt(raw, "catchphrases"),
		CampaignNotes: stringAt(raw, "campaignNotes"), Allies: stringAt(raw, "allies"),
		Enemies: stringAt(raw, "enemies"), Organizations: stringAt(raw, "organizations"),
		Visibility: boolMapAt(raw, "visibility"),
	}
}

func decodeEquipment(raw map[string]json.RawMessage) *CharacterEquipmentDto {
	if raw == nil { return nil }
	return &CharacterEquipmentDto{
		CarryType: stringAt(raw, "carryType"), HandsHeld: intAt(raw, "handsHeld"),
		InSlot: boolAt(raw, "inSlot"), Invested: boolAt(raw, "invested"),
	}
}

func objectAt(raw map[string]json.RawMessage, key string) map[string]json.RawMessage {
	if raw == nil { return nil }
	var value map[string]json.RawMessage
	if json.Unmarshal(raw[key], &value) != nil { return nil }
	return value
}

func intAt(raw map[string]json.RawMessage, path ...string) *int {
	if raw == nil || len(path) == 0 { return nil }
	current := raw
	for _, key := range path[:len(path)-1] { current = objectAt(current, key); if current == nil { return nil } }
	data := current[path[len(path)-1]]
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) { return nil }
	var number int
	if json.Unmarshal(data, &number) == nil { return &number }
	var numeric float64
	if json.Unmarshal(data, &numeric) == nil && numeric == float64(int(numeric)) { number = int(numeric); return &number }
	return nil
}

func stringAt(raw map[string]json.RawMessage, key string) *string {
	if raw == nil || len(raw[key]) == 0 || bytes.Equal(bytes.TrimSpace(raw[key]), []byte("null")) { return nil }
	var value string
	if json.Unmarshal(raw[key], &value) != nil { return nil }
	return &value
}

func boolAt(raw map[string]json.RawMessage, key string) *bool {
	if raw == nil || len(raw[key]) == 0 || bytes.Equal(bytes.TrimSpace(raw[key]), []byte("null")) { return nil }
	var value bool
	if json.Unmarshal(raw[key], &value) != nil { return nil }
	return &value
}

func stringsAt(raw map[string]json.RawMessage, key string) []string {
	var values []string
	if raw == nil || json.Unmarshal(raw[key], &values) != nil { return nil }
	return values
}

func boolMapAt(raw map[string]json.RawMessage, key string) map[string]bool {
	result := map[string]bool{}
	if raw == nil || json.Unmarshal(raw[key], &result) != nil { return nil }
	return result
}

func firstInt(values ...*int) *int {
	for _, value := range values { if value != nil { return value } }
	return nil
}

func cloneRaw(value []byte) json.RawMessage {
	if value == nil { return nil }
	return append(json.RawMessage(nil), value...)
}

func cloneRawSlice(values []json.RawMessage) []json.RawMessage {
	result := make([]json.RawMessage, len(values))
	for i := range values { result[i] = cloneRaw(values[i]) }
	return result
}

func cloneRawMap(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil { return map[string]json.RawMessage{} }
	result := make(map[string]json.RawMessage, len(values))
	for key, value := range values { result[key] = cloneRaw(value) }
	return result
}

// ParseInteger is exposed for adapters that need to parse numeric fields from
// JSON without silently accepting fractional values.
func ParseInteger(raw json.RawMessage) (*int, error) {
	var value json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil { return nil, err }
	parsed, err := strconv.Atoi(value.String())
	if err != nil { return nil, err }
	return &parsed, nil
}
