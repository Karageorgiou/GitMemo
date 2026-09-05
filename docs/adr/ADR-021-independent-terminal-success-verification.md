# ADR-021: Independent verification of terminal-success mutation results

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-014, ADR-019, and ADR-020 where they allow finalizer results to become client-visible terminal outcomes without publication

## Context

The next Phase 2.6 architecture-freeze review started from exact synchronized planning head `34421411f4501f762c9c104e45d2cc92a9c3c5cb`, after ADR-020 had closed the candidate-content trust gap by requiring a fresh auditor to independently prove that exact candidate `C` is the Core-derived semantic result of the exact sealed request against exact `H0`.

That fix is correct for candidate-producing results, but the same trust boundary still has a bypass.

ADR-014 allows the finalizer to return successful terminal results which never enter candidate audit/publication:

- `NO_OP` is Core-validated and produces no candidate/audit/publication;
- `ALREADY_COMMITTED` is accepted from canonical committed-idempotency evidence (or previously reconciled hosted terminal state) and likewise produces no new candidate.

The finalizer's immutable receipt binds the result class, request/evidence digests, Core mutation metadata/fingerprint, and runtime identities, but immutability only proves which result the finalizer reported. It does not independently prove that the result was true.

A faulty or compromised finalizer which ADR-020 is explicitly designed not to trust can therefore avoid the new candidate-conformance gate by returning a false terminal-success result instead of presenting a candidate. For example:

1. an `update` request which should produce a candidate can be falsely reported as `NO_OP`; or
2. a request whose idempotency key is not present in accepted canonical Git can be falsely reported as `ALREADY_COMMITTED`.

Either result can cause the hosted operation to be exposed as successfully complete and release the repository lane without any fresh independent semantic/canonical proof. This is an integrity failure, not merely an availability failure: the caller may believe the requested mutation is already satisfied when it is not.

Failure/stale results are different. A malicious finalizer can always deny service, but a false `NEEDS_REPREPARE`, request-local failure, or retryable provider failure does not assert that durable semantic success already occurred. The prepublication trust boundary therefore needs independent verification for **success-like terminal semantic claims**, not a second expensive audit for every unsuccessful execution result.

## Decision

### 1. No finalizer-only terminal semantic success

A finalizer-authored receipt is not sufficient authority for any result which lets the hosted operation be reported as successfully complete without publishing a newly audited candidate.

At minimum, Phase 2.6 v1 applies this rule to:

- `NO_OP`;
- `ALREADY_COMMITTED` / Core `already_applied` equivalent.

Before either result becomes client-visible durable terminal state or releases the active lane, a fresh reduced-privilege verification context must independently prove the exact result from immutable inputs and canonical evidence.

The same auditor/verifier trust boundary introduced by ADR-020 may perform this work. No new coordinator or semantic implementation is introduced.

### 2. `NO_OP` requires fresh Core result verification

For a claimed `NO_OP`, the verifier reads the exact immutable sealed request selected by the repository Durable Object and evaluates it with the same pinned Core semantics against exact expected canonical base `H0`.

The claim is accepted only if fresh Core execution/verification independently returns the `NO_OP` result for that exact request and base after the ordinary request validation, committed-idempotency lookup, exact revision check, and hard canonical repository validation required by Core.

The terminal verification receipt binds at least:

- repository/canonical-ref identity;
- hosted operation/attempt + verification generation;
- exact sealed-request reference/digest and normalized Core request fingerprint;
- exact `H0`/expected revision;
- pinned Core/runtime/contract/delivery identities;
- independently derived `NO_OP` result class;
- any Core result metadata declared invariant for that release.

A finalizer cannot convert a candidate-producing request into an authoritative `NO_OP` merely by writing a no-op result class into its finalization receipt.

### 3. `ALREADY_COMMITTED` requires fresh canonical committed-idempotency proof

For a claimed `ALREADY_COMMITTED`, the verifier does not trust the finalizer's commit identity or mutation trailers.

It obtains the directly observed bound canonical ref/current accepted canonical state required by the current lane/reconciliation rules and independently runs the pinned Core/repository committed-idempotency lookup for the exact request.

The claim is accepted only when fresh canonical evidence proves all applicable conditions, including:

- the exact Core idempotency identity exists in canonical reachable Git history under the accepted/reconciled history rules;
- the committed mutation's stored request fingerprint equals the independently normalized fingerprint of the exact sealed request;
- conflicting/duplicate/malformed mutation metadata fails closed under ADR-017;
- the independently identified committed operation/result metadata agrees with the terminal claim;
- the canonical history containing that committed evidence is itself eligible under ADR-017/ADR-018/ADR-019 reconciliation and protected-history rules.

ADR-003 ordering remains unchanged: committed-idempotency lookup occurs before ordinary stale-revision classification. The verifier therefore must not incorrectly require current canonical `HEAD == request.expected_revision` before proving an already-committed retry.

A previously established hosted terminal result may be reused without another fresh verification only when the repository Durable Object can verify an immutable prior independent terminal-success receipt for the exact same request/operation and that receipt remains valid under current accepted-history/recovery state.

### 4. Terminal-success evidence is immutable, generation-bound, and rollback-recoverable

Independent terminal-success verification produces an immutable role-separated receipt/evidence object. The finalizer cannot create or replace it.

The repository Durable Object accepts a terminal-success transition only after verifying the receipt against its current repository binding, active operation, request digest, verification generation, release identities, and canonical-history/recovery state.

Because `NO_OP`/`ALREADY_COMMITTED` may be returned to a caller as durable terminal outcomes and release the lane, the corresponding independent terminal-success receipt/reference is recovery-relevant evidence under ADR-019. Before the outcome is exposed as durably terminal, the rollback-independent safety journal records the minimal receipt reference/digest/result identity needed to reconstruct that fact after Durable Object PITR/recreation.

The safety journal still contains no plaintext memory body or credential and does not become a second state machine. On rollback, recovery may either restore the exact independently verified terminal outcome from intact receipt evidence or conservatively re-verify when the protocol explicitly proves that doing so preserves the same result semantics. It must not silently resurrect a previously returned terminal-success operation into a new semantic mutation merely because the DO database was restored.

### 5. Candidate success remains governed by ADR-020

This ADR does not weaken or duplicate ADR-020.

- candidate-producing success -> ADR-020 request-to-candidate conformance + existing trust/index/scope audit;
- no-candidate `NO_OP` success -> fresh Core terminal-result verification under this ADR;
- `ALREADY_COMMITTED` success -> fresh canonical committed-idempotency verification under this ADR;
- actual publication/commit outcome -> ADR-016 through ADR-019 publication, fencing, protected-history, and rollback rules.

All successful semantic completion paths therefore cross an independent trust boundary before the caller is told that the operation is satisfied.

### 6. Unsuccessful execution results do not require equivalent semantic replay

Phase 2.6 does not require the fresh auditor to independently replay every finalizer error merely to defend against denial of service by a compromised finalizer.

`NEEDS_REPREPARE`, request-local validation failure, provider failure, and retryable execution failure continue to follow their existing classification/retry/reconciliation rules. They must still be generation-bound and fail closed, and canonical trust/repository/binding failures still cause lane-level suspension/reconciliation where required.

This distinction is intentional: the independent boundary prevents **false success**, while availability against a fully compromised finalizer is not claimed.

### 7. Release barriers include terminal-result verification protocol

The hosted release/version barrier additionally covers:

- terminal-success verification request/receipt schema;
- Core/result-class compatibility;
- canonical committed-idempotency evidence representation;
- rollback-independent terminal-success receipt references.

An in-flight operation cannot cross incompatible versions of this protocol.

## Consequences

- A compromised/faulty finalizer cannot bypass ADR-020 by falsely declaring `NO_OP` or `ALREADY_COMMITTED`.
- Every client-visible semantic success path is independently proven: either exact candidate conformance, fresh no-op verification, fresh committed-idempotency proof, or exact publication/reconciliation evidence.
- Provider code still does not implement mutation semantics. Fresh verification reuses pinned Core/repository logic.
- Failure paths do not acquire unnecessary duplicate semantic work solely to defend against denial-of-service behavior.
- Terminal-success receipts become a small additional rollback-recovery evidence class, preserving client-visible completion across destructive DO restore without creating another live state authority.
- This review therefore fails the zero-edit architecture-freeze gate. Implementation remains blocked until a later full review of the new exact synchronized planning head itself requires zero architecture/planning edits.

## Alternatives considered

### Trust immutable finalization receipt for `NO_OP` / `ALREADY_COMMITTED`

Rejected. Immutability proves what the finalizer claimed, not whether the claim is correct.

### Only audit candidate-producing results

Rejected. A compromised finalizer can bypass the candidate audit by returning a false success-like terminal result instead.

### Parse `operation == noop` in provider code and trust that for `NO_OP`

Rejected as the general integrity boundary. It does not solve false `ALREADY_COMMITTED`, and the architecture should use one Core-owned verification rule for terminal semantic truth rather than accumulate provider-side semantic shortcuts.

### Independently replay every finalizer error

Rejected for v1. It adds cost without preventing false canonical mutation success; a compromised finalizer can still deny service. Only success-like terminal semantic claims require this additional independent proof.

### Let current canonical ref alone prove `ALREADY_COMMITTED`

Rejected. Ref equality/ancestry alone does not prove the exact idempotency key/request fingerprint/mutation metadata. The fresh verifier must use Core/repository committed-idempotency evidence.

## Verification

Implementation satisfies this ADR only if tests/evidence demonstrate at minimum:

1. legitimate explicit `NO_OP` from exact sealed request + exact `H0` passes fresh independent Core verification;
2. a candidate-producing request falsely reported by the finalizer as `NO_OP` is rejected and cannot become terminal success;
3. a false `NO_OP` receipt with correct request digest/fingerprint but wrong Core result class is rejected;
4. a legitimate exact retry whose operation is already committed in accepted canonical reachable history passes fresh `ALREADY_COMMITTED` verification;
5. a finalizer falsely claims `ALREADY_COMMITTED` when the idempotency key is absent: terminal success is rejected;
6. a finalizer points to a commit with the same idempotency key but different request fingerprint: conflict/failure is preserved and terminal success is rejected;
7. duplicate/conflicting/malformed mutation metadata cannot be used to manufacture `ALREADY_COMMITTED`;
8. committed-idempotency verification remains before stale classification, including a current descendant whose `HEAD` differs from the request's original expected revision;
9. non-descendant/protected-history/recovery states cannot be treated as ordinary `ALREADY_COMMITTED` proof merely because a commit object exists somewhere outside accepted canonical history;
10. finalizer cannot create or replace the independent terminal-success receipt;
11. the DO rejects terminal-success evidence for wrong repository/ref/request/attempt/generation/release identity;
12. a terminal-success result is not exposed/released before its rollback-independent minimal receipt reference/result identity is established;
13. restoring the DO to before a previously returned verified `NO_OP` does not silently turn that accepted operation into a new semantic candidate;
14. restoring the DO to before a previously returned verified `ALREADY_COMMITTED` reconstructs or safely re-verifies the same committed result under accepted-history rules;
15. candidate-producing results still require ADR-020 conformance and cannot use this terminal path to skip candidate audit;
16. unsuccessful finalizer results cannot be accidentally interpreted as independently verified semantic success; and
17. local/offline MemoryService behavior remains unchanged by the hosted verification protocol.
