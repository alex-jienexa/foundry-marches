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

// CharacterDto is a read-only application projection. It contains values represented
// in the persisted Actor document plus narrowly derived ability scores and maximum HP
// whose complete inputs can be reconstructed without executing PF2e Rule Elements.
type CharacterDto struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Identity      CharacterIdentityDto        `json:"identity"`
	Source        CharacterSourceDto          `json:"source"`
	Progression   CharacterProgressionDto     `json:"progression"`
	Build         CharacterBuildDto           `json:"build"`
	Attributes    CharacterAttributesDto      `json:"attributes"`
	Statistics    CharacterStatisticsInputDto `json:"statistics"`
	Resources     CharacterResourcesDto       `json:"resources"`
	Skills        CharacterSkillsDto          `json:"skills"`
	Items         []CharacterItemDto          `json:"items"`
	Biography     CharacterBiographyDto       `json:"biography"`
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
	Ancestry           *CharacterItemReferenceDto  `json:"ancestry"`
	Heritage           *CharacterItemReferenceDto  `json:"heritage"`
	Background         *CharacterItemReferenceDto  `json:"background"`
	Class              *CharacterItemReferenceDto  `json:"class"`
	KeyAbility         *string                     `json:"keyAbility"`
	ClassProficiencies *ClassProficiencyProfileDto `json:"classProficiencies,omitempty"`
}

type CharacterItemReferenceDto struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Slug  *string `json:"slug"`
	Image *string `json:"image"`
}

// Current and temporary HP are copied from Actor.system.attributes.hp. If max is absent,
// the decoder derives it only when class, ancestry, level and Constitution are known.
type CharacterAttributesDto struct {
	HP    *HitPointsDto      `json:"hp"`
	Dying *DyingConditionDto `json:"dying,omitempty"`
}

type DyingConditionDto struct {
	Current *int `json:"current"`
	Max     *int `json:"max"`
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
	Allies          *string         `json:"allies"`
	Enemies         *string         `json:"enemies"`
	Organizations   *string         `json:"organizations"`
	Age             *string         `json:"age"`
	Height          *string         `json:"height"`
	Weight          *string         `json:"weight"`
	Gender          *string         `json:"gender"`
	Ethnicity       *string         `json:"ethnicity"`
	Nationality     *string         `json:"nationality"`
	Languages       []string        `json:"languages"`
	LanguageDetails *string         `json:"languageDetails"`
	Visibility      map[string]bool `json:"visibility,omitempty"`
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
	ID                  string                `json:"id"`
	Skill               *string               `json:"skill,omitempty"`
	Cantrip             bool                  `json:"cantrip,omitempty"`
	PreparedRanks       []int                 `json:"preparedRanks,omitempty"`
	SpellcastingType    *string               `json:"spellcastingType,omitempty"`
	Name                string                `json:"name"`
	Type                string                `json:"type"`
	Slug                *string               `json:"slug"`
	Image               *string               `json:"image"`
	Category            CharacterItemCategory `json:"category"`
	SystemCategory      *string               `json:"systemCategory,omitempty"`
	ActionType          *string               `json:"actionType,omitempty"`
	Quantity            *int                  `json:"quantity"`
	Level               *int                  `json:"level"`
	Equipped            *CharacterEquippedDto `json:"equipped"`
	ContainerID         *string               `json:"containerId,omitempty"`
	Usage               *string               `json:"usage,omitempty"`
	SpellcastingEntryID *string               `json:"spellcastingEntryId,omitempty"`
	Tradition           *string               `json:"tradition,omitempty"`
	SpellList           *string               `json:"spellList,omitempty"`
	Prepared            *bool                 `json:"prepared,omitempty"`
	CompendiumSource    *string               `json:"compendiumSource"`
	RuleElements        []RuleElementDto      `json:"ruleElements,omitempty"`
	Armor               *ArmorItemDataDto     `json:"armor,omitempty"`
	Runes               *ItemRuneDataDto      `json:"runes,omitempty"`
}

type CharacterEquippedDto struct {
	CarryType *string `json:"carryType"`
	HandsHeld *int    `json:"handsHeld"`
	InSlot    *bool   `json:"inSlot"`
	Invested  *bool   `json:"invested"`
}

type DecodedItem struct {
	Foundry     FoundryItemDto
	Character   CharacterItemDto
	RawSnapshot json.RawMessage
}

// DecodedActor keeps the normalized projection and one authoritative raw
// snapshot. Item documents are already included in RawSnapshot.
type DecodedActor struct {
	Character   CharacterDto
	RawSnapshot json.RawMessage
}

// CharacterStatisticsInputDto contains normalized Actor inputs and explicitly marked
// ability scores reconstructed from complete PF2e boost selections. Missing source
// data remains nil; proficiency is never inferred from incomplete progression data.
type CharacterStatisticsInputDto struct {
	Abilities              map[string]AbilityScoreDto `json:"abilities"`
	ReconstructedAbilities map[string]AbilityScoreDto `json:"reconstructedAbilities,omitempty"`
	AbilitiesDerived       bool                       `json:"abilitiesDerived,omitempty"`
	Proficiencies          CharacterProficienciesDto  `json:"proficiencies"`
}

// AbilityScoreDto distinguishes the score and modifier; Derived marks values
// reconstructed from build selections rather than stored on the Actor.
type AbilityScoreDto struct {
	Score    *int  `json:"score"`
	Modifier *int  `json:"modifier"`
	Derived  bool  `json:"derived,omitempty"`
}

type CharacterProficienciesDto struct {
	Perception *ProficiencyDto           `json:"perception"`
	Saves      map[string]ProficiencyDto `json:"saves"`
	Defenses   map[string]ProficiencyDto `json:"defenses"`
}

type ProficiencyDto struct {
	Rank   *int   `json:"rank"`
	Source string `json:"source,omitempty"`
}

// ClassProficiencyProfileDto is a class item's progression baseline, not
// proof of the character's current rank. Consumers must not silently treat it
// as an Actor-prepared proficiency value.
type ClassProficiencyProfileDto struct {
	PerceptionRank   *int                       `json:"perceptionRank"`
	SavingThrows     map[string]int             `json:"savingThrows"`
	Defenses         map[string]int             `json:"defenses"`
	Attacks          map[string]int             `json:"attacks"`
	SpellcastingRank *int                       `json:"spellcastingRank"`
	TrainedSkills    []string                   `json:"trainedSkills"`
	AdditionalSkills *int                       `json:"additionalSkills"`
	Features         []ClassFeatureReferenceDto `json:"features"`
	RuleElements     []RuleElementDto           `json:"ruleElements,omitempty"`
}

// ClassFeatureReferenceDto identifies a feature listed in the class item's
// progression table. It is a compendium reference, not a resolved feature.
type ClassFeatureReferenceDto struct {
	UUID  string  `json:"uuid"`
	Name  string  `json:"name"`
	Level *int    `json:"level"`
	Image *string `json:"image,omitempty"`
}

// RuleElementDto preserves the declarative rule element fields needed for
// later, explicitly supported interpretation. It is never executed by decoder.
type RuleElementDto struct {
	Key          string          `json:"key"`
	Mode         string          `json:"mode,omitempty"`
	Path         string          `json:"path,omitempty"`
	Value        json.RawMessage `json:"value,omitempty"`
	Flag         string          `json:"flag,omitempty"`
	Selection    json.RawMessage `json:"selection,omitempty"`
	Predicate    json.RawMessage `json:"predicate,omitempty"`
	Selector     json.RawMessage `json:"selector,omitempty"`
	ModifierType string          `json:"modifierType,omitempty"`
	Raw          json.RawMessage `json:"raw"`
}

type ArmorItemDataDto struct {
	Category *string `json:"category"`
	ACBonus  *int    `json:"acBonus"`
	DexCap   *int    `json:"dexCap"`
	Strength *int    `json:"strength"`
}

type ItemRuneDataDto struct {
	Potency   *int     `json:"potency"`
	Resilient *int     `json:"resilient"`
	Property  []string `json:"property"`
}

type ModifierKind string

const (
	ModifierItem        ModifierKind = "item"
	ModifierPotency     ModifierKind = "potency"
	ModifierResilient   ModifierKind = "resilient"
	ModifierUntyped     ModifierKind = "untyped"
	ModifierBase        ModifierKind = "base"
	ModifierAbility     ModifierKind = "ability"
	ModifierProficiency ModifierKind = "proficiency"
	ModifierArmor       ModifierKind = "armor"
	ModifierRune        ModifierKind = "rune"
)

// DerivedStatDto is a calculated value plus its auditable arithmetic.
// Value is nil unless the calculation is complete.
type DerivedStatDto struct {
	Value     *int                   `json:"value"`
	Status    CalculationStatus      `json:"status"`
	Breakdown []ModifierBreakdownDto `json:"breakdown"`
	Warnings  []string               `json:"warnings,omitempty"`
}

type CalculationStatus string

const (
	CalculationComplete    CalculationStatus = "complete"
	CalculationIncomplete  CalculationStatus = "incomplete"
	CalculationUnavailable CalculationStatus = "unavailable"
)

type ModifierBreakdownDto struct {
	Source string       `json:"source"`
	Kind   ModifierKind `json:"kind"`
	Value  int          `json:"value"`
}

type CalculatedProficiencyRankDto struct {
	Rank    int      `json:"rank"`
	Sources []string `json:"sources"`
}

type CharacterProficiencyRanksDto struct {
	Perception             *CalculatedProficiencyRankDto           `json:"perception,omitempty"`
	Saves                  map[string]CalculatedProficiencyRankDto `json:"saves"`
	Defenses               map[string]CalculatedProficiencyRankDto `json:"defenses"`
	Attacks                map[string]CalculatedProficiencyRankDto `json:"attacks"`
	Spellcasting           *CalculatedProficiencyRankDto           `json:"spellcasting,omitempty"`
	SpellcastingTraditions map[string]CalculatedProficiencyRankDto `json:"spellcastingTraditions"`
	Skills                 map[string]CalculatedProficiencyRankDto `json:"skills"`
	Warnings               []string                                `json:"warnings,omitempty"`
	Complete               bool                                    `json:"complete"`
}

type CharacterDerivedStatsDto struct {
	Proficiencies CharacterProficiencyRanksDto `json:"proficiencies"`
	Perception    DerivedStatDto               `json:"perception"`
	AC            DerivedStatDto               `json:"ac"`
	Saves         map[string]DerivedStatDto    `json:"saves"`
	Skills        map[string]DerivedStatDto    `json:"skills"`
	Version       string                       `json:"calculationVersion"`
}
