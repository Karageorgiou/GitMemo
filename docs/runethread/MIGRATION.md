# GitMemo to Runethread migration strategy

Status: **Proposed**

The Runethread rebrand must not strand existing GitMemo users, memory repositories, release history, or Go installation paths. This document defines a staged compatibility-first migration.

## 1. Migration principles

1. Rebranding is not permission to break durable-memory compatibility.
2. Existing GitMemo memory repositories remain readable and upgradeable.
3. Existing official GitMemo releases remain immutable historical trust anchors.
4. Old repository identifiers such as `.gitmemo/` are compatibility data, not cosmetic strings to mass-replace.
5. Canonical user memory must not be rewritten merely to change branding.
6. Generated indexes may be regenerated when required.
7. Every destructive or identity-changing step must have an explicit rollback/recovery path.
8. The Go module-path transition must be treated as an API/distribution migration, not a text rename.

## 2. Current identity that must be preserved

Current implementation module:

```text
github.com/Karageorgiou/GitMemo
```

Current CLI identity:

```text
gitmemo
```

Current repository metadata/control paths include `.gitmemo/` and release-pinned GitMemo control-plane files.

Existing memory repositories may refer to GitMemo in:

- config/version metadata;
- trust locks;
- vendored contracts;
- validation workflows;
- setup documentation;
- release pins;
- repository names;
- Git history.

These references are historically meaningful and must not be blindly rewritten.

## 3. Target identity

Public organization:

```text
https://github.com/runethread
```

Target repositories:

```text
runethread/core
runethread/orchestrator
runethread/memory-template
runethread/.github
```

Target primary domain:

```text
https://runethread.dev
```

Target CLI:

```text
runethread
```

Target new-development module path:

```text
github.com/runethread/core
```

The exact timing of the Go module move is release-controlled and must not happen before compatibility behavior is defined and tested.

## 4. Migration stages

### Stage A — architecture only

- keep `Karageorgiou/GitMemo` canonical and unchanged in behavior;
- document Runethread architecture and migration decisions;
- do not rename `.gitmemo/`, schemas, lock formats, CLI, or module paths;
- create no compatibility burden from premature implementation changes.

### Stage B — compatibility release under GitMemo

Publish a final/bridge GitMemo release that:

- clearly announces Runethread as the successor project;
- retains normal GitMemo operation;
- recognizes/records enough metadata for a deterministic transition;
- contains tested migration logic needed by the first Runethread Core release;
- points users to the new organization/domain;
- preserves the old trust contract as an immutable release.

The bridge release should not require users to rename their private memory repository.

### Stage C — establish Runethread Core

Create `runethread/core` from the authoritative implementation history using a migration method that preserves Git history where practical.

The first Runethread Core release must:

- support existing officially supported GitMemo repository formats;
- understand legacy `.gitmemo/` control metadata as required;
- validate old release pins using their historical GitMemo authority;
- provide an explicit upgrade path to a Runethread-native repository contract if/when a native format is introduced;
- never interpret mutable Runethread `main` as authority for old GitMemo repositories.

### Stage D — CLI transition

Introduce `runethread` as the primary CLI.

For a defined compatibility window, options include:

1. retain a `gitmemo` compatibility executable/alias that delegates to compatible Core behavior; or
2. keep the final GitMemo binary independently available and document the `runethread` replacement path.

The chosen option must be decided by ADR after implementation constraints are known.

### Stage E — optional repository-format migration

Do not require legacy repositories to rename themselves merely because the product is renamed.

A legacy repository may remain valid with `.gitmemo/` metadata indefinitely if that is the safest compatibility choice.

If a new `.runethread/` repository format is introduced, migration must be explicit and deterministic:

```text
legacy GitMemo repository
  -> inspect pinned compatibility dimensions
  -> backup/transaction boundary
  -> migrate managed control metadata
  -> preserve user canonical data and UUIDs
  -> regenerate derived indexes
  -> validate current contract
  -> commit only verified result
```

The project should prefer semantic compatibility over aesthetic consistency.

## 5. Go module-path migration

The current module path is part of the public Go identity:

```text
github.com/Karageorgiou/GitMemo
```

The new path will be:

```text
github.com/runethread/core
```

Important consequences:

- old `go install` commands must continue working for historical GitMemo releases;
- old tags cannot be moved or rewritten;
- import paths in external consumers do not magically redirect;
- any public Go API that becomes supported must account for module identity explicitly;
- the CLI transition can occur independently from repository-format migration.

Do not rewrite the existing repository's `go.mod` until the organization/repository transition step is deliberately executed.

## 6. Trust-anchor continuity

Old GitMemo releases remain authoritative for repositories pinned to those releases.

Runethread must be able to distinguish:

```text
legacy authority: official immutable GitMemo release
new authority:    official immutable Runethread Core release
```

A Runethread binary validating a legacy repository must not silently substitute a new Runethread contract for the repository's historical GitMemo contract.

Where cross-brand validation is needed, mappings between release authorities must be deterministic, explicit, and tested.

## 7. Repository names are not data migrations

Existing repositories such as:

```text
Karageorgiou/GitMemo-memory
<user>/GitMemo-memory
```

need not be renamed.

New installations should eventually prefer:

```text
<user>/runethread-memory
```

Repository URL changes are navigation/distribution changes. They must not be conflated with canonical memory-format changes.

## 8. Template migration

The public generated template must remain a release artifact, not a second hand-maintained contract.

Transition:

```text
final GitMemo release
  -> legacy GitMemo-template remains available for legacy docs/releases

Runethread Core release
  -> deterministic Runethread init output
  -> runethread/memory-template
```

Users must never receive a template whose vendored control plane and trust lock disagree about which product/release owns the contract.

## 9. Documentation and redirects

During transition:

- old GitHub repository URLs should use GitHub redirects where transfers/renames support them;
- documentation should state old and new identities explicitly;
- release notes should distinguish branding changes from repository-format changes;
- `runethread.dev` becomes the durable public documentation/home identity;
- old GitMemo documentation for historical releases remains accessible.

## 10. Migration test matrix

Before the first Runethread Core stable release, automated tests should cover at minimum:

```text
GitMemo v0.1 fixture -> current Runethread upgrader -> validation
GitMemo v0.2 fixture -> current Runethread upgrader -> validation
...
every officially supported GitMemo repository release
```

Tests must protect:

- memory UUIDs;
- Markdown/JSON meaning;
- provenance;
- lifecycle and relationships;
- project/user-owned files;
- trust-lock correctness;
- private/sensitivity metadata;
- managed vs unmanaged path boundaries;
- rollback on failure.

## 11. Rollback strategy

Until the transition is proven:

- do not delete the old GitMemo repository;
- do not rewrite historical releases or tags;
- do not force existing private memory repositories into a new layout;
- make migration commits explicit and reviewable;
- preserve pre-migration Git state so a failed managed-state upgrade can be reverted safely.

## 12. Definition of migration success

The rebrand is successful only when all of these are true:

1. a new user can install Runethread without knowing the GitMemo name;
2. an existing GitMemo user can continue using their repository without data loss;
3. every supported legacy repository can be deterministically validated/upgraded;
4. old immutable GitMemo releases remain meaningful trust anchors;
5. Runethread's new module/repository identity is clean for future development;
6. no historical canonical memory is rewritten solely for cosmetic branding.
