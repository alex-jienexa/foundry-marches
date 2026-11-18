package pf2e

import "encoding/json"

// RelayResponseDto is the transport envelope returned by Foundry REST API Relay.
type RelayResponseDto struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	UUID      string          `json:"uuid,omitempty"`
	Data      json.RawMessage `json:"data"`
}

// FoundryActorDto is the raw Foundry document. Raw is authoritative and keeps
// fields unknown to this version of the decoder.
type FoundryActorDto struct {
	ID     string                     `json:"_id"`
	Name   string                     `json:"name"`
	Type   string                     `json:"type"`
	Image  *string                    `json:"img"`
	System map[string]json.RawMessage `json:"system"`
	Items  []FoundryItemDto           `json:"items"`
	Stats  FoundryDocumentStatsDto    `json:"_stats"`
	Raw    json.RawMessage            `json:"-"`
}

type FoundryItemDto struct {
	ID     string                     `json:"_id"`
	Name   string                     `json:"name"`
	Type   string                     `json:"type"`
	Image  *string                    `json:"img"`
	Folder *string                    `json:"folder"`
	Sort   *int                       `json:"sort"`
	System map[string]json.RawMessage `json:"system"`
	Flags  map[string]json.RawMessage `json:"flags"`
	Stats  FoundryDocumentStatsDto    `json:"_stats"`
	Raw    json.RawMessage            `json:"-"`
}

type FoundryDocumentStatsDto struct {
	CoreVersion   string  `json:"coreVersion"`
	SystemID      string  `json:"systemId"`
	SystemVersion string  `json:"systemVersion"`
	Compendium    *string `json:"compendiumSource"`
}

// CharacterDto is the application-facing, read-only projection of a Foundry
// Actor. It deliberately contains no PF2e rule calculations.
type CharacterDto struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Identity      CharacterIdentityDto      `json:"identity"`
	Source        CharacterSourceDto        `json:"source"`
	Progression   CharacterProgressionDto   `json:"progression"`
	Build         CharacterBuildDto         `json:"build"`
	Attributes    CharacterAttributesDto    `json:"attributes"`
	Resources     CharacterResourcesDto     `json:"resources"`
	Skills        map[string]CharacterSkillDto `json:"skills"`
	Items         []CharacterItemDto        `json:"items"`
	Biography     CharacterBiographyDto     `json:"biography"`
}

type CharacterIdentityDto struct {
	ID   string  `json:"id"`
	UUID string  `json:"uuid"`
	Name string  `json:"name"`
	Type string  `json:"type"`
	Image *string `json:"image"`
}

type CharacterSourceDto struct {
	Platform      string `json:"platform"`
	CoreVersion   string `json:"coreVersion,omitempty"`
	SystemID      string `json:"systemId,omitempty"`
	SystemVersion string `json:"systemVersion,omitempty"`
}

type CharacterProgressionDto struct {
	Level       *int                 `json:"level"`
	Experience  ExperienceDto        `json:"experience"`
}

type ExperienceDto struct {
	Value *int `json:"value"`
	Min   *int `json:"min"`
	Max   *int `json:"max"`
}

type CharacterBuildDto struct {
	Ancestry  *CharacterItemReferenceDto `json:"ancestry"`
	Heritage  *CharacterItemReferenceDto `json:"heritage"`
	Background *CharacterItemReferenceDto `json:"background"`
	Class     *CharacterItemReferenceDto `json:"class"`
}

type CharacterItemReferenceDto struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Slug  *string `json:"slug"`
	Image *string `json:"image"`
}

type CharacterAttributesDto struct {
	HP         *HitPointsDto   `json:"hp"`
	Abilities  *AbilityScoresDto `json:"abilities"`
	AC         *int            `json:"ac"`
	Perception *int            `json:"perception"`
	Saves      map[string]int  `json:"saves,omitempty"`
}

type HitPointsDto struct {
	Current *int `json:"current"`
	Temp    *int `json:"temp"`
	Max     *int `json:"max"`
}

// AbilityScoresDto is populated only when Foundry actually supplies scores.
// A null system.abilities remains nil; the decoder does not derive scores.
type AbilityScoresDto struct {
	Strength     *int `json:"strength"`
	Dexterity    *int `json:"dexterity"`
	Constitution *int `json:"constitution"`
	Intelligence *int `json:"intelligence"`
	Wisdom       *int `json:"wisdom"`
	Charisma     *int `json:"charisma"`
}

type CharacterResourcesDto struct {
	HeroPoints *ResourceValueDto `json:"heroPoints"`
}

type ResourceValueDto struct {
	Value *int `json:"value"`
	Max   *int `json:"max"`
}

type CharacterSkillDto struct {
	Rank *int `json:"rank"`
}

type CharacterBiographyDto struct {
	Appearance     *string  `json:"appearance"`
	Backstory      *string  `json:"backstory"`
	BirthPlace     *string  `json:"birthPlace"`
	Attitude       *string  `json:"attitude"`
	Beliefs        *string  `json:"beliefs"`
	Anathema       []string `json:"anathema"`
	Edicts         []string `json:"edicts"`
	Likes          *string  `json:"likes"`
	Dislikes       *string  `json:"dislikes"`
	Catchphrases   *string  `json:"catchphrases"`
	CampaignNotes  *string  `json:"campaignNotes"`
	Allies         *string  `json:"allies"`
	Enemies        *string  `json:"enemies"`
	Organizations  *string  `json:"organizations"`
	Visibility     map[string]bool `json:"visibility,omitempty"`
}

type CharacterItemCategory string

const (
	ItemCategoryEquipment CharacterItemCategory = "equipment"
	ItemCategoryAbility   CharacterItemCategory = "ability"
	ItemCategoryIdentity  CharacterItemCategory = "identity"
	ItemCategoryOther     CharacterItemCategory = "other"
)

type CharacterItemDto struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Type        string                  `json:"type"`
	Slug        *string                 `json:"slug"`
	Image       *string                 `json:"image"`
	Category    CharacterItemCategory   `json:"category"`
	Quantity    *int                    `json:"quantity"`
	Level       *int                    `json:"level"`
	Equipment   *CharacterEquipmentDto  `json:"equipment"`
	CompendiumSource *string            `json:"compendiumSource"`
	// Details contains the complete Item.system object, not a curated
	// whitelist. Type-specific decoders can be layered on without data loss.
	Details     map[string]json.RawMessage `json:"details"`
}

type CharacterEquipmentDto struct {
	CarryType *string `json:"carryType"`
	HandsHeld *int    `json:"handsHeld"`
	InSlot    *bool   `json:"inSlot"`
	Invested  *bool   `json:"invested"`
}

// DecodedActor keeps normalized data and immutable source bytes together.
type DecodedActor struct {
	Character CharacterDto
	RawActor  json.RawMessage
	RawItems  []json.RawMessage
}
