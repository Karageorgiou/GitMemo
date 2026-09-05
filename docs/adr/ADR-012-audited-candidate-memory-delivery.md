# ADR-012: Audited candidate promotion for external memory delivery

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20

## Context

Runethread already has a deterministic two-phase MemoryService boundary. `PrepareMutation` captures semantic context and an exact Git revision; `ApplyMutation` owns canonical placement, lifecycle/relationship enforcement, Index v2 regeneration, hard validation, idempotency, commit creation, and optimistic-concurrency publication.

Some external AI clients can read and write a GitHub repository but cannot execute Runethread Core locally. The current degraded fallback for such clients is manual canonical Markdown/JSON editing followed by later Index v2 regeneration. That fallback is useful for recovery, but it should not become the permanent delivery architecture when an execution-capable runner can invoke the existing MemoryService.

Phase 2.6 therefore needs a delivery boundary between a semantic caller and canonical publication. The design must avoid two weak outcomes:

1. publishing unaudited state to canonical `main` and attempting to repair or revert it later; and
2. storing operational flags such as `audit_pending` or `store_incomplete` inside canonical memory JSON.

The first weakens the meaning of canonical state. The second mixes transient execution state into durable semantic memory and would make clearing the flag another canonical mutation.

This ADR is intentionally compatible with ADR-001 through ADR-005: canonical semantic memory remains in the user-owned Git repository; semantic judgment remains outside deterministic Core; MemoryService remains the mutation owner; exact Git revisions remain the concurrency boundary; and no GitHub-specific dependency is introduced into Core itself.

## Decision

Phase 2.6 uses an **audited candidate-before-canonical publication** model for execution-capable external memory delivery.

### Structured request, not manual canonical editing

When the delivery path has access to an execution-capable Runethread runtime, the semantic caller supplies a structured MemoryService-compatible mutation request containing the intended operation, expected revision, stable idempotency key, mutation time, and proposed semantic content where required.

The delivery mechanism MUST invoke the same deterministic Core/MemoryService mutation logic used by CLI/native callers. It MUST NOT reimplement canonical path construction, lifecycle rules, relationship rules, indexing, validation, idempotency, or Git transaction semantics in workflow scripts or provider-specific adapters.

Direct canonical Markdown/JSON editing remains a degraded/recovery path for clients that truly cannot execute Core; it is not the Phase 2.6 normal path.

### Operation state is not memory state

A delivery operation has its own identity and lifecycle. At minimum the control layer must be able to associate:

- operation identity;
- idempotency identity/key;
- prepared/base revision;
- current delivery state;
- candidate revision when one exists;
- audit result bound to an exact candidate;
- final canonical revision or failure outcome.

Pending, queued, audit, retry, suspension, and failure state MUST NOT be represented by fields added to canonical memory sidecars solely for delivery bookkeeping.

Phase 2.6 does not introduce a new authoritative queue database. In the GitHub-backed implementation, workflow-run state is transitional operational state. Durable recovery anchors remain the operation/idempotency identity and canonical Git history/commit metadata. If an uncommitted execution record is lost, the caller may resubmit the same operation identity and normal MemoryService idempotency/revision rules determine whether it was already committed, is still runnable, or now requires re-preparation.

A later hosted executor such as Cloudflare or the future Orchestrator may replace the GitHub execution backend without changing canonical memory format or the logical delivery lifecycle.

### Phase 2.6 GitHub submission profile

The GitHub-backed Phase 2.6 adapter accepts one complete sealed MemoryService-compatible request through an explicit workflow dispatch operation rather than by watching a caller assemble request files over multiple commits.

The dispatch payload carries the complete operation request or an explicitly supported immutable payload reference plus the operation identity. Request submission therefore has one deliberate admission boundary and cannot race against a half-written canonical request tree.

The GitHub workflow mechanism is an adapter implementation detail. Correctness MUST NOT depend on GitHub's workflow scheduling order.

### Candidate construction

The finalizer executes the requested MemoryService operation from the exact prepared/base revision in an isolated execution environment or clone whose canonical branch is not changed merely by candidate construction.

The produced candidate must represent the same deterministic mutation transaction that Core normally performs, including:

- canonical semantic transition;
- Index v2 regeneration;
- hard repository validation;
- mutation commit metadata and idempotency semantics;
- failure without canonical publication when deterministic invariants fail.

An execution-capable finalizer MUST NOT use `index/STALE` as a successful substitute for Index v2 regeneration.

The candidate may be exposed to the auditor through a noncanonical Git ref or another immutable transport, but the audit and publisher must bind to the exact candidate revision/tree. Candidate transport must never make the candidate canonical before audit.

### Independent audit

A candidate is not eligible for canonical publication until a separate fresh read-only execution job verifies the **exact candidate revision/tree**.

For the Phase 2.6 GitHub profile, the auditor runs on a fresh GitHub-hosted runner/job with read-only repository authority. It does not share the finalizer workspace and cannot repair or publish the candidate.

The audit uses Runethread's deterministic implementation and contract, not a second hand-written interpretation. At minimum it verifies applicable trust/control-plane state, hard repository validity, strict Index v2 freshness/integrity, expected mutation metadata, and that the audited candidate is exactly the candidate associated with the operation being promoted.

Independence here means fresh environment plus reduced privileges over immutable candidate evidence; it does not mean maintaining a second implementation of Runethread semantics.

Auditor/finalizer disagreement is treated as an integrity incident rather than an invitation to retry random repairs until green.

### Canonical promotion and revision check

Git's canonical commit SHA is the publication version token. Phase 2.6 does not add a second repository version counter merely to detect races.

Immediately before publication, the publisher re-reads the canonical branch/ref and requires it to equal the operation's expected/base revision. This check is intentionally cheap and MUST NOT be weakened into sparse sampling or a cached advisory flag.

Only an audited candidate may be promoted. The normal Phase 2.6 publication mechanism is an exact **fast-forward compare-and-swap** from the expected canonical revision to the exact audited candidate commit:

```text
canonical = H0
candidate = C

audit exact C
verify canonical still H0
fast-forward H0 -> C
```

No force-push or last-writer-wins publication is allowed. If canonical state moved, publication fails and the operation requires re-preparation/re-evaluation according to ADR-013.

### Dedicated publisher identity and audited-only canonical enforcement

The GitHub-backed Phase 2.6 profile uses a dedicated Runethread GitHub App as the canonical publisher for the user's memory repository. The app is installed only on the memory repositories the user explicitly authorizes; it is not a publisher for `runethread/core` and must not receive access to unrelated repositories by default.

The publisher's repository permissions must be narrowly scoped to what audited canonical publication requires. It must not receive workflow-modification authority merely because it can update memory repository content.

A managed memory-repository ruleset/branch policy must prevent ordinary agents and routine user tokens from bypassing the audited publication path while permitting the dedicated publisher to perform the exact approved fast-forward. The repository owner retains ultimate administrative control; if canonical state is changed out-of-band, Runethread treats that movement as outside the audited delivery invariant until the repository is independently validated/reconciled.

### Canonical read boundary

Canonical retrieval MUST use audited canonical state. Pending candidates are not silently mixed into ordinary trusted memory search/get results.

A future explicit read-your-own-pending view may be added, but it must identify pending state as noncanonical and must not weaken canonical retrieval semantics.

### Audit failure, publication result, and asynchronous main audit

If deterministic finalization fails, independent prepublication audit fails, or the auditor and finalizer disagree about the exact candidate, canonical state remains unchanged.

The affected repository write lane enters a suspended/fail-closed condition until the discrepancy is understood or an authorized recovery procedure resolves it. Reads from the last known audited canonical revision remain available.

A delivery operation becomes `COMMITTED` when the exact audited candidate is successfully published through the compare-and-swap boundary. Normal read-only validation of the resulting canonical `main` runs asynchronously as an independent post-publication audit and is not required to keep the caller synchronously waiting after the exact audited publication succeeds. A post-publication red audit is an incident signal and suspends further writes until investigated.

## Consequences

- Canonical memory does not need operational `pending` flags.
- The user-visible store operation may be asynchronous even though canonical state remains audited-only.
- The delivery adapter can return an operation identity/status before canonical publication without claiming the memory is already committed.
- MemoryService remains the single deterministic mutation implementation.
- Index regeneration and hard validation happen before audit; audit independently proves the exact resulting candidate rather than repairing it.
- A stale canonical revision causes re-preparation instead of silent Git rebasing.
- Git commit identity remains the concurrency/version boundary; no duplicate version counter is introduced.
- The Phase 2.6 GitHub executor is explicitly replaceable by a hosted executor without changing canonical memory or Core's deterministic boundary.
- A dedicated publisher identity and managed repository policy become part of the GitHub-backed security model.
- The finalizer and auditor incur separate execution environments in Phase 2.6; this latency is accepted because canonicalization is asynchronous and the backend is replaceable.

## Alternatives considered

### Publish to canonical `main`, then audit asynchronously

Rejected as the target model. Audit failure would leave canonical state already containing data that Runethread no longer trusts, requiring revert/freeze/serve-previous-state logic.

### Add `audit_pending` / `store_incomplete` to memory JSON

Rejected. Delivery execution state is not semantic memory state, and clearing the flag would itself create another canonical mutation/index cycle.

### Let the external agent edit canonical files and have Actions only rebuild the index

Rejected as the normal execution-capable path. ADR-002 and MemoryService already establish deterministic Core as the owner of canonical mutation invariants. Manual canonical editing remains only a degraded/recovery path.

### Make GitHub Actions the architectural queue authority

Rejected. GitHub Actions is the initial execution adapter, not the semantic architecture. Provider scheduling behavior and runner latency must be replaceable without changing Runethread's canonical state model.

### Use a second repository version flag instead of exact Git revisions

Rejected. The canonical Git commit SHA already identifies the complete repository state and is the existing ADR-004 concurrency boundary. A duplicate counter would create another value that could itself drift or race.

### Publish through an ordinary PR merge

Rejected for the normal data-plane path. A merge or squash can create a commit different from the exact audited MemoryService candidate. Exact fast-forward publication preserves the audited candidate and its mutation/idempotency metadata as canonical history.

### Make the auditor a second independent implementation of Runethread rules

Rejected. Two implementations would drift. Independence means a separate observational execution boundary over the exact candidate while deterministic semantics remain owned by Core.

## Verification

An implementation complies with this ADR only if tests/integration evidence demonstrate:

1. an execution-capable external delivery path invokes MemoryService/Core rather than manually constructing canonical storage changes;
2. operation pending/audit state does not appear in canonical memory sidecars;
3. the GitHub-backed request enters through one sealed explicit dispatch rather than a half-written watched request tree;
4. candidate construction from an exact base cannot publish to remote canonical state before audit;
5. deterministic finalization rebuilds Index v2 and hard-validates the candidate;
6. strict `index --check` passes for an execution-capable finalized candidate;
7. a fresh read-only auditor validates the exact candidate revision/tree associated with the operation and cannot publish it;
8. audit failure leaves canonical state unchanged and suspends further writes according to policy;
9. publication re-reads canonical Git state and fails if it no longer equals the expected revision;
10. successful publication fast-forwards the canonical ref to the exact audited candidate commit without force or an intervening merge commit;
11. the dedicated GitHub App can publish only to explicitly authorized memory repositories with the minimum intended permissions, while ordinary data-plane writers cannot bypass the audited path under the managed repository policy;
12. a stale candidate is not silently rebased or semantically reinterpreted;
13. exact idempotent retries retain existing MemoryService semantics after lost responses or runner failure;
14. canonical search/get never silently returns unaudited pending candidates;
15. post-publication `main` validation remains read-only and a red result suspends later writes rather than retroactively pretending the failed audit was green;
16. replacing the GitHub execution adapter with another executor does not require a memory schema/repository-format change solely for delivery state;
17. the implementation remains usable without an MCP adapter and does not make Core depend on GitHub or the future Orchestrator.
