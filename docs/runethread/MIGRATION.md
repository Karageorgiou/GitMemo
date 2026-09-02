# GitMemo to Runethread migration strategy

Status: **Proposed**

Runethread is being named and structured while GitMemo is still a pre-v1 project with one known active user/deployment and no known external compatibility dependency. This changes the correct migration strategy.

Rather than carry GitMemo naming and compatibility aliases indefinitely, the project will perform one controlled, verified cutover from the current GitMemo state to a fully Runethread-native identity.

The migration is intentionally aggressive about **names** and conservative about **data**.

## 1. Migration principles

1. Preserve canonical user data, UUIDs, relationships, provenance, lifecycle state, and Git history.
2. Rename current product identifiers consistently now rather than carrying avoidable legacy names into the architecture.
3. Historical Git commits, tags, and release artifacts remain immutable history and may continue to say GitMemo.
4. Generated indexes are disposable and should be rebuilt rather than mechanically renamed.
5. The migration must be deterministic, testable, reviewable, and reversible until verification succeeds.
6. No successful migration may be reported until the resulting repository passes Runethread validation.
7. GitMemo v0.5.0 is the supported source state for the known active private memory repository.
8. No indefinite `gitmemo` CLI alias, `.gitmemo/` native format, or dual-brand control plane is required after the cutover unless a concrete external user is discovered before release.

## 2. Current source identity

Implementation repository:

```text
github.com/Karageorgiou/GitMemo
```

Go module:

```text
github.com/Karageorgiou/GitMemo
```

CLI:

```text
gitmemo
```

Known active memory repository is currently pinned to:

```text
GitMemo release:       v0.5.0
repository format:     1
schema version:         1
contract version:       6
trust lock version:     1
```

Current native control metadata uses:

```text
.gitmemo/
```

Historical GitMemo releases and tags are not rewritten.

## 3. Target Runethread identity

Public organization:

```text
https://github.com/runethread
```

Public repositories:

```text
runethread/core
runethread/orchestrator
runethread/memory-template
runethread/.github
```

Primary domain:

```text
https://runethread.dev
```

Go module:

```text
github.com/runethread/core
```

CLI:

```text
runethread
```

Native repository metadata:

```text
.runethread/
```

New private user repository convention:

```text
<user>/runethread-memory
```

The first Runethread release should continue the existing semantic version lineage rather than reuse historical tags. With GitMemo currently at v0.5.0, the expected first Runethread release is:

```text
v0.6.0
```

## 4. Why a clean cutover is preferable now

A long compatibility bridge is appropriate after public adoption. It is not automatically a virtue before adoption.

Carrying both identities would create permanent complexity in:

- CLI names;
- module/import paths;
- generated repository paths;
- lock/config field names;
- documentation;
- test matrices;
- MCP tool descriptions;
- future Orchestrator integration;
- support/debug output.

Because the project is still pre-v1 and has no known external users, paying that complexity cost now would make the system less coherent without protecting a demonstrated dependency.

The clean-cutover window should therefore be used before Runethread gains additional users.

## 5. Public implementation migration

The authoritative GitMemo repository should become Runethread Core by preserving its Git history rather than copying source into an unrelated new repository.

Target:

```text
Karageorgiou/GitMemo
        |
        | transfer + rename
        v
runethread/core
```

Git history, issues, pull requests, tags, and release history should remain attached where the hosting platform supports it.

After the repository move, perform a reviewed identity-migration change that updates current source identifiers, including at minimum:

- `go.mod` module path;
- internal imports;
- `cmd/gitmemo` -> `cmd/runethread`;
- executable/help text;
- bootstrap/setup manifests;
- current documentation and links;
- generated repository defaults;
- `.gitmemo/` -> `.runethread/` for the new native format;
- config and lock field names where they encode product identity;
- trust-source repository identity;
- workflow/package/release names;
- templates and user-facing command wording.

Historical commits and old immutable release artifacts are not rewritten to hide the previous name.

## 6. Go module and CLI cutover

The module path changes once:

```text
github.com/Karageorgiou/GitMemo
->
github.com/runethread/core
```

The primary executable changes once:

```text
gitmemo
->
runethread
```

Because there are no known external Go consumers or users to protect, the current plan does not require a permanent compatibility module or executable alias.

Historical GitMemo binaries remain downloadable from historical releases if forensic or rollback access is needed.

If an external dependency is discovered before the cutover release, this decision must be revisited explicitly rather than silently adding compatibility behavior.

## 7. Native repository-format cutover

Runethread's native format should use Runethread terminology from its first release.

Expected managed metadata transition:

```text
.gitmemo/
->
.runethread/
```

Expected identity metadata transition includes conceptually:

```text
gitmemo_version
->
runethread_version

source_repository: Karageorgiou/GitMemo
->
source_repository: runethread/core
```

The exact schema/field transformation must be encoded as a deterministic migration rather than a blind text replacement.

The Runethread repository-format version should change if required to make the identity transition explicit and machine-detectable.

## 8. Known private memory migration

The active private repository should be migrated as a controlled transaction, not recreated from scratch.

Repository identity:

```text
Karageorgiou/GitMemo-memory
->
Karageorgiou/runethread-memory
```

Canonical memory identities must be preserved.

Migration flow:

```text
record pre-migration commit/revision
        |
        v
verify source is expected GitMemo v0.5.0 state
        |
        v
create migration branch / recovery point
        |
        v
migrate managed control-plane identity
        |
        v
migrate project slug/metadata: gitmemo -> runethread
        |
        v
review project-memory text that intentionally names the product
        |
        v
regenerate all derived indexes
        |
        v
validate complete Runethread repository
        |
        v
compare protected invariants before/after
        |
        +-- failure -> restore/reject
        |
        v
commit verified migration
```

The migration must not delete and re-create memories simply to change branding.

## 9. Memory-content handling

Brand migration has two different classes of text and they must not be conflated.

### Managed infrastructure text

Current operational documentation, configuration, generated control-plane files, command names, project slugs, and product identity should be rewritten to Runethread.

### Canonical memory text

Memory UUIDs and semantic history remain canonical.

Because the known memory set is small and primarily concerns the newly created project itself, project memories may be explicitly reviewed and rewritten where `GitMemo` merely names the same project that is now Runethread.

This is a semantic migration, not a global string replacement.

If a memory refers specifically to a historical event or historical release under the GitMemo name, preserving `GitMemo` may be the truthful representation.

A migration record should make the rename itself explicit so provenance remains understandable.

## 10. Derived indexes

Do not attempt to preserve generated index files byte-for-byte.

After canonical data/control metadata migration:

```text
delete/stage old generated index
-> rebuild with Runethread Core
-> verify deterministic index freshness
```

The record/UUID set represented by the rebuilt index must match the migrated canonical repository.

## 11. Trust-anchor transition

GitMemo v0.5.0 remains an immutable historical source state.

The migration tool must positively recognize the expected old trust/config state before transforming it.

Runethread v0.6.0 becomes the first native Runethread trust anchor after the migration.

The migration must not pretend that an old GitMemo v0.5.0 lock was originally issued by Runethread.

Conceptually:

```text
verified GitMemo v0.5.0 source
        |
        | explicit migration
        v
verified Runethread v0.6.0 target
```

This preserves historical truth without carrying dual-brand native state forever.

## 12. Verification contract

Before migration, capture at minimum:

- source commit SHA;
- memory record count;
- complete UUID set;
- relationships/lifecycle state;
- provenance fields;
- project membership;
- repository format/schema/contract/trust versions.

After migration, verify at minimum:

- the expected number of memory records remains present;
- UUID set is identical;
- every Markdown/JSON pair is valid;
- relationship graph remains valid;
- lifecycle and provenance are preserved except explicitly reviewed brand changes;
- no unintended canonical user files disappeared;
- all generated indexes are rebuilt and fresh;
- native metadata contains no unintended GitMemo operational identity;
- full repository validation passes.

A migration that changes the count or UUID set unexpectedly is a hard failure.

## 13. Rollback

Before changing the known private memory repository, retain a durable pre-migration Git revision/branch/tag.

Until target validation succeeds:

- do not delete the source repository/history;
- do not force-push rewritten history;
- do not remove the old release artifacts;
- do not claim the migration is complete.

Rollback should normally mean returning the branch/repository to the pre-migration commit, not reconstructing data manually.

## 14. Compatibility scope after cutover

Runethread's normal supported state begins with the native Runethread format.

A narrow GitMemo v0.5.0 -> Runethread migration path may remain in tooling or documented recovery code because it is cheap insurance for the known source state.

The project does **not** commit to indefinite support for every experimental GitMemo pre-v1 format unless a real user/dependency is discovered before the cutover.

This explicitly supersedes the broader experimental GitMemo compatibility promise for the purpose of the rebrand.

Once Runethread has external users, compatibility expectations become materially stricter and breaking identity migrations must no longer be treated this casually.

## 15. Definition of migration success

The cutover is complete only when:

1. the authoritative implementation is `runethread/core`;
2. current source/build/docs use Runethread naming consistently;
3. the primary CLI is `runethread`;
4. the module path is `github.com/runethread/core`;
5. newly initialized memory repositories are Runethread-native and use `.runethread/`;
6. the known private memory repository has been migrated and validated without UUID/data loss;
7. project-specific memory/index metadata uses the Runethread project identity;
8. GitMemo names remain only where historically truthful or intentionally retained for migration/history;
9. historical GitMemo commits/tags/releases remain intact;
10. future Core/MCP/Orchestrator work can proceed without carrying avoidable GitMemo aliases.
