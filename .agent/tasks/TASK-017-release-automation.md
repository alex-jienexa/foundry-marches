---
id: TASK-017
title: Automate GitHub Releases and release notes
status: backlog
priority: high
assignee: unassigned
created_at: '2026-11-25'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-003
requirement_ids:
- FM-REQ-002
files_owned:
- .github/workflows/
- docs/release/
- CHANGELOG.md
labels:
- release
- github-actions
- conventional-commits
---

# Automate GitHub Releases and release notes

## Objective

Подготовить воспроизводимую публикацию релизов на основе обязательной политики Conventional Commits.

## Scope

- GitHub Actions workflow for tag validation and release creation.
- SemVer validation and prerelease handling.
- Release notes grouped by Conventional Commit type/scope.
- Breaking-change detection from `!` and `BREAKING CHANGE:`.
- Changelog preview/review before publication.
- First target release: `v0.2.0-beta`.

## Excluded

Automatic Production deployment. CD is separately gated by TASK-018 and explicit owner approval.

## Acceptance criteria

- [ ] Invalid tags/versions are rejected.
- [ ] `v0.2.0-beta` is treated as prerelease, not stable.
- [ ] Release notes include user-facing feat/fix/perf and breaking changes; internal docs/chore can be omitted or grouped.
- [ ] Workflow runs on a test tag/dry-run without publishing an unintended release.
- [ ] Owner reviews generated changelog before first public release.
- [ ] No Production deploy occurs from this workflow.

## Verification

GitHub Actions dry-run and test tag; inspect generated release notes and permissions.

## Progress log

- 2026-11-25 — Created; Conventional Commits policy already exists.
