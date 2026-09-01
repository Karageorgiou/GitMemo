# GitMemo compatibility and support policy

GitMemo exists to preserve durable memory. Repository evolution must therefore prioritize recoverability and explicit migration over convenience for the current implementation.

## Compatibility promise

For every official GitMemo memory-repository release beginning with `v0.1.0`, the project SHOULD retain a tested migration path to the current release whenever technically possible.

A new release MUST NOT intentionally strand an older officially supported repository format without one of:

1. a deterministic supported migration path;
2. a documented technical reason that automated migration cannot safely preserve the repository;
3. an explicit, loss-avoiding manual recovery procedure.

“Create a new repository and start over” is not an acceptable normal migration strategy for durable user memory.

## Version dimensions

GitMemo tracks separate compatibility dimensions:

- **GitMemo release version** — implementation/distribution version;
- **repository format version** — layout and repository-level protocol generation;
- **schema version** — atomic memory data shape;
- **contract version** — operational semantics for LLMs and tooling;
- **trust lock version** — trust-lock envelope used to pin and verify the control plane.

A release version change does not imply that every compatibility dimension changes.

## Pinning

A memory repository remains pinned to its installed GitMemo release until an explicit supported upgrade occurs.

Public `main` is not an implicit upgrade channel.

The pinned release and contract digests are recorded in `.gitmemo/lock.json` for trust-aware repositories.

## Migration design

Migrations SHOULD be implemented as ordered transformations between known compatibility states rather than a growing collection of arbitrary one-off jumps.

A migration must preserve user-owned canonical data unless the migration explicitly documents why a representation change is necessary.

At minimum, upgrade testing must protect:

- memory UUIDs;
- Markdown/JSON memory meaning;
- project/user-owned files;
- lifecycle and relationships;
- provenance and temporal information;
- private/sensitivity metadata;
- unrelated user files that are outside GitMemo-managed paths.

Generated indexes may be rebuilt rather than preserved byte-for-byte.

## Release regression fixtures

Before GitMemo is declared stable for long-term use, the repository should retain representative fixtures for every official memory-repository release.

Every future release gate should exercise:

```text
old official fixture -> current upgrader -> current validation
```

for each supported historical fixture.

Fixtures should be small but must contain enough canonical data to detect accidental loss, identity changes, relationship damage, project-file rewriting, and managed/user-boundary violations.

## Rollback and failure

An upgrade must validate its result before reporting success. When an upgrade fails after modifying GitMemo-managed files, tooling should restore the pre-upgrade managed state when safely possible.

User-owned canonical memory/project data must not be used as a rollback scratch area.

## Deprecation

If a compatibility behavior ever must be retired, the decision should be explicit, versioned, documented, and based on a concrete technical or security constraint. Deprecation must not happen merely to reduce maintenance effort without considering durable user data.

## Free and local operation

Compatibility must not depend on a paid GitMemo server. Official release binaries/source, repository data, and documented migrations should remain sufficient to interpret and upgrade a repository locally when the relevant Git history/files are available.
