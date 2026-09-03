# ADR-011: Separate runtime releases from contract releases

Status: **Accepted**
Date: 2026-09-04

## Context

Runethread memory repositories pin an immutable official release so their vendored operational contract can be verified without trusting mutable `main`. The initial native implementation used one version value, `ReleaseVersion`, both for the executable product release and for the repository's pinned control-plane authority.

That coupling is unnecessarily strong. Phase 2 demonstrated the problem: v0.7.0 added the transport-independent MemoryService without changing repository format 2, memory schema 1, contract 7, Index v2, trust-lock 2, or any vendored control-plane bytes, yet existing memory repositories still had to repin from v0.6.0 to v0.7.0 merely because the binary version advanced.

Future executable-only work such as the local MCP adapter must not rewrite trusted repository metadata when the memory contract has not changed.

## Decision

Runethread maintains two explicit release dimensions:

- `ReleaseVersion` identifies the running product/binary release.
- `ContractReleaseVersion` identifies the immutable official Runethread release that owns the embedded memory-repository control plane.

The existing `.runethread/config.json` and `.runethread/lock.json` field `runethread_version` remains backward-compatible and represents the pinned **contract release**. It is not a requirement that the running executable have that same release number.

A running Runethread release may operate on a repository pinned to an older contract release only when all of the following hold:

1. the running binary's `ContractReleaseVersion` equals the repository pin;
2. repository format, memory schema, operational contract, Index format, and trust-lock compatibility are supported;
3. the aggregate embedded contract digest matches the lock;
4. every locked control-plane file digest matches both the embedded contract and the vendored repository copy.

The running `ReleaseVersion` is never substituted for those checks.

If any vendored control-plane file or compatibility dimension changes, `ContractReleaseVersion` must advance and repositories require an explicit supported upgrade before using that new contract. A runtime-only release keeps the previous `ContractReleaseVersion` and must not repin otherwise-valid repositories.

The stable validation bootstrap continues to resolve `runethread_version` from the repository lock and install that immutable contract release for independent repository validation. That remains intentional even when a newer compatible runtime exists.

## Consequences

- MCP, Orchestrator-facing adapters, CLI UX, performance work, packaging, and other executable-only releases can ship without meaningless memory-repository commits.
- Trust remains release-anchored and digest-verified; no mutable `main`, remote lookup, compatibility range, or second trust root is introduced.
- The runtime version and repository contract pin may legitimately differ and diagnostics must describe them distinctly.
- New repository initialization pins `ContractReleaseVersion`, not `ReleaseVersion`.
- Repository upgrade results describe movement between contract releases rather than merely reporting the running binary version.
- A future real contract change still requires an explicit migration and new contract release anchor.

## Alternatives considered

### Keep exact runtime-release pinning

Rejected because executable-only releases create repository churn without increasing trust or compatibility guarantees.

### Replace the release pin with only `contract_version`

Rejected because the immutable release plus aggregate/per-file digests provides a concrete independently verifiable authority for the exact contract bytes. The integer contract version alone is insufficient.

### Add semantic-version ranges to the lock

Rejected because ranges weaken the simple immutable trust anchor and introduce compatibility policy into repository metadata. Runtime compatibility is better expressed by the running binary's embedded contract anchor and existing version/digest dimensions.

### Rename `runethread_version` immediately

Rejected for now because the existing field can carry the more precise contract-release meaning without a repository-format or lock-format migration. The semantic distinction is recorded here and in current documentation.

## Verification

Implementation and tests must demonstrate that:

- trust locks and starter configs are generated from `ContractReleaseVersion`;
- native compatibility accepts a repository pinned to contract release v0.7.0 when a simulated runtime release is v0.8.0 and the compatibility dimensions are unchanged;
- a repository pin equal only to the simulated runtime release is not accepted when the runtime's contract release remains v0.7.0;
- `upgrade` reports the target contract release, not merely `ReleaseVersion`;
- no contract file, repository format, schema, contract, Index format, or trust-lock version changes as part of this decoupling;
- a future runtime release can deliberately advance `ReleaseVersion` while leaving `ContractReleaseVersion` unchanged and still validate a repository pinned to that contract release.
