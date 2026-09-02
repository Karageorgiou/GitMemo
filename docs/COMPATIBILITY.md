# Runethread compatibility and support policy

Runethread exists to preserve durable user memory. Repository evolution therefore prioritizes recoverability, explicit migration, and preservation of canonical data over implementation convenience.

## Native compatibility baseline

Native Runethread repositories begin with v0.6.0 / repository format 2.

Future official Runethread releases SHOULD retain tested migrations from supported earlier native Runethread repository states whenever technically possible. A release MUST NOT intentionally strand an officially supported repository format without one of:

1. a deterministic supported migration path;
2. a documented technical or security reason automated migration cannot safely preserve the repository; or
3. an explicit loss-avoiding recovery procedure.

“Create a new repository and start over” is not an acceptable normal migration strategy for durable user memory.

## One predecessor bridge in v0.6.0

Runethread v0.6.0 has one finite pre-native migration path: the exact trusted GitMemo v0.5.0 state with repository format 1, schema 1, contract 6, and trust lock 1.

That bridge is intentionally narrow. The upgrader verifies the legacy config, lock metadata, aggregate/per-file control-plane hashes, and managed validation workflow before writing native state. Unknown, mixed, customized, or tampered `.gitmemo` state is refused rather than guessed.

This is a migration compatibility rule, not a permanent GitMemo compatibility layer. Native Runethread uses `.runethread/`, `runethread` commands, `runethread_version`, and `runethread/core` as its source authority.

## Version dimensions

Runethread tracks separate compatibility dimensions:

- **release version** — implementation/distribution version;
- **repository format version** — repository-level layout/protocol generation;
- **schema version** — atomic memory data shape;
- **contract version** — operational semantics for LLMs and tooling;
- **index format version** — generated discovery layout;
- **trust lock version** — envelope used to pin and verify the control plane;
- **bootstrap protocol version** — machine-readable onboarding manifest contract.

A release change does not imply every dimension changes.

## Pinning

A memory repository remains pinned to its installed official Runethread release until an explicit supported upgrade occurs. Mutable public `main` is not an implicit upgrade channel.

Native release and contract digests are recorded in `.runethread/lock.json`.

## Migration design

Migrations SHOULD be ordered transformations between known compatibility states, not permissive repair routines.

A migration must preserve user-owned canonical data unless a representation change is explicitly required and tested. At minimum, regression testing protects:

- memory UUIDs;
- Markdown/JSON memory meaning and bytes when their schema is unchanged;
- project/user-owned files;
- lifecycle and relationships;
- provenance and temporal information;
- sensitivity metadata;
- unrelated files outside managed paths.

Generated indexes may be deterministically rebuilt rather than preserved byte-for-byte.

## Release regression fixtures

For each supported historical source state, the repository SHOULD keep representative fixtures and exercise:

```text
supported old fixture -> current upgrader -> current validation
```

The v0.6.0 suite includes an exact historical v0.5.0 control-plane fixture and tests successful migration, source-tamper refusal, mixed-state refusal, custom-workflow refusal, idempotent native operation, and rollback after post-write validation failure.

## Rollback and failure

An upgrade must validate its result before reporting success. When an upgrade fails after modifying Runethread-managed/generated files, tooling should restore the pre-upgrade managed state when safely possible.

User-owned canonical memory/project data must not be used as rollback scratch space.

## Deprecation

If supported compatibility ever must be retired, the decision should be explicit, versioned, documented, and based on a concrete technical or security constraint rather than maintenance convenience alone.

## Free and local operation

Compatibility must not depend on a paid Runethread server. Official release binaries/source, repository data, and documented migrations should remain sufficient to interpret and upgrade a repository locally when the relevant files/history are available.
