package pf2e

import "encoding/json"

// RelayResponseDto is the transport envelope returned by Foundry REST API Relay.
type RelayResponseDto struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	UUID      string          `json:"uuid,omitempty"`
	Data      json.RawMessage `json:"data"`
}

// FoundryActorDto and FoundryItemDto describe only fields the adapter reads.
// The complete source document is retained separately as DecodedActor.RawSnapshot.
type FoundryActorDto struct {
	ID     string                     `json:"_id"`
	Name   string                     `json:"name"`
	Type   string                     `json:"type"`
	Image  *string                    `json:"img"`
	System map[string]json.RawMessage `json:"system"`
	Items  []FoundryItemDto           `json:"items"`
	Stats  FoundryDocumentStatsDto    `json:"_stats"`
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
}

type FoundryDocumentStatsDto struct {
	CoreVersion   string  `json:"coreVersion"`
	SystemID      string  `json:"systemId"`
	SystemVersion string  `json:"systemVersion"`
	Compendium    *string `json:"compendiumSource"`
}

// CharacterDto is a read-only application projection. It contains only values
// represented in the persisted Actor document; prepared/calculated PF2e values
// are deliberately not inferred here.
type CharacterDto struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Identity      CharacterIdentityDto    `json:"identity"`
	Source        CharacterSourceDto      `json:"source"`
	Progression   CharacterProgressionDto `json:"progression"`
	Build         CharacterBuildDto       `json:"build"`
	Attributes    CharacterAttributesDto  `json:"attributes"`
	Resources     CharacterResourcesDto   `json:"resources"`
	Skills        CharacterSkillsDto      `json:"skills"`
	Items         []CharacterItemDto      `json:"items"`
	Biography     CharacterBiographyDto   `json:"biography"`
}

type CharacterIdentityDto struct {
	ID    string  `json:"id"`
	UUID  string  `json:"uuid"`
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Image *string `json:"image"`
}

type CharacterSourceDto struct {
	Platform         string   `json:"platform"`
	CoreVersion      string   `json:"coreVersion,omitempty"`
	SystemID         string   `json:"systemId,omitempty"`
	SystemVersion    string   `json:"systemVersion,omitempty"`
	MigrationVersion *float64 `json:"migrationVersion,omitempty"`
}

type CharacterProgressionDto struct {
	Level      *int          `json:"level"`
	Experience ExperienceDto `json:"experience"`
}

type ExperienceDto struct {
	Value *int `json:"value"`
	Min   *int `json:"min"`
	Max   *int `json:"max"`
}

type CharacterBuildDto struct {
	Ancestry   *CharacterItemReferenceDto `json:"ancestry"`
	Heritage   *CharacterItemReferenceDto `json:"heritage"`
	Background *CharacterItemReferenceDto `json:"background"`
	Class      *CharacterItemReferenceDto `json:"class"`
	KeyAbility *string                   `json:"keyAbility"`
}

type CharacterItemReferenceDto struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Slug  *string `json:"slug"`
	Image *string `json:"image"`
}

// HP values are copied only when present in Actor.system.attributes.hp.
// PF2e's prepared maximum HP is not present in this Actor snapshot.
type CharacterAttributesDto struct {
	HP *HitPointsDto `json:"hp"`
}

type HitPointsDto struct {
	Current *int `json:"current"`
	Temp    *int `json:"temp"`
	Max     *int `json:"max"`
}

type CharacterResourcesDto struct {
	HeroPoints *ResourceValueDto `json:"heroPoints"`
}

type ResourceValueDto struct {
	Value *int `json:"value"`
	Max   *int `json:"max"`
}

type CharacterSkillsDto struct {
	Standard map[string]CharacterSkillDto `json:"standard"`
	Lore     map[string]CharacterSkillDto `json:"lore"`
}

type CharacterSkillDto struct {
	Rank *int `json:"rank"`
}

type CharacterBiographyDto struct {
	Appearance      *string         `json:"appearance"`
	Backstory       *string         `json:"backstory"`
	BirthPlace      *string         `json:"birthPlace"`
	Attitude        *string         `json:"attitude"`
	Beliefs         *string         `json:"beliefs"`
	Anathema        []string        `json:"anathema"`
	Edicts          []string        `json:"edicts"`
	Likes           *string         `json:"likes"`
	Dislikes        *string         `json:"dislikes"`
	Catchphrases    *string         `json:"catchphrases"`
	CampaignNotes   *string         `json:"campaignNotes"`
	Allies           *string         `json:"allies"`
	Enemies          *string         `json:"enemies"`
	Organizations    *string         `json:"organizations"`
	Age              *string         `json:"age"`
	Height           *string         `json:"height"`
	Weight           *string         `json:"weight"`
	Gender           *string         `json:"gender"`
	Ethnicity        *string         `json:"ethnicity"`
	Nationality      *string         `json:"nationality"`
	Languages        []string        `json:"languages"`
	LanguageDetails  *string         `json:"languageDetails"`
	Visibility       map[string]bool `json:"visibility,omitempty"`
}

type CharacterItemCategory string

const (
	ItemCategoryEquipment CharacterItemCategory = "equipment"
	ItemCategoryAbility   CharacterItemCategory = "ability"
	ItemCategoryIdentity  CharacterItemCategory = "identity"
	ItemCategoryOther     CharacterItemCategory = "other"
)

// CharacterItemDto contains common fields suitable for application queries.
// Type-specific source fields remain available in the Actor RawSnapshot and
// are not copied wholesale into every normalized Item.
type CharacterItemDto struct {
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	Type             string                  `json:"type"`
	Slug             *string                 `json:"slug"`
	Image            *string                 `json:"image"`
	Category         CharacterItemCategory   `json:"category"`
	Quantity         *int                    `json:"quantity"`
	Level            *int                    `json:"level"`
	Equipped         *CharacterEquippedDto   `json:"equipped"`
	CompendiumSource *string                 `json:"compendiumSource"`
}

type CharacterEquippedDto struct {
	CarryType *string `json:"carryType"`
	HandsHeld *int    `json:"handsHeld"`
	InSlot    *bool   `json:"inSlot"`
	Invested  *bool   `json:"invested"`
}

type DecodedItem struct {
	Foundry       FoundryItemDto
	Character     CharacterItemDto
	RawSnapshot   json.RawMessage
}

// DecodedActor keeps the normalized projection and one authoritative raw
// snapshot. Item documents are already included in RawSnapshot.
type DecodedActor struct {
	Character   CharacterDto
	RawSnapshot json.RawMessage
}
