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
