# ADR-011: Separate runtime releases from contract releases

Status: **Accepted**
Date: 2026-09-04

## Context

Runethread memory repositories vendor an operational control plane and pin its authority in `.runethread/config.json` and `.runethread/lock.json`.

Through v0.7.0, one release identity serves two roles:

1. the version of the running Runethread executable; and
2. the immutable official release that owns the repository's vendored operational contract.

That coupling is harmless while every executable release also changes or deliberately repins the repository contract, but it creates unnecessary repository churn for a future runtime-only release.

A superseded design attempted to reinterpret the existing v0.7 `runethread_version` pin as a contract-release pin without changing the contract. That is invalid. Published contract v7 normatively describes `runethread_version` and trust validation in terms of the running/pinned Runethread release. Changing implementation semantics alone would make the runtime disagree with the immutable released contract.

The separation therefore requires one explicit contract transition.

## Decision

Runethread tracks two distinct release identities:

- `ReleaseVersion` is the running implementation/distribution release.
- `ContractReleaseVersion` is the immutable official release that owns the operational contract embedded in that runtime.

For the first separated release:

```text
runtime release             v0.8.0
contract release            v0.8.0
repository format           2
memory schema               1
contract version            8
index format                2
trust-lock version          2
bootstrap protocol          1
bootstrap verifier          v0.6.0
```

Contract version 8 changes the semantics of the existing native config/lock field `runethread_version`: for contract-v8-and-later repositories it identifies the **contract release**, not necessarily the running runtime release. The JSON field is retained so repository format and trust-lock envelope do not change merely to rename an already adequate field.

Published contract-v7 repositories retain their original historical semantics. They become contract-v8 repositories only through an explicit supported migration. No v0.7 artifact is retroactively reinterpreted.

Repository-producing and repository-validating code MUST use `ContractReleaseVersion` for the native repository pin, expected trust lock, config, historical-source classification, and contract-authority diagnostics. Runtime-facing version reporting MUST use `ReleaseVersion`.

Machine-facing status MUST expose the two values separately. Existing `release_version` retains its runtime-release meaning; `contract_release_version` is additive.

A future runtime-only release MAY advance `ReleaseVersion` while leaving `ContractReleaseVersion`, contract bytes, and all repository compatibility dimensions unchanged. Such a runtime MUST accept an unchanged repository pinned to the contract release and MUST NOT require a repository repin or canonical commit solely because the executable version advanced.

Conversely, any genuine change to operational-contract semantics or bytes requires an explicit contract release/version transition and supported migration for retained historical source states.

### Stable bootstrap

Bootstrap protocol 1 remains valid. The stable bootstrap command `runethread trust version` resolves the repository's `runethread_version`, which under contract v8 is the contract release. Installing that exact release remains a conservative valid way to verify the repository.

A bootstrap or setup client may run a newer compatible runtime, but it must not infer that the repository pin equals that runtime's release version.

### Historical source recognition

Native migrations are recognized through explicit immutable source anchors rather than a rolling `previousNativeReleaseVersion` special case.

At minimum, v0.8 retains exact trusted native v0.6.0 and v0.7.0 sources plus the existing exact GitMemo v0.5.0 predecessor bridge. Historical source verification is based on known compatibility dimensions and trusted control-plane digests before any write occurs.

Historical fixtures MUST represent released historical state. A future/current generator MUST NOT be used to manufacture an older contract after bytes or semantics diverge. Reuse of current embedded bytes in a historical fixture is allowed only when their SHA-256 exactly matches the historical lock; otherwise the historical bytes must be frozen explicitly.

## Consequences

- Runtime-only releases no longer force otherwise meaningless memory-repository commits.
- Repository authority remains tied to an immutable official release and exact contract digests.
- `runethread_version` has a contract-version-dependent historical meaning: release/runtime pin in contract v7, contract-release pin in contract v8 and later. Migration makes that boundary explicit.
- The v0.8 migration changes managed contract/config/lock state but does not require repository format, memory schema, index format, or lock-envelope changes.
- Operators and future MCP adapters can distinguish runtime capability/version from repository contract authority.
- The upgrader must model supported source anchors explicitly and preserve rollback/tamper refusal.
- Documentation must say “contract release” where repository authority is intended and “runtime release” where executable distribution is intended.

## Alternatives considered

### Reinterpret v0.7 `runethread_version` without a contract bump

Rejected. It would change the meaning of an immutable published contract after release and was the flaw in superseded PR #13.

### Rename the JSON field and bump repository format / lock version

Rejected for this transition. The existing field can represent the contract release unambiguously once contract v8 defines its semantics. Changing the envelope would add migration surface without improving integrity.

### Keep one release identity permanently

Rejected. It makes every runtime-only release require repository metadata churn even when the contract is byte-for-byte unchanged.

### Let the latest runtime silently validate older contracts without an explicit contract identity

Rejected. Compatibility would become implicit and difficult to audit. The contract release must remain explicit and independently reportable.

## Verification

Implementation is compliant only if tests and release verification demonstrate all of the following:

1. exact trusted v0.7 contract state migrates to contract v8 before any contract-v8 semantics are used;
2. exact trusted v0.6 native and GitMemo v0.5 predecessor paths remain supported unless separately deprecated;
3. tampered, mixed, unsupported, and unknown-newer source states are refused before writes;
4. canonical memory/project data is preserved through migration and rollback remains effective;
5. new config/lock state pins `ContractReleaseVersion`, not `ReleaseVersion`;
6. trust validation compares repository pins/digests to the embedded contract release;
7. status reports runtime and contract release separately;
8. a synthetic future runtime with `ReleaseVersion != ContractReleaseVersion` accepts the unchanged matching contract repository without repinning it;
9. a repository incorrectly pinned to that newer runtime is rejected when the runtime still embeds the older contract release;
10. contract-impact CI rejects future contract changes that omit the corresponding contract version/release and migration evidence.
