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

Missing values stay absent. The decoder does not reconstruct ability scores from boosts or execute PF2e rules. The calculator may use class progression and supported Rule Elements to derive effective proficiency ranks while preserving the baseline and source breakdown.

`CharacterBuildDto.ClassProficiencies` is deliberately separate. It records the class item's progression baseline (`perception`, `savingThrows`, `defenses`, `attacks`, `spellcasting`, `trainedSkills`) and is not by itself the character's final effective rank. The calculator starts from this baseline and applies supported upgrades. The `additional` class skill count is never used to invent selected skills.

Equipment calculation inputs are exposed only where their source data is structured and recognized:
- armor `acBonus`, `dexCap` and `strength`;
- armor/other item's rune `potency`, `resilient` and `property` values;
- the existing equipped state.

The raw Actor snapshot remains authoritative for all unrecognized fields.

`CharacterDerivedStatsDto` defines the future calculator output contract: each statistic has an optional value, a status (`complete`, `incomplete`, `unavailable`), a breakdown and warnings. A missing input must never be replaced with zero. The calculation version is explicit so future algorithm changes can be distinguished.

The decoder only prepares DTOs; calculations live in the sibling `calculator` package. The calculator intentionally excludes temporary effects, situational modifiers and unsupported runtime-prepared behavior.


## Class progression and Rule Elements

`CharacterBuildDto.ClassProficiencies` now captures the class baseline from the class Item: perception, all saving throws, defense categories, attack categories, spellcasting rank, trained skills and additional skill count.

The class Item's `system.items` map is decoded as `ClassFeatureReferenceDto` entries (UUID, name, level and image). These are references to compendium documents, not embedded Actor Items. The decoder does not pretend that a reference contains its Rule Elements: the referenced document must be resolved separately before its rules can be interpreted.

Rule Elements present on the class Item and owned Actor Items are projected into `RuleElementDto`. The original rule JSON is retained in `Raw`, including unsupported fields. This is declarative data only: no predicates, formulas, grants, selectors, or other Rule Element behavior are executed by the decoder.

PF2e `ActiveEffectLike` can upgrade proficiency ranks, including through a `ChoiceSet`-selected path. The calculator supports a deliberately narrow, documented subset (static numeric `upgrade` values and simple level brackets). Unsupported expressions, predicates, conditional rules and unresolved compendium references must produce warnings/incomplete results, never guessed ranks. This is necessary because Rule Elements can depend on selections, predicates, actor level and data-preparation phases.

Fighter fixture expectations: perception 2; Fortitude 2, Reflex 2, Will 1; defenses Heavy/Light/Medium/Unarmored 1; attacks Advanced 1, Martial 2, Simple 2, Unarmed 2; spellcasting 0. Its class progression table contains 16 feature references, two at level 1. The `fighterSkill` ChoiceSet and its ActiveEffectLike are preserved as source rules.


## Proficiency calculator

The sibling `calculator` package resolves effective proficiency ranks from:
1. class progression baseline (Perception, saves, defenses, attacks and spellcasting);
2. explicit Actor ranks and Actor skill ranks, when present;
3. static numeric `ActiveEffectLike` rules with `mode: upgrade`, including simple
   level brackets and `ChoiceSet` selections preserved on the source item.

The rank resolver does not evaluate predicates, formulas, grants or conditional
rank rules. The derived-stat calculator additionally supports only unconditional
numeric `FlatModifier` rules with `type: item`. Class feature references whose
level has been reached but whose document is not present among Actor Items can
be loaded by an implementation of `calculator.CompendiumItemResolver` and
`CalculateDerivedStatsWithResolver`; otherwise they are reported as warnings
and the rank result is incomplete. The resolver must return the exact Item
JSON for the requested UUID; it is read-only.

Run tests from `src/backend` with `go test ./...` and `go vet ./...`.


The calculator's `CalculateDerivedStats` returns both effective proficiency ranks (including attack, defense and tradition-specific spellcasting ranks) and totals for Perception, AC, Fortitude, Reflex, Will and skills present in the normalized rank set. It uses PF2e proficiency bonuses (untrained 0; trained `level+2`; each subsequent rank adds 2), the stored ability modifier (or a modifier derived from an explicitly stored ability score), armor category/AC bonus/Dexterity cap, armor potency rune and resilient rune. It also accepts simple, unconditional `FlatModifier` item bonuses with a numeric value and `type: item`.

Item bonuses of the same type do not stack: only the highest supported item bonus is applied. Unsupported predicates, formulas, missing equipment state, unresolved class-feature documents and missing ability/rank/level inputs yield incomplete or unavailable results rather than fabricated totals. This first calculator version intentionally does not implement circumstance/status bonuses, conditions, temporary effects, spell effects, attacks, DCs, shield AC, or the complete Rule Element engine.


## PostgreSQL persistence adapter

The `internal/characters` package maps the decoded Actor into a database write
model and persists it transactionally using the v0.3 schema:

- `characters` receives the normalized identity, build labels, progression,
  HP values, attributes payload and skills payload. Fields owned by the app
  (owner, status, approval) are not changed.
- `foundry_actors` receives the permanent Actor UUID, complete JSONB snapshot,
  SHA-256 snapshot hash and sync timestamp. An Actor already linked to another
  character is rejected; the link is never silently reassigned.
- `inventory_items` is reconciled by Foundry Item ID. Existing row IDs and
  catalog links are preserved, stale Actor-owned rows are removed, and no
  `inventory_events` are fabricated by a snapshot refresh.
- The full Actor JSON exists only in `foundry_actors.snapshot`. Each inventory
  row stores only the normalized `CharacterItemDto` projection in `state`.

The repository sets transaction-local `app.writer = 'foundry_sync'` before
writing. It accepts an already-established world/character/Actor association;
it does not implement player ownership confirmation or the linking API. It does
not call Foundry or mutate an Actor.

Missing DTO values remain absent in the mapping. The current SQL schema has
non-null defaults for `characters.level`, `characters.xp` and
`inventory_items.quantity`; for a missing item quantity the exact null remains
in `inventory_items.state`, while the indexed relational column follows the
existing schema default on insert and retains its prior value on update. This
schema behavior is explicit and should be revisited if exact nullable quantity
semantics become a database requirement.

The persistence package uses `database/sql` and accepts a caller-configured
database handle; driver selection and connection-pool lifecycle belong to the
application composition root. Mapper tests and recording-driver transaction tests
run with `go test ./...`. These tests verify mapping, transaction ordering and
rollback behavior, but they do not execute SQL against PostgreSQL. A live
PostgreSQL integration test still needs to be wired into the project test
environment before this adapter is considered fully verified.
