# Foundry PF2e Actor decoder

This package is a read-only boundary adapter. It unwraps Relay entity responses,
validates complete Actor snapshots, retains the full raw Actor document, and
projects verified stored fields into the versioned application Character DTO.

## Contract

- Foundry/Relay JSON remains the source of truth; unknown fields stay in the raw snapshot.
- Actor Items are represented once in Character.Items.
- Raw Item documents are already part of the raw Actor snapshot; they are not
  duplicated in every normalized Item.
- Item category is an application/UI hint, not a closed PF2e type registry.
- Normalized Items expose common fields; type-specific data remains in the raw snapshot.
- Missing and null values remain absent; no default quantity, equipment state,
  ability score, skill rank, or derived statistic is fabricated.
- The persisted Actor document does not contain prepared AC, maximum HP, or
  prepared Perception in the supplied fixture. This adapter does not calculate them.
- Skills come only from Actor.system.skills. Class/background selections and
  Rule Elements are never evaluated or used to synthesize skills.
- Lore entries are separated only when their Actor.system.skills key uses the
  `lore-` prefix; the original key is retained.
- Compendium references inside class/background Items are not expanded into
  Actor Items.
- This package performs no writes to Foundry and no PF2e rules calculations.

## Tests

Run from `src/backend`:

```sh
go test ./...
go vet ./...
```

The tests decode the committed Relay fixtures under the repository-level
`docs/foundry/` directory.


## Statistics and derived-stat DTO contracts

`CharacterDto.Statistics` contains only explicit Actor-level calculation inputs:
- ability score and modifier, when `system.abilities` contains them;
- current perception, saving-throw and defense proficiency ranks, when the Actor contains them under `system.proficiencies`;
- current skill ranks remain in `CharacterDto.Skills`.

Missing values stay absent. The decoder does not reconstruct ability scores from boosts, derive current proficiency from class progression, or execute PF2e rules.

`CharacterBuildDto.ClassProficiencies` is deliberately separate. It records the class item's progression baseline (`perception`, `savingThrows`, `defenses`, `trainedSkills`) and is **not** authoritative evidence of the character's current prepared rank. In particular, the `additional` class skill count is not used to invent selected skills.

Equipment calculation inputs are exposed only where their source data is structured and recognized:
- armor `acBonus`, `dexCap` and `strength`;
- armor/other item's rune `potency`, `resilient` and `property` values;
- the existing equipped state.

The raw Actor snapshot remains authoritative for all unrecognized fields.

`CharacterDerivedStatsDto` defines the future calculator output contract: each statistic has an optional value, a status (`complete`, `incomplete`, `unavailable`), a breakdown and warnings. A missing input must never be replaced with zero. The calculation version is explicit so future algorithm changes can be distinguished.

The current decoder only prepares these DTOs; it does not yet calculate AC, Perception, saves or skill totals. Temporary effects, situational modifiers, Rule Elements and other runtime-prepared values are outside this contract.


## Class progression and Rule Elements

`CharacterBuildDto.ClassProficiencies` now captures the class baseline from the class Item: perception, all saving throws, defense categories, attack categories, spellcasting rank, trained skills and additional skill count.

The class Item's `system.items` map is decoded as `ClassFeatureReferenceDto` entries (UUID, name, level and image). These are references to compendium documents, not embedded Actor Items. The decoder does not pretend that a reference contains its Rule Elements: the referenced document must be resolved separately before its rules can be interpreted.

Rule Elements present on the class Item and owned Actor Items are projected into `RuleElementDto`. The original rule JSON is retained in `Raw`, including unsupported fields. This is declarative data only: no predicates, formulas, grants, selectors, or other Rule Element behavior are executed by the decoder.

PF2e `ActiveEffectLike` can upgrade proficiency ranks, including through a `ChoiceSet`-selected path. The eventual calculator may support a deliberately narrow, documented subset (static numeric `upgrade` values and simple level brackets). Unsupported expressions, predicates, conditional rules and unresolved compendium references must produce warnings/incomplete results, never guessed ranks. This is necessary because Rule Elements can depend on selections, predicates, actor level and data-preparation phases.

Fighter fixture expectations: perception 2; Fortitude 2, Reflex 2, Will 1; defenses Heavy/Light/Medium/Unarmored 1; attacks Advanced 1, Martial 2, Simple 2, Unarmed 2; spellcasting 0. Its class progression table contains 15 feature references, two at level 1. The `fighterSkill` ChoiceSet and its ActiveEffectLike are preserved as source rules.
