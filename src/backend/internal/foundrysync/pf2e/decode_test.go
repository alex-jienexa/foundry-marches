package pf2e

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const fixtureActor = "mxgc0DRWC8bA9ruI-20261118T073146Z-actor.json"
const fixtureResponse = "mxgc0DRWC8bA9ruI-20261118T073146Z-response.json"

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "docs", "foundry", name)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath(t, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestDecodeActorRealFixture(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil {
		t.Fatal(err)
	}
	c := decoded.Character
	if c.Identity.ID != "mxgc0DRWC8bA9ruI" || c.Identity.Name != "АНДРЕЙ" {
		t.Fatalf("unexpected identity: %+v", c.Identity)
	}
	if c.Source.CoreVersion != "14.367" || c.Source.SystemID != "pf2e" || c.Source.SystemVersion != "8.7.0" {
		t.Fatalf("unexpected source: %+v", c.Source)
	}
	if c.Source.MigrationVersion == nil || *c.Source.MigrationVersion != 0.961 {
		t.Fatalf("unexpected migration version: %+v", c.Source.MigrationVersion)
	}
	if c.Progression.Level == nil || *c.Progression.Level != 1 || c.Progression.Experience.Value == nil || *c.Progression.Experience.Value != 0 {
		t.Fatalf("unexpected progression: %+v", c.Progression)
	}
	if c.Attributes.HP == nil || c.Attributes.HP.Current == nil || *c.Attributes.HP.Current != 20 {
		t.Fatalf("unexpected HP: %+v", c.Attributes.HP)
	}
	if c.Attributes.HP.Max == nil || *c.Attributes.HP.Max != 20 {
		t.Fatalf("maximum HP should be derived from class, ancestry, CON and level: %+v", c.Attributes.HP)
	}
	if c.Resources.HeroPoints == nil || c.Resources.HeroPoints.Value == nil || *c.Resources.HeroPoints.Value != 1 {
		t.Fatalf("unexpected hero points: %+v", c.Resources.HeroPoints)
	}
	if len(c.Items) != 16 {
		t.Fatalf("got %d Items, want 16", len(c.Items))
	}
	if c.Build.Class == nil || c.Build.Class.Name != "Воин / Fighter" {
		t.Fatalf("class reference not decoded: %+v", c.Build.Class)
	}
	if c.Build.Background == nil || c.Build.Background.Name != "Канонир / Cannoneer" {
		t.Fatalf("background reference not decoded: %+v", c.Build.Background)
	}
	if c.Build.KeyAbility == nil || *c.Build.KeyAbility != "str" {
		t.Fatalf("key ability not decoded: %+v", c.Build.KeyAbility)
	}
	if _, ok := c.Skills.Standard["thievery"]; !ok {
		t.Fatal("thievery skill missing")
	}
	if skill, ok := c.Skills.Standard["athletics"]; !ok || skill.Rank == nil || *skill.Rank != 1 {
		t.Fatalf("background trainedSkills.value must grant trained Athletics: %+v", c.Skills.Standard)
	}
	if skill, ok := c.Skills.Lore["lore-warfare"]; !ok || skill.Rank == nil || *skill.Rank != 1 {
		t.Fatalf("background trainedSkills.lore must grant trained Warfare Lore: %+v", c.Skills.Lore)
	}
	if c.Biography.Age == nil || *c.Biography.Age != "" {
		t.Fatalf("empty but present age should be preserved: %+v", c.Biography.Age)
	}
	if len(decoded.RawSnapshot) == 0 {
		t.Fatal("raw Actor snapshot was not retained")
	}

	var weapon, armor, potion, coin *CharacterItemDto
	for i := range c.Items {
		switch c.Items[i].Type {
		case "weapon":
			weapon = &c.Items[i]
		case "armor":
			armor = &c.Items[i]
		case "consumable":
			potion = &c.Items[i]
		case "treasure":
			if c.Items[i].Name == "Серебряные монеты / Silver Pieces" {
				coin = &c.Items[i]
			}
		}
	}
	if weapon == nil || weapon.Quantity == nil || *weapon.Quantity != 1 {
		t.Fatalf("weapon quantity not decoded: %+v", weapon)
	}
	if armor == nil || armor.Equipped == nil || armor.Equipped.Invested == nil || !*armor.Equipped.Invested {
		t.Fatalf("armor equipped state not decoded: %+v", armor)
	}
	if potion == nil || potion.Level == nil || *potion.Level != 3 {
		t.Fatalf("potion level not decoded: %+v", potion)
	}
	if coin == nil || coin.Quantity == nil || *coin.Quantity != 34 {
		t.Fatalf("coin quantity not decoded: %+v", coin)
	}
}

func TestDecodePostSessionActorSnapshot(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "testdata", "actor-after-session.json"))
	if err != nil {
		t.Fatalf("read post-session Actor fixture: %v", err)
	}
	decoded, err := DecodeActor(data)
	if err != nil {
		t.Fatal(err)
	}
	character := decoded.Character
	if character.Progression.Level == nil || *character.Progression.Level != 2 {
		t.Fatalf("level = %+v, want 2", character.Progression.Level)
	}
	if character.Progression.Experience.Value == nil || *character.Progression.Experience.Value != 400 {
		t.Fatalf("XP = %+v, want 400", character.Progression.Experience.Value)
	}
	if character.Build.KeyAbility == nil || *character.Build.KeyAbility != "int" {
		t.Fatalf("key ability = %+v, want class-defined INT", character.Build.KeyAbility)
	}
	if !character.Statistics.AbilitiesDerived || len(character.Statistics.Abilities) != 0 || len(character.Statistics.ReconstructedAbilities) != 6 {
		t.Fatalf("complete post-session build selections should derive six display-only abilities: %+v", character.Statistics)
	}
	if got := character.Statistics.ReconstructedAbilities["int"]; got.Score == nil || *got.Score != 16 || !got.Derived {
		t.Fatalf("INT = %+v, want derived score 16", got)
	}
	if got := character.Statistics.ReconstructedAbilities["cha"]; got.Score == nil || *got.Score != 16 {
		t.Fatalf("CHA = %+v, want score 16", got)
	}

	if character.Attributes.HP == nil || character.Attributes.HP.Current == nil || *character.Attributes.HP.Current != 2 {
		t.Fatalf("current HP = %+v, want 2", character.Attributes.HP)
	}
	if character.Attributes.HP.Max == nil || *character.Attributes.HP.Max != 24 {
		t.Fatalf("derived maximum HP = %+v, want 24", character.Attributes.HP)
	}
	if len(character.Items) != 46 {
		t.Fatalf("decoded Actor Items = %d, want 46", len(character.Items))
	}
	inventoryCount := 0
	for _, item := range character.Items {
		if item.Category == ItemCategoryEquipment {
			inventoryCount++
		}
	}
	if inventoryCount != 16 {
		t.Fatalf("inventory item count = %d, want 16 across equipment, consumables, backpack and treasure", inventoryCount)
	}
	if character.Attributes.Dying != nil {
		t.Fatalf("snapshot has no dying condition and must not synthesize one: %+v", character.Attributes.Dying)
	}
	if character.Identity.Image == nil || !strings.Contains(*character.Identity.Image, "nm26zlUZr0rpxk6pxyttyDXHAOl5w6N1GDsJrYdm") {
		t.Fatalf("portrait path was not preserved: %+v", character.Identity.Image)
	}
}

func TestDecodeRelayEnvelopeFixture(t *testing.T) {
	decoded, err := DecodeRelayResponse(readFixture(t, fixtureResponse))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Character.Identity.ID != "mxgc0DRWC8bA9ruI" {
		t.Fatalf("unexpected Actor ID %q", decoded.Character.Identity.ID)
	}
}

func TestDecodeItemUnknownTypeIsPreserved(t *testing.T) {
	raw := []byte(`{"_id":"custom1","name":"Custom","type":"future-type","system":{"quantity":7,"newField":{"x":1}},"flags":{"module":{"keep":true}}}`)
	item, err := DecodeItem(raw)
	if err != nil {
		t.Fatal(err)
	}
	if item.Character.Category != ItemCategoryEquipment {
		t.Fatalf("an unknown Actor Item with a quantity must remain visible as inventory; category = %q", item.Character.Category)
	}
	if item.Character.Quantity == nil || *item.Character.Quantity != 7 {
		t.Fatal("quantity missing")
	}
	if _, ok := item.Foundry.System["newField"]; !ok {
		t.Fatal("unknown system field lost from raw Item DTO")
	}
	if _, ok := item.Foundry.Flags["module"]; !ok {
		t.Fatal("unknown flags lost from raw Item DTO")
	}
	if string(item.RawSnapshot) != string(raw) {
		t.Fatal("standalone Item raw snapshot was not retained")
	}
}

func TestDecodeSpellcastingTraditionAndList(t *testing.T) {
	raw := []byte(`{"_id":"spellcaster","name":"Spellcaster","type":"character","items":[
		{"_id":"entry1","name":"Арканная книга","type":"spellcastingEntry","system":{"tradition":{"value":"arcane"},"prepared":{"value":"prepared"},"slots":{"slot1":{"prepared":[{"id":"spell1"}]}}}},
		{"_id":"spell1","name":"Магическая стрела","type":"spell","system":{"location":"entry1","level":{"value":1}}}
	]}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Character.Items) != 2 {
		t.Fatalf("items = %d, want spell entry and spell", len(decoded.Character.Items))
	}
	var spell *CharacterItemDto
	for i := range decoded.Character.Items {
		if decoded.Character.Items[i].Type == "spell" {
			spell = &decoded.Character.Items[i]
		}
	}
	if spell == nil {
		t.Fatal("spell Item missing")
	}
	if spell.Level == nil || *spell.Level != 1 {
		t.Fatalf("spell rank = %+v", spell.Level)
	}
	if spell.Tradition == nil || *spell.Tradition != "arcane" {
		t.Fatalf("spell tradition = %+v", spell.Tradition)
	}
	if spell.SpellList == nil || *spell.SpellList != "Арканная книга" {
		t.Fatalf("spell list = %+v", spell.SpellList)
	}
	if spell.Prepared == nil || !*spell.Prepared {
		t.Fatalf("prepared state = %+v", spell.Prepared)
	}
}

func TestDecodeActorRejectsIncompleteItemsCollection(t *testing.T) {
	for _, raw := range []string{
		`{"_id":"a1","type":"character"}`,
		`{"_id":"a1","type":"character","items":null}`,
		`{"_id":"a1","type":"character","items":{}}`,
	} {
		if _, err := DecodeActor([]byte(raw)); err == nil {
			t.Fatalf("expected error for %s", raw)
		}
	}
}

func TestDecodeActorRejectsDuplicateItemIDs(t *testing.T) {
	raw := []byte(`{"_id":"a1","type":"character","items":[{"_id":"same","type":"feat"},{"_id":"same","type":"action"}]}`)
	if _, err := DecodeActor(raw); err == nil {
		t.Fatal("expected duplicate Item ID to be rejected")
	}
}

func TestDecodeActorKeepsMissingOptionalValuesNil(t *testing.T) {
	raw := []byte(`{"_id":"a1","type":"character","name":"Test","items":[],"system":{"abilities":null,"skills":{}}}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Character.Progression.Level != nil {
		t.Fatal("missing level must remain nil")
	}
	if decoded.Character.Attributes.HP != nil {
		t.Fatal("missing HP must remain nil")
	}
}

func TestDecodeSkillsSeparatesLoreWithoutDerivingIt(t *testing.T) {
	raw := []byte(`{"lore-warfare":{"rank":1},"athletics":{"rank":2},"thievery":{"rank":3}}`)
	skills := decodeSkills(raw)
	if _, ok := skills.Lore["lore-warfare"]; !ok {
		t.Fatal("Lore skill not separated")
	}
	if _, ok := skills.Standard["athletics"]; !ok {
		t.Fatal("standard skill missing")
	}
	if len(skills.Standard) != 2 {
		t.Fatalf("unexpected standard skills: %+v", skills.Standard)
	}
}

func TestDecodeMaximumHPUsesAncestryClassAndConstitutionPerLevel(t *testing.T) {
	for _, test := range []struct {
		level int
		want  int
	}{
		{level: 1, want: 19},
		{level: 3, want: 41},
	} {
		raw := []byte(fmt.Sprintf(`{
			"_id":"hp-actor",
			"name":"Андрей",
			"type":"character",
			"system":{
				"details":{"level":{"value":%d}},
				"attributes":{"hp":{"value":19}},
				"abilities":{"con":{"mod":1}}
			},
			"items":[
				{"_id":"human","name":"Человек","type":"ancestry","system":{"hp":8}},
				{"_id":"fighter","name":"Воин","type":"class","system":{"hp":10}}
			]
		}`, test.level))
		decoded, err := DecodeActor(raw)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Character.Attributes.HP == nil || decoded.Character.Attributes.HP.Max == nil ||
			*decoded.Character.Attributes.HP.Max != test.want {
			t.Errorf("level %d maximum HP = %+v, want %d", test.level, decoded.Character.Attributes.HP, test.want)
		}
	}
}

func TestDecodeMaximumHPAddsSimplePerLevelFlatModifier(t *testing.T) {
	raw := []byte(`{
		"_id":"toughness-actor",
		"name":"Персонаж с Живучестью",
		"type":"character",
		"system":{
			"details":{"level":{"value":3}},
			"attributes":{"hp":{"value":20}},
			"abilities":{"con":{"mod":1}}
		},
		"items":[
			{"_id":"human","name":"Человек","type":"ancestry","system":{"hp":8}},
			{"_id":"fighter","name":"Воин","type":"class","system":{"hp":10}},
			{"_id":"toughness","name":"Живучесть","type":"feat","system":{"rules":[
				{"key":"FlatModifier","selector":"hp","type":"untyped","value":"@actor.level"}
			]}}
		]
	}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Character.Attributes.HP == nil || decoded.Character.Attributes.HP.Max == nil ||
		*decoded.Character.Attributes.HP.Max != 44 {
		t.Fatalf("maximum HP with per-level FlatModifier = %+v, want 44", decoded.Character.Attributes.HP)
	}
}

func TestDecodeAncestryFeatureProgression(t *testing.T) {
	raw := []byte(`{
		"_id":"ancestry-features",
		"name":"Тест",
		"type":"character",
		"system":{"details":{"level":{"value":3}}},
		"items":[{
			"_id":"human",
			"name":"Человек",
			"type":"ancestry",
			"system":{"items":{
				"later":{"uuid":"Compendium.pf2e.ancestryfeatures.Item.later","name":"Поздняя особенность","level":5},
				"early":{"uuid":"Compendium.pf2e.ancestryfeatures.Item.early","name":"Ранняя особенность","level":1}
			}}
		}]
	}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	features := decoded.Character.Build.AncestryFeatures
	if len(features) != 2 {
		t.Fatalf("ancestry features = %+v, want 2 references", features)
	}
	if features[0].Name != "Ранняя особенность" || features[1].Name != "Поздняя особенность" {
		t.Fatalf("ancestry features are not sorted by level: %+v", features)
	}
}

func TestDecodeFocusSpellcastingEntryType(t *testing.T) {
	raw := []byte(`{
		"_id":"focus-actor",
		"name":"Оракул",
		"type":"character",
		"system":{"details":{"level":{"value":1}}},
		"items":[
			{
				"_id":"focus-entry",
				"name":"Фокусные заклинания",
				"type":"spellcastingEntry",
				"system":{"category":{"value":"focus"},"prepared":{"value":"prepared"},"tradition":{"value":"divine"}}
			},
			{
				"_id":"focus-spell",
				"name":"Свет",
				"type":"spell",
				"system":{"location":"focus-entry","level":{"value":1},"traits":{"value":["cantrip"]}}
			}
		]
	}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range decoded.Character.Items {
		if item.ID != "focus-spell" {
			continue
		}
		if item.SpellcastingType == nil || *item.SpellcastingType != "focus" {
			t.Fatalf("focus spellcasting type = %v, want focus", item.SpellcastingType)
		}
		if !item.Cantrip {
			t.Fatal("focus cantrip classification was not preserved")
		}
		return
	}
	t.Fatal("focus spell was not decoded")
}

func TestDecodeStatisticsKeepsMissingValuesUnknown(t *testing.T) {
	raw := []byte(`{"_id":"unknown","type":"character","items":[],"system":{"abilities":null}}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	stats := decoded.Character.Statistics
	if len(stats.Abilities) != 0 || stats.AbilitiesDerived {
		t.Fatalf("missing Actor values without complete boost selections must remain unknown: %+v", stats)
	}
	if stats.Proficiencies.Perception != nil || len(stats.Proficiencies.Saves) != 0 {
		t.Fatalf("missing Actor proficiencies must remain unavailable: %+v", stats.Proficiencies)
	}
}

func TestDecodeAbilityScoresFromBuildBoostSelections(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil { t.Fatal(err) }
	stats := decoded.Character.Statistics
	if !stats.AbilitiesDerived || len(stats.Abilities) != 0 || len(stats.ReconstructedAbilities) != 6 { t.Fatalf("expected six display-only derived ability scores: %+v", stats) }
	want := map[string]int{"str": 18, "dex": 12, "con": 14, "int": 10, "wis": 14, "cha": 10}
	for ability, score := range want {
		got := stats.ReconstructedAbilities[ability]
		if got.Score == nil || *got.Score != score || !got.Derived { t.Errorf("%s = %+v, want derived score %d", ability, got, score) }
	}
	class := decoded.Character.Build.ClassProficiencies
	if class == nil || class.PerceptionRank == nil || *class.PerceptionRank != 2 { t.Fatalf("class progression profile not decoded: %+v", class) }
	if class.SavingThrows["fortitude"] != 2 || class.SavingThrows["reflex"] != 2 || class.SavingThrows["will"] != 1 { t.Fatalf("unexpected class save baseline: %+v", class.SavingThrows) }
	if class.AdditionalSkills == nil || *class.AdditionalSkills != 3 { t.Fatalf("class additional skill count missing: %+v", class.AdditionalSkills) }
}
func TestDecodeAbilityBoostsApplyInLevelOrderAfterAncestryFlaw(t *testing.T) {
	raw := []byte(`{"_id":"ordered-boosts","type":"character","items":[
		{"_id":"ancestry","type":"ancestry","system":{"boosts":{},"flaws":{"0":{"value":["dex"],"selected":"dex"}}}},
		{"_id":"background","type":"background","system":{"boosts":{"0":{"value":["str","dex"],"selected":"dex"}}}},
		{"_id":"class","type":"class","system":{"keyAbility":{"value":["dex"],"selected":"dex"}}}
	],"system":{"details":{"level":{"value":9}},"build":{"attributes":{"boosts":{
		"1":["dex","str","con","int"],"5":["dex","str","con","wis"],"9":["dex","str","int","cha"]
	}}},"abilities":null}}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	got := decoded.Character.Statistics.ReconstructedAbilities["dex"]
	if got.Score == nil || *got.Score != 18 {
		t.Fatalf("DEX = %+v, want 18 after ancestry flaw and subsequent boosts", got)
	}
}

func TestDecodeAbilityScoresIncludeClassItemBoostSelections(t *testing.T) {
	raw := []byte(`{"_id":"class-boosts","type":"character","items":[
		{"_id":"ancestry","type":"ancestry","system":{"boosts":{"0":{"value":["str","dex","con","int","wis","cha"],"selected":"cha"}}}},
		{"_id":"background","type":"background","system":{"boosts":{"0":{"value":["str","dex"],"selected":"str"}}}},
		{"_id":"class","type":"class","system":{"boosts":{"0":{"value":["int","wis"],"selected":"int"}}}}
	],"system":{"details":{"level":{"value":1}},"build":{"attributes":{"boosts":{"1":["dex","con","wis","cha"]}}},"abilities":null}}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	want := map[string]int{"str":12,"dex":12,"con":12,"int":12,"wis":12,"cha":14}
	for ability, score := range want {
		got := decoded.Character.Statistics.ReconstructedAbilities[ability]
		if got.Score == nil || *got.Score != score || !got.Derived { t.Errorf("%s reconstructed score = %+v, want %d", ability, got, score) }
	}
}

func TestDecodeAbilityScoresApplyAncestryAndVoluntaryFlaws(t *testing.T) {
	raw := []byte(`{"_id":"flaws","type":"character","items":[
		{"_id":"ancestry","type":"ancestry","system":{
			"boosts":{"0":{"value":["str","dex"],"selected":"str"}},
			"flaws":{"0":{"value":["dex"],"selected":"dex"}},
			"voluntary":{"boost":"wis","flaws":["con"]}
		}},
		{"_id":"background","type":"background","system":{"boosts":{"0":{"value":["str","dex"],"selected":"dex"}}}},
		{"_id":"class","type":"class","system":{"keyAbility":{"value":["str"],"selected":null}}}
	],"system":{"details":{"level":{"value":1}},"build":{"attributes":{"boosts":{"1":["str","dex","con","int"]}}},"abilities":null}}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	want := map[string]int{"str":16,"dex":12,"con":10,"int":12,"wis":12,"cha":10}
	for ability, score := range want {
		got := decoded.Character.Statistics.ReconstructedAbilities[ability]
		if got.Score == nil || *got.Score != score || !got.Derived { t.Errorf("%s reconstructed score = %+v, want %d", ability, got, score) }
	}
}

func TestDecodeSkillsIncludesClassAndBackgroundTraining(t *testing.T) {
	raw := []byte(`{"_id":"skills","type":"character","items":[
		{"_id":"class","type":"class","system":{"trainedSkills":{"value":["religion"],"additional":2},"rules":[{"key":"ChoiceSet","flag":"skillChoice","selection":"diplomacy"},{"key":"ActiveEffectLike","mode":"upgrade","path":"system.skills.{item|flags.system.rulesSelections.skillChoice}.rank","value":1}]}},
		{"_id":"background","type":"background","system":{"trainedSkills":{"value":["athletics"],"lore":["Warfare Lore"]}}}
	],"system":{"skills":{}}}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	for _, slug := range []string{"religion", "athletics", "diplomacy"} {
		if got := decoded.Character.Skills.Standard[slug].Rank; got == nil || *got != 1 { t.Errorf("%s rank = %v, want trained (1)", slug, got) }
	}
	if got := decoded.Character.Skills.Lore["lore-warfare"].Rank; got == nil || *got != 1 { t.Errorf("Warfare Lore rank = %v, want trained (1)", got) }
}

func TestDecodeSkillFeatAssociationFromPrerequisite(t *testing.T) {
	item, err := DecodeItem([]byte(`{"_id":"feat","name":"Bon Mot","type":"feat","system":{"category":"skill","prerequisites":{"value":[{"value":"Обучен Дипломатии"}]}}}`))
	if err != nil { t.Fatal(err) }
	if item.Character.Skill == nil || *item.Character.Skill != "diplomacy" { t.Fatalf("skill association = %v, want diplomacy", item.Character.Skill) }
}

func TestDecodePreparedSpellSlotsAndSpellbook(t *testing.T) {
	raw := []byte(`{"_id":"caster","type":"character","items":[
		{"_id":"entry","name":"Арканная книга","type":"spellcastingEntry","system":{
			"tradition":{"value":"arcane"},"prepared":{"value":"prepared"},
			"slots":{"slot0":{"prepared":[{"id":"cantrip","expended":false}],"value":0,"max":0},
			"slot1":{"prepared":[{"id":"prepared-spell","expended":false}],"value":0,"max":1},
			"slot2":{"prepared":[],"value":0,"max":1}}
		}},
		{"_id":"cantrip","name":"Свет","type":"spell","system":{"location":"entry","level":{"value":0}}},
		{"_id":"prepared-spell","name":"Магическая стрела","type":"spell","system":{"location":"entry","level":{"value":1}}},
		{"_id":"spellbook","name":"Щит","type":"spell","system":{"location":"entry","level":{"value":1}}}
	]}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	spells := make(map[string]CharacterItemDto)
	for _, item := range decoded.Character.Items { if item.Type == "spell" { spells[item.ID] = item } }
	if got := spells["cantrip"]; got.Prepared == nil || !*got.Prepared || len(got.PreparedRanks) != 1 || got.PreparedRanks[0] != 0 { t.Errorf("cantrip slot data = %+v, want prepared at slot0", got) }
	if got := spells["prepared-spell"]; got.Prepared == nil || !*got.Prepared || len(got.PreparedRanks) != 1 || got.PreparedRanks[0] != 1 { t.Errorf("prepared spell slot data = %+v, want prepared at rank 1", got) }
	if got := spells["spellbook"]; got.Prepared == nil || *got.Prepared || len(got.PreparedRanks) != 0 { t.Errorf("spellbook spell = %+v, want available for preparation only", got) }
}

func TestDecodeSpontaneousCantripClassification(t *testing.T) {
	raw := []byte(`{"_id":"caster","type":"character","items":[
		{"_id":"entry","name":"Гармонизация явлений / Apparition Attunement","type":"spellcastingEntry","system":{"tradition":{"value":"divine"},"prepared":{"value":"spontaneous"}}},
		{"_id":"cantrip","name":"Престидижитация / Prestidigitation","type":"spell","system":{"location":"entry","level":{"value":1},"traits":{"value":["cantrip","manipulate"]}}},
		{"_id":"spell","name":"Оплетающая лоза / Tangle Vine","type":"spell","system":{"location":"entry","level":{"value":1},"traits":{"value":["plant"]}}}
	]}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	items := make(map[string]CharacterItemDto)
	for _, item := range decoded.Character.Items {
		if item.Type == "spell" {
			items[item.ID] = item
		}
	}
	cantrip := items["cantrip"]
	if !cantrip.Cantrip || cantrip.SpellcastingType == nil || *cantrip.SpellcastingType != "spontaneous" {
		t.Fatalf("cantrip = %+v, want spontaneous cantrip", cantrip)
	}
	spell := items["spell"]
	if spell.Cantrip || spell.SpellcastingType == nil || *spell.SpellcastingType != "spontaneous" {
		t.Fatalf("spell = %+v, want ordinary spontaneous spell", spell)
	}
}

func TestDecodeEquipmentCalculationInputs(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil {
		t.Fatal(err)
	}
	var armor *CharacterItemDto
	for i := range decoded.Character.Items {
		if decoded.Character.Items[i].Type == "armor" {
			armor = &decoded.Character.Items[i]
			break
		}
	}
	if armor == nil || armor.Armor == nil || armor.Runes == nil {
		t.Fatalf("armor calculation inputs missing: %+v", armor)
	}
	if armor.Armor.Category == nil || *armor.Armor.Category != "light" {
		t.Fatalf("unexpected armor category: %+v", armor.Armor.Category)
	}
	if armor.Armor.ACBonus == nil || *armor.Armor.ACBonus != 2 ||
		armor.Armor.DexCap == nil || *armor.Armor.DexCap != 3 {
		t.Fatalf("unexpected armor values: %+v", armor.Armor)
	}
	if armor.Runes.Potency == nil || *armor.Runes.Potency != 1 ||
		armor.Runes.Resilient == nil || *armor.Runes.Resilient != 1 {
		t.Fatalf("unexpected rune values: %+v", armor.Runes)
	}
}

func TestDecodeExplicitAbilityAndProficiencyInputs(t *testing.T) {
	raw := []byte(`{"_id":"a1","type":"character","items":[],"system":{
		"abilities":{"str":{"value":18,"mod":4},"dex":{"value":14,"mod":2}},
		"proficiencies":{
			"perception":{"rank":2},
			"savingThrows":{"fortitude":{"rank":3},"reflex":{"rank":2}},
			"defenses":{"unarmored":{"rank":1}}
		}
	}}`)
	decoded, err := DecodeActor(raw)
	if err != nil {
		t.Fatal(err)
	}
	stats := decoded.Character.Statistics
	if got := stats.Abilities["str"]; got.Score == nil || *got.Score != 18 || got.Modifier == nil || *got.Modifier != 4 {
		t.Fatalf("unexpected Strength input: %+v", got)
	}
	if stats.Proficiencies.Perception == nil || stats.Proficiencies.Perception.Rank == nil || *stats.Proficiencies.Perception.Rank != 2 {
		t.Fatalf("perception proficiency missing: %+v", stats.Proficiencies.Perception)
	}
	if got := stats.Proficiencies.Saves["fortitude"]; got.Rank == nil || *got.Rank != 3 {
		t.Fatalf("Fortitude proficiency missing: %+v", got)
	}
	if got := stats.Proficiencies.Defenses["unarmored"]; got.Rank == nil || *got.Rank != 1 {
		t.Fatalf("unarmored proficiency missing: %+v", got)
	}
}

func TestDecodeCompleteClassProgressionAndFeatureReferences(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil {
		t.Fatal(err)
	}
	class := decoded.Character.Build.ClassProficiencies
	if class == nil {
		t.Fatal("class progression missing")
	}
	if class.PerceptionRank == nil || *class.PerceptionRank != 2 {
		t.Fatalf("perception baseline: %+v", class.PerceptionRank)
	}
	if class.SavingThrows["fortitude"] != 2 || class.SavingThrows["reflex"] != 2 || class.SavingThrows["will"] != 1 {
		t.Fatalf("saving throw baselines: %+v", class.SavingThrows)
	}
	if class.Defenses["heavy"] != 1 || class.Defenses["unarmored"] != 1 {
		t.Fatalf("defense baselines: %+v", class.Defenses)
	}
	if class.Attacks["martial"] != 2 || class.Attacks["advanced"] != 1 || class.Attacks["simple"] != 2 || class.Attacks["unarmed"] != 2 {
		t.Fatalf("attack baselines: %+v", class.Attacks)
	}
	if class.SpellcastingRank == nil || *class.SpellcastingRank != 0 {
		t.Fatalf("spellcasting baseline: %+v", class.SpellcastingRank)
	}
	if len(class.Features) != 16 {
		t.Fatalf("got %d class feature references, want 16", len(class.Features))
	}
	foundBravery := false
	for _, feature := range class.Features {
		if feature.Name == "Храбрость" {
			foundBravery = feature.UUID == "Compendium.pf2e.classfeatures.Item.GJKJafDGuX4BeAeN" &&
				feature.Level != nil && *feature.Level == 3
		}
	}
	if !foundBravery {
		t.Fatal("Bravery class feature reference was not preserved")
	}
	if len(class.RuleElements) != 2 || class.RuleElements[0].Key == "" {
		t.Fatalf("class Rule Elements not preserved: %+v", class.RuleElements)
	}
}

func TestDecodeRuleElementsOnActorItems(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range decoded.Character.Items {
		if item.Slug == nil || *item.Slug != "hefty-hauler" {
			continue
		}
		found = true
		if len(item.RuleElements) != 2 {
			t.Fatalf("Hefty Hauler Rule Elements: %+v", item.RuleElements)
		}
		if item.RuleElements[0].Key != "ActiveEffectLike" || len(item.RuleElements[0].Raw) == 0 {
			t.Fatalf("Rule Element projection/raw missing: %+v", item.RuleElements[0])
		}
	}
	if !found {
		t.Fatal("Hefty Hauler Actor Item not found")
	}
}

func TestDecodeActorPreservesFeatAndActionCategories(t *testing.T) {
	decoded, err := DecodeActor(readFixture(t, fixtureActor))
	if err != nil {
		t.Fatal(err)
	}

	found := map[string]bool{}
	for _, item := range decoded.Character.Items {
		if item.Slug == nil {
			continue
		}
		switch *item.Slug {
		case "double-slice":
			found["class-feat"] = item.Type == "feat" &&
				item.SystemCategory != nil && *item.SystemCategory == "class" &&
				item.ActionType != nil && *item.ActionType == "action"
		case "gloomseer":
			found["ancestry-feat"] = item.Type == "feat" &&
				item.SystemCategory != nil && *item.SystemCategory == "ancestry"
		case "reactive-strike":
			if item.Type == "action" {
				found["action"] = item.ActionType != nil && *item.ActionType == "reaction"
			} else if item.Type == "feat" {
				found["class-feature"] = item.SystemCategory != nil && *item.SystemCategory == "classfeature"
			}
		case "longsword":
			found["equipment"] = item.Category == ItemCategoryEquipment
		}
	}

	for _, category := range []string{"class-feat", "ancestry-feat", "action", "class-feature", "equipment"} {
		if !found[category] {
			t.Errorf("fixture did not expose expected category %q", category)
		}
	}
}


func TestDecodeAbilityScoresPreservePartialSourcesAndSingletonBoosts(t *testing.T) {
	raw := []byte(`{
		"_id":"partial-boosts","type":"character",
		"items":[
			{"_id":"tengu","type":"ancestry","system":{"boosts":{
				"0":{"value":["dex"],"selected":null},
				"1":{"value":[],"selected":null},
				"2":{"value":["str","dex","con","int","wis","cha"],"selected":"wis"}
			}}},
			{"_id":"background","type":"background","system":{"boosts":{
				"0":{"value":["str","con"],"selected":null}
			}}}
		],
		"system":{"details":{"level":{"value":1}},"build":{"attributes":{"boosts":{"1":["str","con"]}}},"abilities":null}
	}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	got := decoded.Character.Statistics.ReconstructedAbilities
	if len(got) != 6 { t.Fatalf("partial reconstruction should retain all six computed scores: %+v", got) }
	want := map[string]int{"str":14,"dex":12,"con":14,"int":10,"wis":12,"cha":10}
	for ability, score := range want {
		if got[ability].Score == nil || *got[ability].Score != score {
			t.Errorf("%s = %+v, want %d", ability, got[ability], score)
		}
	}
}

func TestDecodeAbilityScoresDoNotRequireMissingBackgroundOrClass(t *testing.T) {
	raw := []byte(`{
		"_id":"missing-build-components","type":"character",
		"items":[
			{"_id":"human","type":"ancestry","system":{"boosts":{"0":{"value":["con"],"selected":"con"}}}}
		],
		"system":{"details":{"level":{"value":1}},"abilities":null}
	}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	got := decoded.Character.Statistics.ReconstructedAbilities
	if got["con"].Score == nil || *got["con"].Score != 12 {
		t.Fatalf("ancestry boost should survive missing background and class: %+v", got)
	}
	if got["str"].Score == nil || *got["str"].Score != 10 {
		t.Fatalf("unmodified baseline should remain available: %+v", got["str"])
	}
}

func TestDecodeMaximumHPPreservesAvailableTerms(t *testing.T) {
	raw := []byte(`{
		"_id":"partial-hp","type":"character",
		"system":{"details":{"level":{"value":3}},"attributes":{"hp":{"value":10}},"abilities":{"con":{"mod":1}}},
		"items":[{"_id":"human","type":"ancestry","system":{"hp":8}}]
	}`)
	decoded, err := DecodeActor(raw)
	if err != nil { t.Fatal(err) }
	if decoded.Character.Attributes.HP == nil || decoded.Character.Attributes.HP.Max == nil || *decoded.Character.Attributes.HP.Max != 11 {
		t.Fatalf("ancestry HP plus known CON contribution should be retained without class: %+v", decoded.Character.Attributes.HP)
	}
}
