package characters

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

type FoundrySyncInput struct {
	WorldID     string
	CharacterID string
	ActorUUID   string
	Decoded     pf2e.DecodedActor
}

type CharacterRecord struct {
	ID          string
	Name        string
	PortraitURL *string
	Ancestry    *string
	Heritage    *string
	Background  *string
	Class       *string
	Level       *int
	XP          *int
	HPCurrent   *int
	HPMax       *int
	Attributes  json.RawMessage
	Skills      json.RawMessage
}

type InventoryRecord struct {
	FoundryItemID string
	Name          string
	Quantity      *int
	State         json.RawMessage
}

type FoundrySyncRecord struct {
	WorldID       string
	CharacterID   string
	ActorUUID     string
	Character     CharacterRecord
	Inventory     []InventoryRecord
	ActorSnapshot json.RawMessage
	SnapshotHash  string
}

type characterAttributesPayload struct {
	Source      pf2e.CharacterSourceDto          `json:"source"`
	Progression pf2e.CharacterProgressionDto     `json:"progression"`
	Build       pf2e.CharacterBuildDto           `json:"build"`
	Statistics  pf2e.CharacterStatisticsInputDto `json:"statistics"`
	Resources   pf2e.CharacterResourcesDto       `json:"resources"`
	Biography   pf2e.CharacterBiographyDto       `json:"biography"`
}

func MapFoundryActor(input FoundrySyncInput) (FoundrySyncRecord, error) {
	c := input.Decoded.Character
	if input.WorldID == "" || input.CharacterID == "" || input.ActorUUID == "" {
		return FoundrySyncRecord{}, errors.New("world ID, character ID and Actor UUID are required")
	}
	if c.Identity.ID == "" || c.Identity.Name == "" || c.Identity.Type != "character" || c.Identity.UUID != input.ActorUUID {
		return FoundrySyncRecord{}, errors.New("decoded document must be a named character Actor")
	}
	if c.Progression.Level != nil && (*c.Progression.Level < 1 || *c.Progression.Level > 30) {
		return FoundrySyncRecord{}, fmt.Errorf("character level %d is outside database range 1..30", *c.Progression.Level)
	}
	if c.Progression.Experience.Value != nil && *c.Progression.Experience.Value < 0 {
		return FoundrySyncRecord{}, errors.New("character XP cannot be negative")
	}
	if len(input.Decoded.RawSnapshot) == 0 || !json.Valid(input.Decoded.RawSnapshot) {
		return FoundrySyncRecord{}, errors.New("a valid complete Actor snapshot is required")
	}
	attributes, err := json.Marshal(characterAttributesPayload{
		Source: c.Source, Progression: c.Progression, Build: c.Build,
		Statistics: c.Statistics, Resources: c.Resources, Biography: c.Biography,
	})
	if err != nil {
		return FoundrySyncRecord{}, fmt.Errorf("marshal character attributes: %w", err)
	}
	skills, err := json.Marshal(c.Skills)
	if err != nil {
		return FoundrySyncRecord{}, fmt.Errorf("marshal character skills: %w", err)
	}

	record := FoundrySyncRecord{
		WorldID: input.WorldID, CharacterID: input.CharacterID, ActorUUID: input.ActorUUID,
		Character: CharacterRecord{
			ID: input.CharacterID, Name: c.Identity.Name, PortraitURL: c.Identity.Image,
			Ancestry: referenceName(c.Build.Ancestry), Heritage: referenceName(c.Build.Heritage),
			Background: referenceName(c.Build.Background), Class: referenceName(c.Build.Class),
			Level: c.Progression.Level, XP: c.Progression.Experience.Value,
			Attributes: attributes, Skills: skills,
		},
		ActorSnapshot: append(json.RawMessage(nil), input.Decoded.RawSnapshot...),
	}
	if c.Attributes.HP != nil {
		record.Character.HPCurrent = c.Attributes.HP.Current
		record.Character.HPMax = c.Attributes.HP.Max
	}
	record.Inventory = make([]InventoryRecord, 0, len(c.Items))
	seenItemIDs := make(map[string]struct{}, len(c.Items))
	for _, item := range c.Items {
		if item.ID == "" || item.Name == "" {
			return FoundrySyncRecord{}, errors.New("Actor item is missing its Foundry ID or name")
		}
		if _, exists := seenItemIDs[item.ID]; exists {
			return FoundrySyncRecord{}, fmt.Errorf("duplicate Actor item ID %q", item.ID)
		}
		seenItemIDs[item.ID] = struct{}{}
		if item.Quantity != nil && *item.Quantity < 0 {
			return FoundrySyncRecord{}, fmt.Errorf("Actor item %q has negative quantity", item.ID)
		}
		state, marshalErr := json.Marshal(item)
		if marshalErr != nil {
			return FoundrySyncRecord{}, fmt.Errorf("marshal item %q: %w", item.ID, marshalErr)
		}
		record.Inventory = append(record.Inventory, InventoryRecord{
			FoundryItemID: item.ID, Name: item.Name, Quantity: item.Quantity, State: state,
		})
	}
	hash := sha256.Sum256(record.ActorSnapshot)
	record.SnapshotHash = hex.EncodeToString(hash[:])
	return record, nil
}

func referenceName(ref *pf2e.CharacterItemReferenceDto) *string {
	if ref == nil {
		return nil
	}
	name := ref.Name
	return &name
}
