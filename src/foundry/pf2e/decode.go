package pf2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

type object map[string]json.RawMessage

// DecodeRelayCharacter validates a Relay entity-result envelope and decodes its Actor.
func DecodeRelayCharacter(payload []byte) (*CharacterDTO, error) {
	var envelope RelayResponseDTO
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("decode relay envelope: %w", err)
	}
	if envelope.Type != "entity-result" {
		return nil, fmt.Errorf("unexpected relay response type %q: want entity-result", envelope.Type)
	}
	if !isJSONObject(envelope.Data) {
		return nil, fmt.Errorf("relay response data must be a JSON object")
	}
	return DecodeCharacter(envelope.Data)
}

// DecodeCharacter converts one complete PF2e character Actor into the application DTO.
// The original Actor and each original Item remain available as raw JSON.
func DecodeCharacter(payload []byte) (*CharacterDTO, error) {
	actor, err := decodeFoundryActor(payload)
	if err != nil {
		return nil, err
	}
	if actor.Type != "character" {
		return nil, fmt.Errorf("actor %q has type %q: want character", actor.ID, actor.Type)
	}

	var system object
	if err := json.Unmarshal(actor.System, &system); err != nil {
		return nil, fmt.Errorf("actor %q system: %w", actor.ID, err)
	}

	dto := &CharacterDTO{
		SchemaVersion: CharacterSchemaVersion,
		Identity: CharacterIdentityDTO{
			ID: actor.ID, UUID: "Actor." + actor.ID, Name: actor.Name,
			Type: actor.Type, Image: stringAt(mustObject(actor.Raw), "img"),
		},
		Source: CharacterSourceDTO{
			Platform: "foundry-vtt", CoreVersion: actor.Stats.CoreVersion,
			SystemID: actor.Stats.SystemID, SystemVersion: actor.Stats.SystemVersion,
		},
		Items: make([]CharacterItemDTO, 0, len(actor.Items)),
		RawSystem: cloneRaw(actor.System), RawActor: cloneRaw(actor.Raw),
	}

	details := objectAt(system, "details")
	dto.Progression.Level = intAtPath(details, "level", "value")
	dto.Progression.Experience = experienceAt(details)

	dto.Build.KeyAbility = stringAtPath(details, "keyability", "value")
	dto.Build.Boosts = cloneRaw(rawAtPath(objectAt(system, "build"), "attributes", "boosts"))

	attributes := objectAt(system, "attributes")
	if hp := objectAt(attributes, "hp"); hp != nil {
		dto.Attributes.HP = &HitPointsDTO{
			Value: intAt(hp, "value"), Max: intAt(hp, "max"), Temp: intAt(hp, "temp"),
		}
	}
	// These values are copied only when present on the Actor. We never derive them from class/items.
	dto.Attributes.ArmorClass = intAtPath(attributes, "ac", "value")
	dto.Attributes.Perception = intAtPath(system, "perception", "value")
	dto.Attributes.Abilities = cloneRaw(rawAt(system, "abilities"))
	dto.Attributes.Saves = cloneRaw(rawAt(system, "saves"))
	dto.Attributes.Initiative = stringAtPath(objectAt(system, "initiative"), "statistic")

	resources := objectAt(system, "resources")
	if hero := objectAt(resources, "heroPoints"); hero != nil {
		dto.Resources.HeroPoints = &CurrentMaximumDTO{Value: intAt(hero, "value"), Max: intAt(hero, "max")}
	}

	dto.Skills = decodeSkills(objectAt(system, "skills"))
	dto.Biography = decodeBiography(details)

	for i := range actor.Items {
		item, err := decodeCharacterItem(actor.Items[i])
		if err != nil {
			return nil, fmt.Errorf("actor %q item[%d]: %w", actor.ID, i, err)
		}
		dto.Items = append(dto.Items, item)
		switch item.Type {
		case "ancestry":
			if dto.Build.Ancestry == nil { dto.Build.Ancestry = itemReference(item) }
		case "heritage":
			if dto.Build.Heritage == nil { dto.Build.Heritage = itemReference(item) }
		case "background":
			if dto.Build.Background == nil { dto.Build.Background = itemReference(item) }
		case "class":
			if dto.Build.Class == nil { dto.Build.Class = itemReference(item) }
		}
	}
	return dto, nil
}

func decodeFoundryActor(payload []byte) (*FoundryActorDTO, error) {
	if !isJSONObject(payload) {
		return nil, fmt.Errorf("actor document must be a JSON object")
	}
	var root object
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("decode actor document: %w", err)
	}
	id, name, typ := stringAt(root, "_id"), stringAt(root, "name"), stringAt(root, "type")
	if id == "" || name == "" || typ == "" {
		return nil, fmt.Errorf("actor document requires non-empty _id, name and type")
	}
	system := rawAt(root, "system")
	if !isJSONObject(system) {
		return nil, fmt.Errorf("actor %q system must be a JSON object", id)
	}
	itemsRaw := rawAt(root, "items")
	if itemsRaw == nil || bytes.Equal(bytes.TrimSpace(itemsRaw), []byte("null")) {
		return nil, fmt.Errorf("actor %q items must be a JSON array (empty array is valid)", id)
	}
	var itemPayloads []json.RawMessage
	if err := json.Unmarshal(itemsRaw, &itemPayloads); err != nil || itemPayloads == nil {
		return nil, fmt.Errorf("actor %q items must be a JSON array", id)
	}
	items := make([]FoundryItemDTO, 0, len(itemPayloads))
	for i, raw := range itemPayloads {
		item, err := decodeFoundryItem(raw)
		if err != nil { return nil, fmt.Errorf("item[%d]: %w", i, err) }
		items = append(items, *item)
	}
	var stats FoundryStatsDTO
	if raw := rawAt(root, "_stats"); isJSONObject(raw) {
		if err := json.Unmarshal(raw, &stats); err != nil {
			return nil, fmt.Errorf("actor %q _stats: %w", id, err)
		}
	}
	return &FoundryActorDTO{ID:id, Name:name, Type:typ, System:cloneRaw(system), Items:items, Stats:stats, Raw:cloneRaw(payload)}, nil
}

func decodeFoundryItem(payload []byte) (*FoundryItemDTO, error) {
	if !isJSONObject(payload) { return nil, fmt.Errorf("item document must be a JSON object") }
	var root object
	if err := json.Unmarshal(payload, &root); err != nil { return nil, fmt.Errorf("decode item: %w", err) }
	id, name, typ := stringAt(root, "_id"), stringAt(root, "name"), stringAt(root, "type")
	if id == "" || name == "" || typ == "" { return nil, fmt.Errorf("item requires non-empty _id, name and type") }
	system := rawAt(root, "system")
	if !isJSONObject(system) { return nil, fmt.Errorf("item %q system must be a JSON object", id) }
	var stats FoundryStatsDTO
	if raw := rawAt(root, "_stats"); isJSONObject(raw) {
		if err := json.Unmarshal(raw, &stats); err != nil { return nil, fmt.Errorf("item %q _stats: %w", id, err) }
	}
	return &FoundryItemDTO{ID:id, Name:name, Type:typ, System:cloneRaw(system), Stats:stats, Raw:cloneRaw(payload)}, nil
}

func decodeCharacterItem(item FoundryItemDTO) (CharacterItemDTO, error) {
	var system object
	if err := json.Unmarshal(item.System, &system); err != nil { return CharacterItemDTO{}, err }
	out := CharacterItemDTO{
		ID:item.ID, Name:item.Name, Type:item.Type, Image:stringAt(mustObject(item.Raw), "img"),
		Slug:stringAt(system, "slug"), Category:categoryForType(item.Type),
		Quantity:intAt(system, "quantity"), Level:levelAt(system),
		Details:cloneRaw(item.System), Source:item.Stats, Raw:cloneRaw(item.Raw),
	}
	if equippedRaw := rawAt(system, "equipped"); isJSONObject(equippedRaw) {
		var equipped object
		_ = json.Unmarshal(equippedRaw, &equipped)
		out.Equipped = &EquippedStateDTO{
			CarryType:stringAt(equipped, "carryType"), Invested:boolAt(equipped, "invested"),
			HandsHeld:intAt(equipped, "handsHeld"), InSlot:boolAt(equipped, "inSlot"),
		}
	}
	return out, nil
}

func categoryForType(typ string) CharacterItemCategory {
	switch typ {
	case "weapon", "armor", "consumable", "treasure", "equipment", "backpack":
		return ItemCategoryEquipment
	case "feat", "action", "spell", "effect":
		return ItemCategoryAbility
	case "ancestry", "heritage", "background", "class":
		return ItemCategoryIdentity
	default:
		return ItemCategoryOther
	}
}

func itemReference(item CharacterItemDTO) *CharacterItemReferenceDTO {
	var system object
	_ = json.Unmarshal(item.Details, &system)
	ref := &CharacterItemReferenceDTO{ID:item.ID, Name:item.Name, Type:item.Type, Slug:item.Slug, Image:item.Image, Level:item.Level}
	ref.UUID = stringAtPath(system, "ancestry", "uuid")
	return ref
}

func decodeSkills(raw object) map[string]SkillDTO {
	out := make(map[string]SkillDTO, len(raw))
	for name, value := range raw {
		if !isJSONObject(value) { continue }
		var skill object
		if json.Unmarshal(value, &skill) == nil {
			out[name] = SkillDTO{Rank:intAt(skill, "rank")}
		}
	}
	return out
}

func decodeBiography(details object) CharacterBiographyDTO {
	bio := objectAt(details, "biography")
	return CharacterBiographyDTO{
		Appearance:stringAt(bio,"appearance"), Backstory:stringAt(bio,"backstory"),
		BirthPlace:stringAt(bio,"birthPlace"), Attitude:stringAt(bio,"attitude"),
		Beliefs:stringAt(bio,"beliefs"), Anathema:stringSliceAt(bio,"anathema"),
		Edicts:stringSliceAt(bio,"edicts"), Likes:stringAt(bio,"likes"),
		Dislikes:stringAt(bio,"dislikes"), Catchphrases:stringAt(bio,"catchphrases"),
		CampaignNotes:stringAt(bio,"campaignNotes"), Allies:stringAt(bio,"allies"),
		Enemies:stringAt(bio,"enemies"), Organizations:stringAt(bio,"organizations"),
		Age:stringAt(details,"age"), Height:stringAt(details,"height"),
		Weight:stringAt(details,"weight"), Gender:stringAt(details,"gender"),
		Ethnicity:stringAt(details,"ethnicity"), Nationality:stringAt(details,"nationality"),
	}
}

func experienceAt(details object) *ExperienceDTO {
	xp := objectAt(details, "xp")
	if xp == nil { return nil }
	return &ExperienceDTO{Value:intAt(xp,"value"), Min:intAt(xp,"min"), Max:intAt(xp,"max")}
}

func levelAt(system object) *int {
	raw := rawAt(system, "level")
	if isJSONObject(raw) { return intAtPath(system, "level", "value") }
	return intFromRaw(raw)
}

func intAtPath(root object, path ...string) *int { return intFromRaw(rawAtPath(root, path...)) }
func stringAtPath(root object, path ...string) string { return stringFromRaw(rawAtPath(root, path...)) }

func rawAtPath(root object, path ...string) json.RawMessage {
	current := root
	for i, key := range path {
		raw := rawAt(current, key)
		if raw == nil { return nil }
		if i == len(path)-1 { return raw }
		if !isJSONObject(raw) { return nil }
		current = objectAt(current, key)
	}
	return nil
}
func rawAt(root object, key string) json.RawMessage { return root[key] }
func objectAt(root object, key string) object {
	raw := rawAt(root,key)
	if !isJSONObject(raw) { return nil }
	var out object
	if json.Unmarshal(raw,&out) != nil { return nil }
	return out
}
func mustObject(raw []byte) object {
	var out object
	_ = json.Unmarshal(raw,&out)
	return out
}
func stringAt(root object, key string) string { return stringFromRaw(rawAt(root,key)) }
func stringFromRaw(raw json.RawMessage) string {
	var value string
	if len(raw)==0 || json.Unmarshal(raw,&value)!=nil { return "" }
	return value
}
func intAt(root object, key string) *int { return intFromRaw(rawAt(root,key)) }
func intFromRaw(raw json.RawMessage) *int {
	if len(raw)==0 { return nil }
	var n int
	if json.Unmarshal(raw,&n)==nil { return &n }
	var number json.Number
	dec := json.NewDecoder(bytes.NewReader(raw)); dec.UseNumber()
	if dec.Decode(&number)==nil {
		if parsed,err:=strconv.Atoi(number.String()); err==nil { return &parsed }
	}
	return nil
}
func boolAt(root object, key string) *bool {
	raw:=rawAt(root,key); if len(raw)==0 { return nil }
	var value bool; if json.Unmarshal(raw,&value)!=nil { return nil }; return &value
}
func stringSliceAt(root object,key string) []string {
	raw:=rawAt(root,key); if len(raw)==0 { return nil }
	var values []string; if json.Unmarshal(raw,&values)!=nil { return nil }; return values
}
func isJSONObject(raw []byte) bool {
	raw=bytes.TrimSpace(raw)
	return len(raw)>1 && raw[0]=='{' && raw[len(raw)-1]=='}'
}
func cloneRaw(raw []byte) json.RawMessage {
	if raw==nil { return nil }
	return append(json.RawMessage(nil),raw...)
}
