package pf2e

import "encoding/json"

const CharacterSchemaVersion = 1

// RelayResponseDTO is the transport envelope returned by the Foundry REST relay.
type RelayResponseDTO struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	UUID      string          `json:"uuid,omitempty"`
	Data      json.RawMessage `json:"data"`
}

// FoundryActorDTO and FoundryItemDTO retain the complete source documents.
// Raw is authoritative: typed fields are a convenience, never a replacement.
type FoundryActorDTO struct {
	ID      string
	Name    string
	Type    string
	System  json.RawMessage
	Items   []FoundryItemDTO
	Stats   FoundryStatsDTO
	Raw     json.RawMessage
}

type FoundryItemDTO struct {
	ID     string
	Name   string
	Type   string
	System json.RawMessage
	Stats  FoundryStatsDTO
	Raw    json.RawMessage
}

type FoundryStatsDTO struct {
	CoreVersion    string `json:"coreVersion,omitempty"`
	SystemID       string `json:"systemId,omitempty"`
	SystemVersion  string `json:"systemVersion,omitempty"`
	CompendiumSource *string `json:"compendiumSource"`
}

type CharacterDTO struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Identity       CharacterIdentityDTO  `json:"identity"`
	Source         CharacterSourceDTO    `json:"source"`
	Progression    CharacterProgressionDTO `json:"progression"`
	Build          CharacterBuildDTO     `json:"build"`
	Attributes     CharacterAttributesDTO `json:"attributes"`
	Resources      CharacterResourcesDTO `json:"resources"`
	Skills         map[string]SkillDTO   `json:"skills"`
	Biography      CharacterBiographyDTO `json:"biography"`
	Items          []CharacterItemDTO    `json:"items"`

	// RawSystem preserves every PF2e field, including fields not yet modelled.
	RawSystem json.RawMessage `json:"rawSystem"`
	// RawActor preserves the complete Foundry document for re-decoding/auditing.
	RawActor json.RawMessage `json:"rawActor"`
}

type CharacterIdentityDTO struct {
	ID    string `json:"id"`
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Image string `json:"image,omitempty"`
}

type CharacterSourceDTO struct {
	Platform      string `json:"platform"`
	CoreVersion   string `json:"coreVersion,omitempty"`
	SystemID      string `json:"systemId,omitempty"`
	SystemVersion string `json:"systemVersion,omitempty"`
}

type CharacterProgressionDTO struct {
	Level       *int             `json:"level,omitempty"`
	Experience  *ExperienceDTO   `json:"experience,omitempty"`
}

type ExperienceDTO struct {
	Value *int `json:"value,omitempty"`
	Min   *int `json:"min,omitempty"`
	Max   *int `json:"max,omitempty"`
}

type CharacterBuildDTO struct {
	Ancestry  *CharacterItemReferenceDTO `json:"ancestry,omitempty"`
	Heritage  *CharacterItemReferenceDTO `json:"heritage,omitempty"`
	Background *CharacterItemReferenceDTO `json:"background,omitempty"`
	Class     *CharacterItemReferenceDTO `json:"class,omitempty"`
	KeyAbility string                   `json:"keyAbility,omitempty"`
	Boosts    json.RawMessage            `json:"boosts,omitempty"`
}

type CharacterItemReferenceDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Slug  string `json:"slug,omitempty"`
	UUID  string `json:"uuid,omitempty"`
	Image string `json:"image,omitempty"`
	Level *int   `json:"level,omitempty"`
}

type CharacterAttributesDTO struct {
	HP          *HitPointsDTO `json:"hp,omitempty"`
	ArmorClass  *int          `json:"armorClass,omitempty"`
	Perception  *int          `json:"perception,omitempty"`
	Abilities   json.RawMessage `json:"abilities,omitempty"`
	Saves       json.RawMessage `json:"saves,omitempty"`
	Initiative  string        `json:"initiative,omitempty"`
}

type HitPointsDTO struct {
	Value *int `json:"value,omitempty"`
	Max   *int `json:"max,omitempty"`
	Temp  *int `json:"temp,omitempty"`
}

type CharacterResourcesDTO struct {
	HeroPoints *CurrentMaximumDTO `json:"heroPoints,omitempty"`
}

type CurrentMaximumDTO struct {
	Value *int `json:"value,omitempty"`
	Max   *int `json:"max,omitempty"`
}

type SkillDTO struct {
	Rank *int `json:"rank,omitempty"`
}

type CharacterBiographyDTO struct {
	Appearance    string   `json:"appearance,omitempty"`
	Backstory     string   `json:"backstory,omitempty"`
	BirthPlace    string   `json:"birthPlace,omitempty"`
	Attitude      string   `json:"attitude,omitempty"`
	Beliefs       string   `json:"beliefs,omitempty"`
	Anathema      []string `json:"anathema,omitempty"`
	Edicts        []string `json:"edicts,omitempty"`
	Likes         string   `json:"likes,omitempty"`
	Dislikes      string   `json:"dislikes,omitempty"`
	Catchphrases  string   `json:"catchphrases,omitempty"`
	CampaignNotes string   `json:"campaignNotes,omitempty"`
	Allies        string   `json:"allies,omitempty"`
	Enemies       string   `json:"enemies,omitempty"`
	Organizations string   `json:"organizations,omitempty"`
	Age           string   `json:"age,omitempty"`
	Height        string   `json:"height,omitempty"`
	Weight        string   `json:"weight,omitempty"`
	Gender        string   `json:"gender,omitempty"`
	Ethnicity     string   `json:"ethnicity,omitempty"`
	Nationality   string   `json:"nationality,omitempty"`
}

type CharacterItemCategory string

const (
	ItemCategoryEquipment CharacterItemCategory = "equipment"
	ItemCategoryAbility   CharacterItemCategory = "ability"
	ItemCategoryIdentity  CharacterItemCategory = "identity"
	ItemCategoryOther     CharacterItemCategory = "other"
)

type CharacterItemDTO struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Type      string                `json:"type"`
	Slug      string                `json:"slug,omitempty"`
	Image     string                `json:"image,omitempty"`
	Category  CharacterItemCategory `json:"category"`
	Quantity  *int                  `json:"quantity,omitempty"`
	Level     *int                  `json:"level,omitempty"`
	Equipped  *EquippedStateDTO     `json:"equipped,omitempty"`
	Details   json.RawMessage       `json:"details"`
	Source    FoundryStatsDTO       `json:"source"`
	Raw       json.RawMessage       `json:"raw"`
}

type EquippedStateDTO struct {
	CarryType string `json:"carryType,omitempty"`
	Invested  *bool  `json:"invested,omitempty"`
	HandsHeld *int   `json:"handsHeld,omitempty"`
	InSlot    *bool  `json:"inSlot,omitempty"`
}
