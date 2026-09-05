# ADR-016: Phase 2.6 hosted trust-boundary hardening

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-014 and ADR-015 for hosted Phase 2.6 eligibility, evidence authority, retention, and exact publication

## Context

The pre-implementation Phase 2.6 attack-review cycle continued after ADR-014/ADR-015 and found four remaining trust-boundary gaps that must be explicit before hosted code is allowed to start.

First, contract v8 is immutable and still requires relevant project current-state synchronization when a memory materially changes present project state. Current `MemoryService.ApplyMutation` does not implement that project-view write. ADR-015 therefore correctly requires a next-contract transition before asynchronous project projections are legal, but hosted admission must also make the compatibility boundary unambiguous rather than attempting to guess which contract-v8 mutations happen not to need project-view synchronization.

Second, finalizer/auditor separation is meaningful only if their **evidence-write capabilities are also separated**. A fresh auditor does not prove an independent gate if a finalizer context can manufacture an object that the repository Durable Object would accept as authoritative audit evidence.

Third, short-retention private object storage must not race the durable operation state. A queue/retry/outage/reconciliation may legitimately outlive a nominal object TTL. Deleting a still-referenced request, candidate, finalization receipt, or audit artifact would turn a healthy recoverable operation into an avoidable integrity failure.

Fourth, the current GitHub REST `Update a reference` API exposes the target SHA and a force/non-force choice, but does not expose an expected-old commit parameter. A non-force fast-forward check is not equivalent to Runethread's required exact `H0 -> C` compare-and-swap: if the canonical ref is moved backward to an ancestor of `H0` between observation and update, a plain fast-forward update to `C` may still succeed. The architecture already requires an invariant-preserving exact-push fallback, but a Worker-only publisher topology does not itself provide a Git executable for that fallback.

These findings do not invalidate the Cloudflare Worker + repository Durable Object + finalizer Container + fresh auditor architecture. They tighten the contract-eligibility, evidence, storage-liveness, and publication boundaries.

## Decision

### 1. Normal hosted mutation admission requires the projection-capable contract

Phase 2.6 v1 normal hosted memory mutation is admitted only for a repository explicitly migrated to the operational contract that implements ADR-015 (target contract v9), or a later contract explicitly declared compatible with the hosted delivery release.

A contract-v8 repository may still be:

- inspected/read;
- trust/compatibility checked;
- reconciled;
- upgraded through the supported migration path;

but the hosted normal mutation path MUST NOT call `ApplyMutation` and then claim a complete v8 write while omitting the v8-required project current-state synchronization.

Runethread does not add a provider-side heuristic such as `affects_project_state=false` to bypass this rule. That would move semantic judgment into the hosted adapter and recreate the dual-write ambiguity ADR-015 was introduced to remove.

ADR-014 statements that project current-state/overview prose is an asynchronous materialized projection therefore apply only after the repository has migrated to the ADR-015 contract semantics. They do not reinterpret contract v8.

### 2. Evidence storage is not an unrestricted shared write capability

The repository Durable Object remains the sole hosted operation-state and publication authority. Private object storage carries sealed request/candidate/receipt/audit bytes, but possession of a generic object-store binding MUST NOT become authority to advance an operation.

Artifact creation is role- and generation-scoped:

- the public admission path may create only the exact sealed request object for the admitted request identity;
- the finalizer role may submit only candidate/finalization artifact classes for its exact repository + operation + phase + execution generation;
- the auditor role may read only the exact request/candidate material needed for its audit and may submit only audit artifact classes for its exact audit generation;
- the finalizer MUST NOT be able to create an authoritative audit receipt;
- the auditor MUST NOT be able to create/replace finalization evidence or publication authorization;
- neither finalizer nor auditor receives canonical Git write authority.

Containers therefore do not receive an unrestricted R2 write surface that lets them choose arbitrary authoritative artifact keys/classes. The hosted implementation MUST mediate writes through a private evidence boundary or single-purpose upload grant that fixes, before upload, at least:

- immutable repository identity;
- hosted operation/attempt identity;
- phase and execution generation;
- artifact class;
- exact object key/reference;
- expected digest/size policy;
- create-if-absent/no-overwrite semantics.

The repository Durable Object accepts a returned artifact only after re-reading its current operation/phase/generation and verifying the immutable object digest/binding. A stale or wrong-role artifact is nonauthoritative even if its bytes exist in storage.

This is a capability boundary, not a second state machine. The private evidence mediation may be implemented inside an existing hosted Worker/service boundary; it does not require another durable coordinator.

### 3. Live evidence is pinned by operation liveness

Privacy retention remains short and explicit, but a wall-clock TTL MUST NOT delete evidence that is still required by a live or unresolved operation.

At minimum:

- queued/active/retrying operations retain their sealed request object;
- `CANDIDATE_READY`/audit phases retain candidate + finalization evidence;
- `AUDITED`/`PUBLISHING` retain all evidence needed to authorize or reconcile exact publication;
- publication ambiguity/reconciliation retains the small exact publication identity and any still-required candidate objects until canonical Git can determine the outcome or policy explicitly transitions to a safe terminal failure;
- referenced evidence is not garbage-collected merely because a nominal age threshold elapsed.

Implementation may satisfy this with reference-aware garbage collection, leases/pins, or a provider lifecycle horizon proven to exceed the maximum supported operation/recovery lifetime. Automatic bucket lifecycle deletion MUST NOT undercut that invariant.

Only unreferenced/orphan evidence, or terminal evidence whose bounded retry/recovery/incident-retention window has expired, is eligible for ordinary deletion. Deterministic integrity-incident evidence may be retained under explicit incident policy.

If required evidence is nevertheless missing/corrupt, fail closed. Do not regenerate new bytes under an old immutable receipt.

### 4. Exact publication uses a real expected-old Git update

The publication invariant remains:

```text
bound canonical ref = H0
audited candidate    = C

require exact current ref == H0
move only that ref to exact C
```

A provider operation that proves only "new value is a fast-forward from whatever the ref currently is" is insufficient.

Phase 2.6 v1 therefore has a concrete safe publication executor:

```text
repository DO
  |
  | durable AUDITED -> PUBLISHING authorization
  v
private GitHub App gateway Worker
  |
  | mint short-lived one-repository Contents-write token
  v
minimal trusted Git publisher executor / Container
  |
  | exact audited Git objects + H0 + C + bound ref
  | no source clone
  | no semantic mutation / repair / audit
  v
Git smart-protocol expected-old update
  |
  v
GitHub bound canonical ref
```

The long-lived GitHub App private key remains only in the private gateway Worker. The publisher executor receives a short-lived installation token narrowed to the one repository and minimum Contents-write permission only after the repository Durable Object has durably won `AUDITED -> PUBLISHING`.

The publisher executor:

1. accepts only the immutable DO-authorized publication identity and exact audited candidate Git objects;
2. verifies the package/bindings needed to prove it is publishing exact `C` based on `H0` to the bound ref;
3. does not run MemoryService, semantic logic, repair, project logic, or repository-controlled hooks/filters;
4. does not perform another source clone;
5. performs an expected-old Git push/update whose server-side ref update rejects a ref value other than `H0`;
6. never constructs a replacement commit `C2`;
7. discards the short-lived write credential and disposable Git state after the attempt.

A normal Git implementation may use explicit expected-old/lease semantics as part of that push. The implementation test must prove races where the remote ref moves forward, sideways, backward, is deleted/recreated, or already equals `C` are all classified according to the ADR-014 publication/reconciliation rules.

A future clone-free GitHub API publication path MAY replace the minimal publisher executor only if authoritative current GitHub documentation plus integration tests prove both:

- exact candidate commit/tree/object identity is preserved; and
- the ref mutation is a true atomic expected-old-`H0` compare-and-swap, not merely a generic fast-forward check.

The currently documented REST ref-update endpoint alone is not accepted as that proof.

### 5. Hosted release/version barriers include the publication/evidence protocol

The hosted release identity and incompatible-change barrier now cover:

- public/internal Worker protocol;
- repository Durable Object schema/state-machine version;
- finalizer/auditor Core runtime and Container image;
- evidence capability/object/receipt protocol;
- publisher executor image/protocol;
- canonical-ref binding semantics;
- supported operational contract identities.

An in-flight operation MUST NOT cross incompatible versions of any of these boundaries.

## Consequences

- Hosted Phase 2.6 has one unambiguous compatibility floor for normal writes: migrate the repository to the ADR-015 contract semantics first.
- Contract v8 remains historically correct and is never silently weakened by provider behavior.
- Fresh audit now has a capability boundary as well as a process/runtime boundary.
- Short privacy retention can no longer strand live work through premature evidence deletion.
- Exact Git publication has a concrete v1 implementation path even if GitHub never adds an API-level expected-old ref CAS.
- The extra publisher executor is intentionally tiny and clone-free. It exists because exact remote CAS is a distinct invariant; it is not another mutation/audit engine.
- The public/internal gateway keeps the long-lived App key and gives the publisher only short-lived repository-scoped authority after publication has been authorized.
- A future proven API-level CAS can delete the publisher executor without changing semantic/Core behavior.
- Because this ADR records material changes found by the architecture-freeze attack, that attack does **not** satisfy the zero-edit gate. Implementation remains blocked until a later full review of the new exact planning head passes with zero required edits.

## Alternatives considered

### Permit contract-v8 hosted writes when the caller says no project view is affected

Rejected. The provider would be trusting semantic classification outside deterministic Core and could claim completion contrary to the pinned v8 contract.

### Give finalizer and auditor the same generic R2 write binding

Rejected. Storage immutability alone does not provide role separation; the finalizer could potentially manufacture the evidence intended to prove an independent audit.

### Use a fixed short R2 lifecycle TTL regardless of operation state

Rejected. Provider/GitHub outages and reconciliation can legitimately outlive the TTL and would convert recoverable work into avoidable integrity failures.

### Use GitHub REST `Update a reference` with `force=false` as exact CAS

Rejected for the v1 correctness proof. Fast-forward safety is weaker than requiring the old ref to equal exact `H0`.

### Give the finalizer Container Contents-write only after audit and let it push

Rejected as the default boundary. Reusing the semantic finalizer as publisher broadens the component that receives canonical write authority and weakens the separation between candidate construction and promotion. A tiny dedicated publisher executor is easier to constrain and test.

### Reintroduce GitHub Actions only to perform the exact push

Rejected. It restores runner/workflow latency and a redundant normal orchestration hop solely to obtain a Git executable.

## Verification

Implementation is compliant only if tests/evidence demonstrate at least:

1. a contract-v8 repository is refused for normal hosted mutation before candidate construction and is directed to the supported migration path;
2. an explicitly migrated ADR-015 contract repository can enter the hosted mutation lifecycle;
3. a finalizer context cannot create or replace an authoritative audit receipt/key;
4. an auditor context cannot create/replace finalization evidence or obtain publication authority;
5. wrong-role, wrong-generation, wrong-repository, wrong-attempt, wrong-key, digest-mismatched, and overwrite artifact submissions cannot advance DO state;
6. exact concurrent duplicate artifact submissions are idempotent while byte-different collisions fail closed;
7. queued/active/audited/publishing/reconciling evidence survives nominal retention boundaries while it remains referenced;
8. orphan and safely terminal evidence is deleted under the explicit privacy policy;
9. missing/corrupt referenced evidence fails closed without silent regeneration under an old receipt;
10. the long-lived GitHub App private key is absent from public API, finalizer, auditor, evidence objects, and publisher executor;
11. the publisher token is installation-scoped to the exact repository and minimum required Contents-write permission and is minted only after durable `PUBLISHING` authorization;
12. the publisher imports exact audited candidate objects without a second source clone and cannot create `C2`;
13. exact publication succeeds only for `ref == H0` and exact audited `C`;
14. forward/sideways/backward/deleted/recreated ref races cannot be overwritten as if `H0` still held;
15. ambiguous response recovery still resolves `ref == C` as committed, `ref == H0` as eligible for the same exact retry, and every other ref as reconciliation;
16. ordinary GitHub REST fast-forward ref update is not treated as the exact-CAS proof unless a later supported API exposes and tests a real expected-old condition;
17. finalizer/auditor/evidence/publisher protocol versions are pinned into one compatible hosted release generation and deployment barriers prevent mixing incompatible generations;
18. removal of the publisher executor in a future optimization is allowed only after an independently tested API path proves the same exact-object and expected-old invariants.