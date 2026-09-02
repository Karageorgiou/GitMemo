# ADR-009: Controlled GitMemo-to-Runethread cutover

Status: **Accepted**
Date: 2026-09-02

## Context

GitMemo was created immediately before the Runethread architecture was defined. At the time of this decision there is one known user, one private active memory repository, and 24 indexed memory records. The active repository is pinned to GitMemo v0.5.0 with repository format 1, schema version 1, and contract version 6.

Carrying permanent `GitMemo` aliases, `.gitmemo/` paths, old CLI naming, and dual metadata into a new platform would create long-lived complexity for compatibility that currently has almost no external constituency. The user explicitly prefers consistent Runethread naming from the beginning.

At the same time, migration must preserve canonical memory data, UUIDs, provenance, Git history, and recoverability.

## Decision

Runethread will perform one controlled hard identity cutover before new Core service/MCP/Orchestrator APIs are built.

Target current/native identity after the cutover:

```text
organization:       runethread
implementation:     runethread/core
module:             github.com/runethread/core
CLI:                runethread
managed metadata:   .runethread/
private repo name:  <user>/runethread-memory
project memory slug: runethread
first release:      v0.6.0 (planned)
```

The authoritative implementation repository must be transferred/renamed in a way that preserves Git commit history, tags, issues, pull requests, and release provenance where GitHub supports preservation/redirects.

The private memory repository will be migrated deterministically from GitMemo v0.5.0 to Runethread-native managed metadata. The migration must:

1. snapshot/record the pre-migration revision;
2. preserve all canonical memory UUIDs and semantic content unless a branding reference is intentionally reviewed and changed;
3. preserve provenance, lifecycle, relationships, privacy/sensitivity metadata, and user-owned project data;
4. migrate managed control metadata from `.gitmemo/` to `.runethread/`;
5. update trust/config metadata to the new immutable Runethread release authority;
6. rename product-specific current project paths/slugs where semantically appropriate;
7. regenerate derived indexes from canonical memory;
8. run full validation before reporting success;
9. commit the migration as an explicit reviewable boundary;
10. retain a recovery path to the recorded pre-migration Git revision until the cutover is verified.

Historical Git commits, old tags/releases, and text that accurately describes historical GitMemo behavior must not be rewritten merely for aesthetics.

The project will retain a narrow tested migration/recovery path from the known GitMemo v0.5.0 state into Runethread v0.6.0. It will not intentionally carry permanent dual-brand compatibility machinery unless evidence of real external users/dependencies appears before cutover.

## Consequences

- New implementation work starts with consistent Runethread names instead of immediately creating rename debt.
- The cutover becomes an early high-risk migration that must be tested and reviewed carefully.
- Existing GitMemo v0.1-v0.5 Git history remains visible as product history rather than being rewritten.
- The earlier broad promise to support every historical GitMemo repository indefinitely is narrowed for the rebrand decision because there is currently one known user and a controlled repository state.
- If previously unknown external users are discovered before cutover, this ADR must be revisited before destructive compatibility assumptions are implemented.

## Alternatives considered

### Long compatibility bridge with `.gitmemo/` and `gitmemo` aliases indefinitely
Rejected because the product is one day old and permanent compatibility burden would outweigh its current value.

### Rewrite history so GitMemo never existed
Rejected because it destroys provenance and makes releases/commits misleading.

### Start a blank `runethread/core` repository and copy current files
Rejected because it loses valuable GitHub/Git lineage and audit history.

### Keep `.gitmemo/` forever while renaming only the brand
Rejected because the user prefers a clean native identity and the current migration surface is still small enough to change safely.

## Verification

Before cutover is declared successful:

1. record count and canonical UUID set before/after are compared;
2. all migrated memories validate under the Runethread contract;
3. provenance/lifecycle/relationships/privacy metadata are preserved;
4. derived indexes are rebuilt and match canonical sources;
5. `.runethread/` is the sole native managed metadata directory in the migrated repository;
6. current CLI/module/docs use Runethread naming consistently;
7. historical GitMemo commits/tags remain reachable and unchanged;
8. rollback to the recorded pre-migration Git revision is demonstrated or mechanically available;
9. repository-level and release CI pass before the migration is considered complete.
