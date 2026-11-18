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
- The decoder projects skills from Actor.system.skills and preserves class/item
  Rule Elements separately. The calculator may apply a narrow, explicitly
  supported static proficiency-upgrade subset (including a resolved ChoiceSet);
  it does not rewrite Character.Skills.
- Lore entries are separated only when their Actor.system.skills key uses the
  `lore-` prefix; the original key is retained.
- Compendium references inside class/background Items are not expanded into
  Actor Items.
- The decoder performs no writes to Foundry and does not execute PF2e rules.
  The sibling calculator package performs only documented, limited rank and
  derived-stat calculations; it is not a replacement for Foundry preparation.

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


## Proficiency calculator

The sibling `calculator` package resolves effective proficiency ranks from:
1. class progression baseline (Perception, saves, defenses, attacks and spellcasting);
2. explicit Actor ranks and Actor skill ranks, when present;
3. static numeric `ActiveEffectLike` rules with `mode: upgrade`, including simple
   level brackets and `ChoiceSet` selections preserved on the source item.

The resolver does not evaluate predicates, formulas, grants, `FlatModifier`,
conditional rules or other Rule Element kinds. Class feature references whose
level has been reached but whose compendium document is not present among Actor
Items are reported as warnings. A warning makes the rank result incomplete.

Run tests from `src/backend` with `go test ./...` and `go vet ./...`.


The calculator's `CalculateDerivedStats` currently supports Perception, AC, Fortitude, Reflex, Will and skills present in the normalized rank set. It uses PF2e proficiency bonuses (untrained 0; trained `level+2`; each subsequent rank adds 2), the stored ability modifier (or a modifier derived from an explicitly stored ability score), armor category/AC bonus/Dexterity cap, armor potency rune and resilient rune. It also accepts simple, unconditional `FlatModifier` item bonuses with a numeric value and `type: item`.

Item bonuses of the same type do not stack: only the highest supported item bonus is applied. Unsupported predicates, formulas, missing equipment state, unresolved class-feature documents and missing ability/rank/level inputs yield incomplete or unavailable results rather than fabricated totals. This first calculator version intentionally does not implement circumstance/status bonuses, conditions, temporary effects, spell effects, attacks, DCs, shield AC, or the complete Rule Element engine.
