# Runethread v0.6.0 cutover checklist

Status: **Accepted Phase 1 execution plan**
Date: 2026-09-02

This checklist operationalizes ADR-009. It is intentionally conservative: rename product identity aggressively now, but preserve canonical user data and historical Git truth.

## 1. Verified pre-cutover state

Implementation repository:

```text
Karageorgiou/GitMemo
module github.com/Karageorgiou/GitMemo
current latest memory repo pin: v0.5.0
```

Known active private memory repository:

```text
Karageorgiou/GitMemo-memory
visibility: private
repository format: 1
schema version: 1
contract version: 6
GitMemo version: v0.5.0
Index v2 record count: 24
managed metadata: .gitmemo/
project-memory slug: gitmemo
```

Before migration starts, record the exact implementation and private-memory Git revisions and the complete canonical memory UUID set.

## 2. Target native identity

```text
organization:          runethread
implementation repo:   runethread/core
Go module:             github.com/runethread/core
CLI:                   runethread
command source:        cmd/runethread/
bootstrap manifest:    runethread-bootstrap.json
managed metadata:      .runethread/
private memory repo:   Karageorgiou/runethread-memory
project-memory slug:   runethread
planned first release: v0.6.0
primary domain:        runethread.dev
```

Historical commits, v0.1.0-v0.5.0 tags/releases, and accurate historical references remain unchanged.

## 3. Compatibility-version proposal

The identity cutover changes repository-level managed layout and trust metadata but does not require changing the atomic memory JSON schema.

Proposed Runethread v0.6.0 compatibility dimensions:

```text
repository_format: 2   # .runethread native managed layout
schema_version:    1   # atomic memory shape unchanged unless implementation proves otherwise
contract_version:  7   # Runethread-native operational contract
lock_version:      2   # Runethread-native authority/version fields
```

These values are proposals to be validated by implementation tests. If the migration requires an atomic memory shape change, schema version must be reconsidered explicitly rather than changed incidentally.

## 4. Implementation repository identity changes

### Repository/GitHub

- transfer/rename `Karageorgiou/GitMemo` to `runethread/core` while preserving repository history;
- verify issues, pull requests, branches, tags, releases, and GitHub redirects after transfer;
- update repository description/homepage to Runethread / `runethread.dev`;
- preserve old tags and release artifacts unchanged.

### Go/module/CLI

- change `go.mod` module path to `github.com/runethread/core`;
- update all internal import paths from the old module path;
- rename `cmd/gitmemo/` to `cmd/runethread/`;
- make `runethread` the only native CLI command after cutover unless a temporary recovery shim is proven necessary;
- update version/build metadata and tests;
- run `go fmt`, `go vet`, and full tests after path changes.

### Product files and public docs

Review and migrate current/native references in at least:

- `README.md`;
- `AI_SETUP.md`;
- `MEMORY_PROTOCOL.md`;
- `gitmemo-bootstrap.json` -> `runethread-bootstrap.json`;
- `docs/EXTENDING_GITMEMO.md` -> Runethread-native filename/content;
- getting-started, compatibility, repository-role, trust, validation, taxonomy, source, index, and command docs;
- schemas/templates when they contain product identity;
- `.github/workflows/validate.yml` and release automation;
- release-contract/bootstrap tests;
- source/tests containing CLI name, managed directory name, repository URL, or product-specific constants.

Do not perform a blind global replacement over historical fixtures, changelog/release history, or text that is intentionally describing old GitMemo releases.

## 5. Native managed metadata changes

For newly initialized Runethread repositories:

```text
.gitmemo/     -> .runethread/
```

Native config should use Runethread terminology. For example, the old `gitmemo_version` field should not remain the native current field merely for compatibility.

The trust lock must pin the immutable Runethread Core release/contract and identify `runethread/core` as the new authority.

The initializer, upgrader, validator, trust verifier, generated workflow, and bootstrap contract must all agree on the same native managed paths and version dimensions.

## 6. Private memory migration

Migrate `Karageorgiou/GitMemo-memory` in a dedicated reviewable migration commit/branch.

Required steps:

1. capture pre-migration HEAD;
2. export/snapshot canonical UUID set and canonical-memory digest information;
3. validate the v0.5.0 repository before touching it;
4. migrate managed control files to the Runethread v0.6.0 contract;
5. rename `.gitmemo/` to `.runethread/` through deterministic migration logic;
6. rename project-specific memory directory/slug `gitmemo` -> `runethread` where it is current product identity;
7. review memory titles/content containing `GitMemo` individually:
   - preserve references that describe historical GitMemo facts;
   - update references whose meaning is "the current project/product";
   - never change UUIDs merely because title/content changes;
8. regenerate the entire derived index from canonical memory;
9. validate using the Runethread v0.6.0 validator;
10. compare canonical UUID set before/after;
11. compare record count and inspect semantic diffs;
12. commit only after all hard checks pass;
13. rename repository to `Karageorgiou/runethread-memory` only after content migration is verified.

## 7. Data invariants that must survive exactly

Migration must preserve, unless an explicitly reviewed semantic memory edit says otherwise:

- every memory UUID;
- every Markdown/JSON pairing;
- memory type;
- lifecycle state;
- relationships and targets;
- provenance/source metadata;
- temporal fields;
- privacy/sensitivity metadata;
- open-loop state;
- unrelated user-owned files;
- Git history before the migration commit.

Generated indexes are expected to change and should be regenerated rather than hand-edited.

## 8. Validation gates

The cutover cannot be reported successful until all applicable checks pass.

### Core

```text
go test ./...
go vet ./...
Runethread validator self/fixture tests
bootstrap/manifest tests
release-contract tests
```

### Private memory

```text
pre-migration v0.5.0 validation PASS
migration completes without partial-success state
post-migration Runethread validation PASS
index freshness/check PASS
UUID set before == UUID set after
record count expected == record count after
no .gitmemo native managed directory remains
.runethread trust/config metadata internally consistent
```

### Git/GitHub

```text
old GitMemo tags still reachable
old releases still reachable
implementation Git history preserved
new repository URL is runethread/core
new module/CLI install path works
private repo remains private
```

## 9. Rollback boundary

Until the new Runethread release and private-memory migration have both passed all verification:

- record and retain the pre-migration commit SHAs;
- do not delete old tags/releases;
- do not force-push rewritten history;
- do not use user canonical memory as scratch/rollback storage;
- prefer a dedicated branch or reversible migration commit for the private repository;
- if hard validation fails, restore managed state to the recorded pre-migration revision and report failure.

## 10. Deliberately deferred work

The identity cutover does **not** yet implement:

- `MemoryService` / two-phase runtime mutation API;
- MCP server;
- Orchestrator;
- Codex worker;
- general agent worker;
- cloud hosting;
- Kubernetes.

Those are built only after the repository and memory format are consistently Runethread-native.

## 11. Cutover definition of done

Phase 1 is complete only when:

1. `runethread/core` is the authoritative implementation repository;
2. new users see only Runethread-native setup, CLI, module, and managed metadata names;
3. Runethread v0.6.0 is reproducibly buildable/testable from the preserved implementation history;
4. the known private repository is migrated and still contains the same canonical UUID identities;
5. all hard validation/index/trust checks pass;
6. historical GitMemo releases remain truthful and reachable;
7. there is no known active current-state dependency on the GitMemo brand/name that was left accidentally rather than intentionally.
