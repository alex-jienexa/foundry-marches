# Commit message convention

FoundryMarches uses **Conventional Commits 1.0.0**. This is a mandatory project-wide rule for every agent and human-authored commit. Commit messages must be structured so release notes and semantic versioning can be generated reliably in a future release workflow.

## Required format

```text
<type>(<scope>): <imperative description>

[optional body]

[optional footer(s)]
```

- `type` is required and lowercase.
- `scope` is recommended; use a concise area such as `api`, `discord`, `sync`, `database`, `foundry`, `frontend`, `docs`, `ci`, or `deps`. Omit it only when no useful scope exists.
- Separate type and scope from the description with a colon and one space.
- Write the description in English, concise, imperative, and without a trailing period.
- Use the body to explain motivation and relevant implementation details, not to repeat the subject.
- Separate body and footers with a blank line.
- Do not combine unrelated changes into one commit. Prefer one logical change per commit.

## Types

| Type | Meaning | Release-note treatment |
|---|---|---|
| `feat` | Adds user-visible or API functionality | Features |
| `fix` | Fixes incorrect behavior | Bug fixes |
| `perf` | Improves performance | Performance |
| `refactor` | Internal restructuring without behavior change | Normally omitted |
| `docs` | Documentation-only change | Normally omitted |
| `test` | Adds or changes tests | Normally omitted |
| `build` | Build system or dependency/build tooling | Normally omitted |
| `ci` | CI workflow or automation configuration | Normally omitted |
| `chore` | Maintenance that does not fit another type | Normally omitted |
| `style` | Formatting or style-only changes without behavior change | Normally omitted |
| `revert` | Reverts a previous commit | Reverts |

Choose the type based on the change's actual impact, not the files touched. A user-visible bug fix remains `fix` even if it changes many files. A new user-facing capability is `feat`.

## Breaking changes

Mark a breaking change with `!` immediately after the type/scope and explain it in a `BREAKING CHANGE:` footer.

```text
feat(api)!: replace character registration endpoint

BREAKING CHANGE: clients must use POST /api/v2/characters.
```

The footer is required even when `!` is present. Use breaking-change markers only when existing consumers or stored data require migration or otherwise become incompatible.

## Examples

```text
feat(discord): add NPC message impersonation
fix(sync): handle actors without a background
perf(database): reduce character lookup queries
refactor(api): isolate relay response decoding
docs(api): document integration endpoints
test(sync): cover partial actor fetch failures
build(deps): update pgx to v5
ci(actions): cache Go build dependencies
chore(repo): remove obsolete development fixture
revert: feat(discord): add NPC message impersonation
```

## Agent and contributor behavior

1. Before committing, classify the change and write a Conventional Commit message. Do not use generic messages such as `update`, `fixes`, `changes`, or `WIP`.
2. If a task produces multiple independent changes, split them into logically coherent commits with their own accurate types and scopes.
3. Do not mislabel internal maintenance as `feat` just to make it appear in release notes.
4. Never invent a feature or breaking change that the diff does not support. The message must describe the actual committed diff.
5. If the correct type or breaking status is unclear, inspect the behavior and requirements; ask the task owner when impact cannot be established.
6. When asked to create a commit, use this convention by default. When asked to prepare a commit but not actually commit, provide the proposed Conventional Commit subject.
7. Existing historical commits are not to be rewritten merely to normalize their messages.

## Release automation compatibility

Future release automation may generate notes from these commit types and scopes and derive SemVer increments:
- `fix` and `perf` normally indicate PATCH.
- `feat` normally indicates MINOR.
- Any valid `BREAKING CHANGE:` footer indicates MAJOR.
- Other types do not independently trigger a version increment.

This document establishes the commit convention only. It does **not** enable release automation, change CI/CD workflows, publish a release, or deploy Production. Release-note grouping and version calculation must be implemented and tested as a separate, explicitly approved CI/CD task.
