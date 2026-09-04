# Runethread compatibility and support policy

Runethread exists to preserve durable user memory. Repository evolution therefore prioritizes recoverability, explicit migration, and preservation of canonical data over implementation convenience.

## Native compatibility baseline

Native Runethread repositories begin with v0.6.0 / repository format 2.

Future official Runethread releases SHOULD retain tested migrations from supported earlier native Runethread repository states whenever technically possible. A release MUST NOT intentionally strand an officially supported repository format without one of:

1. a deterministic supported migration path;
2. a documented technical or security reason automated migration cannot safely preserve the repository; or
3. an explicit loss-avoiding recovery procedure.

“Create a new repository and start over” is not an acceptable normal migration strategy for durable user memory.

## Historical predecessor bridge

Runethread retains one finite pre-native migration path: the exact trusted GitMemo v0.5.0 state with repository format 1, schema 1, contract 6, and trust lock 1.

That bridge is intentionally narrow. The upgrader verifies the legacy config, lock metadata, aggregate/per-file control-plane hashes, managed validation workflow, and applicable filesystem-object safety before writing native state. Unknown, mixed, customized, newer-unknown, or tampered `.gitmemo` state is refused rather than guessed.

This is a migration compatibility rule, not a permanent GitMemo compatibility layer. Native Runethread uses `.runethread/`, `runethread` commands, `runethread_version`, and `runethread/core` as its source authority.

## Version dimensions

Runethread tracks separate compatibility dimensions:

- **runtime release version** — implementation/distribution version of the executable;
- **contract release version** — immutable official release that owns the embedded operational repository contract;
- **repository format version** — repository-level layout/protocol generation;
- **schema version** — atomic memory data shape;
- **contract version** — operational semantics for LLMs and tooling;
- **index format version** — generated discovery layout;
- **trust lock version** — envelope used to pin and verify the control plane;
- **bootstrap protocol version** — machine-readable onboarding manifest contract.

A runtime release change does not imply a contract change, and a contract change does not imply every other compatibility dimension changes.

Runethread v0.8.0 introduces the explicit runtime/contract-release split through contract version 8. Under contract v8, `.runethread/config.json` and `.runethread/lock.json` retain the existing `runethread_version` field, but that field is the **contract release anchor**. Historical contract-v7 repositories keep their published v7 meaning until they are explicitly migrated.

The v0.8.0 transition keeps repository format 2, schema 1, index format 2, trust-lock version 2, bootstrap protocol 1, and bootstrap verifier v0.6.0 unchanged while advancing contract version from 7 to 8.

## Pinning

A memory repository is pinned to its installed official **contract release** until an explicit supported contract upgrade occurs. Mutable public `main` is not an implicit upgrade channel.

The runtime executing against that repository may be newer than the pinned contract release when the runtime explicitly embeds and supports that exact contract and all compatibility dimensions/digests match. In that case, a runtime-only release MUST NOT require repository churn, lock rewriting, or a memory-repository commit merely to record the newer executable version.

Conversely, a repository MUST NOT be repinned to a newer runtime release when that runtime still embeds an older contract release. A genuine future contract change requires an explicit migration.

The contract-release identity and contract digests are recorded in `.runethread/lock.json`; runtime release identity is reported separately by the executable/service status surface.

## Historical native source anchors

Contract migrations are accepted only from explicitly supported exact source states. For the contract-v8 transition, Runethread preserves exact native v0.6.0 and v0.7.0 / contract-v7 source anchors rather than synthesizing historical state from the current generator.

Historical fixture material may reuse a current embedded contract byte only when its SHA-256 exactly matches the historical trusted lock for that path. Otherwise the historical byte must be frozen explicitly. Source tampering, mixed managed metadata, unsupported versions, or newer unknown states are refused before migration writes.

## Filesystem-object compatibility boundary

Contract v8 makes authoritative repository filesystem objects fail-closed. Repository-owned canonical/control-plane/index-source paths must use real directories and regular files as specified by the trust and repository-validation contracts. Symbolic links and unsupported special filesystem objects are rejected rather than followed for authoritative inputs.

Migration source verification establishes those conditions before writes. Rollback snapshots operate on regular managed files and must not dereference a symbolic link and later restore copied target bytes in its place.

Freshly initialized v8 memory repositories also include a managed root `.gitattributes` support file for byte-stable LF text checkouts. `.gitattributes` is repository/bootstrap support rather than a `ContractPaths()` member or trust-lock digest. A v7 -> v8 migration may create it when absent, accepts the exact managed bytes when already present, and refuses to overwrite a conflicting custom file.

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

The suite protects successful historical migration, source-tamper refusal, mixed-state refusal, custom-workflow/support-file ownership, idempotent current-native operation, and rollback after post-write validation failure.

For runtime/contract-release separation, regression coverage must also prove a deliberately newer runtime can accept the unchanged pinned contract release without repository churn, while rejecting a repository incorrectly pinned to the newer runtime when no new contract exists.

## Rollback and failure

An upgrade must validate its result before reporting success. When an upgrade fails after modifying Runethread-managed/generated files, tooling should restore the pre-upgrade managed state when safely possible.

User-owned canonical memory/project data must not be used as rollback scratch space.

## Deprecation

If supported compatibility ever must be retired, the decision should be explicit, versioned, documented, and based on a concrete technical or security constraint rather than maintenance convenience alone.

## Free and local operation

Compatibility must not depend on a paid Runethread server. Official release binaries/source, repository data, and documented migrations should remain sufficient to interpret and upgrade a repository locally when the relevant files/history are available.
