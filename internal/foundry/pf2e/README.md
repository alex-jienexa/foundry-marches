# Foundry PF2e Actor decoder

This package is a read-only boundary adapter. It unwraps Relay entity responses,
validates complete Actor snapshots, retains raw Actor/Item JSON, and projects
known fields into the versioned application Character DTO.

## Contract

- Foundry/Relay JSON remains the source of truth; unknown fields are retained.
- Actor Items are represented once in Character.Items.
- Item category is a UI/application hint, not a closed PF2e type registry.
- Item.system is retained in Details without type-specific assumptions.
- Missing and null values remain nil; no default quantity, equipment state,
  ability score, skill rank, or derived statistic is fabricated.
- Class/background Rule Elements and Compendium references are not evaluated.
- This package performs no writes to Foundry and no PF2e rules calculations.

## Tests

Run from the repository root:

```sh
go test ./internal/foundry/pf2e
```

The tests decode both committed Relay fixtures under docs/foundry.
