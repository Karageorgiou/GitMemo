# ADR-012: Audited candidate promotion for external memory delivery

Status: **Proposed**
Date: 2026-09-05
Tracking issue: #20

## Context

Runethread already has a deterministic two-phase MemoryService boundary. `PrepareMutation` captures semantic context and an exact Git revision; `ApplyMutation` owns canonical placement, lifecycle/relationship enforcement, Index v2 regeneration, hard validation, idempotency, commit creation, and optimistic-concurrency publication.

Some external AI clients can read and write a GitHub repository but cannot execute Runethread Core locally. The current degraded fallback for such clients is manual canonical Markdown/JSON editing followed by later Index v2 regeneration. That fallback is useful for recovery, but it should not become the permanent delivery architecture when an execution-capable runner can invoke the existing MemoryService.

Phase 2.6 therefore needs a delivery boundary between a semantic caller and canonical publication. The design must avoid two weak outcomes:

1. publishing unaudited state to canonical `main` and attempting to repair or revert it later; and
2. storing operational flags such as `audit_pending` or `store_incomplete` inside canonical memory JSON.

The first weakens the meaning of canonical state. The second mixes transient execution state into durable semantic memory and would make clearing the flag another canonical mutation.

This ADR is intentionally compatible with ADR-001 through ADR-005: canonical semantic memory remains in the user-owned Git repository; semantic judgment remains outside deterministic Core; MemoryService remains the mutation owner; exact Git revisions remain the concurrency boundary; and no GitHub-specific dependency is introduced into Core.

## Decision

Phase 2.6 will use an **audited candidate-before-canonical publication** model for execution-capable external memory delivery.

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

### Candidate construction

The finalizer executes the requested MemoryService operation from the exact prepared/base revision in an isolated execution environment or clone whose remote canonical branch is not changed merely by local candidate construction.

The produced candidate must represent the same deterministic mutation transaction that Core normally performs, including:

- canonical semantic transition;
- Index v2 regeneration;
- hard repository validation;
- mutation commit metadata and idempotency semantics;
- failure without canonical publication when deterministic invariants fail.

An execution-capable finalizer MUST NOT use `index/STALE` as a successful substitute for Index v2 regeneration.

### Independent audit

A candidate is not eligible for canonical publication until an independent read-only audit verifies the **exact candidate revision/tree** in a fresh execution context.

The audit must use Runethread's deterministic implementation and contract, not a second hand-written interpretation. At minimum it must verify applicable trust/control-plane state, hard repository validity, strict Index v2 freshness/integrity, and that the audited candidate is the candidate associated with the operation being promoted.

Auditor/finalizer disagreement is treated as an integrity incident rather than an invitation to retry random repairs until green.

### Canonical promotion

Only an audited candidate may be promoted to the canonical memory branch/ref.

Promotion MUST fail closed unless all of the following remain true at publication time:

1. the canonical repository is still at the operation's expected/base revision or another explicitly accepted equivalent precondition;
2. the exact candidate being promoted is the candidate that passed audit;
3. the publication operation cannot silently overwrite a newer canonical revision.

If canonical state moved, the candidate is not semantically rebased automatically. The operation requires re-preparation/re-evaluation unless a future accepted ADR defines a deterministic independence proof for that class of operation.

### Canonical read boundary

Canonical retrieval MUST use audited canonical state. Pending candidates are not silently mixed into ordinary trusted memory search/get results.

A future explicit read-your-own-pending view may be added, but it must identify pending state as noncanonical and must not weaken canonical retrieval semantics.

### Audit failure and suspension

If deterministic finalization fails, independent audit fails, or the auditor and finalizer disagree about the exact candidate, canonical state remains unchanged.

The affected repository write lane enters a suspended/fail-closed condition until the discrepancy is understood or an authorized recovery procedure resolves it. Reads from the last known audited canonical revision remain available.

## Consequences

- Canonical memory does not need operational `pending` flags.
- The user-visible store operation may be asynchronous even though canonical state remains audited-only.
- The delivery adapter can return an operation identity/status before canonical publication without claiming the memory is already committed.
- MemoryService remains the single deterministic mutation implementation.
- Index regeneration and hard validation happen before audit; audit independently proves the exact resulting candidate rather than repairing it.
- A stale canonical revision causes re-preparation instead of silent Git rebasing.
- A delivery implementation needs a control-plane representation for operation state that is separate from semantic memory.
- The stronger claim that `main` is always audited-only requires an enforcement mechanism that prevents unmanaged direct publication; that mechanism is still an open Phase 2.6 decision.

## Alternatives considered

### Publish to canonical `main`, then audit asynchronously

Rejected as the target model. Audit failure would leave canonical state already containing data that Runethread no longer trusts, requiring revert/freeze/serve-previous-state logic.

### Add `audit_pending` / `store_incomplete` to memory JSON

Rejected. Delivery execution state is not semantic memory state, and clearing the flag would itself create another canonical mutation/index cycle.

### Let the external agent edit canonical files and have Actions only rebuild the index

Rejected as the normal execution-capable path. ADR-002 and MemoryService already establish deterministic Core as the owner of canonical mutation invariants. Manual canonical editing remains only a degraded/recovery path.

### Make the auditor a second independent implementation of Runethread rules

Rejected. Two implementations would drift. Independence means a separate observational execution boundary over the exact candidate, while deterministic semantics remain owned by Core.

## Decisions intentionally left open before acceptance

This ADR defines the invariant and lifecycle boundary but does not yet choose the following provider/runtime mechanisms. They are tracked in issue #20 and must be resolved before Phase 2.6 implementation is treated as final:

1. **Operation/queue state owner:** where durable Phase 2.6 delivery state lives before the Orchestrator exists.
2. **Request submission/seal mechanism:** how a GitHub-only caller submits one complete request without racing a half-written multi-commit request.
3. **Promotion mechanism:** exact fast-forward of a canonical ref, PR-based promotion with exact tree/base proof, or another compare-and-swap mechanism.
4. **Audited-only enforcement:** how unmanaged direct writes to the canonical branch/ref are prevented or detected without weakening the invariant.
5. **Independent-audit boundary:** whether separate jobs/runners, separate workflow runs, or stronger isolation is required.

These choices MUST NOT be solved by adding GitHub dependencies inside Core or by introducing a repository-format/contract change merely to store runtime delivery metadata unless a later impact review independently proves such a change necessary.

## Verification

An implementation complies with this ADR only if tests/integration evidence demonstrate:

1. an execution-capable external delivery path invokes MemoryService/Core rather than manually constructing canonical storage changes;
2. operation pending/audit state does not appear in canonical memory sidecars;
3. candidate construction from an exact base cannot publish to remote canonical state before audit;
4. deterministic finalization rebuilds Index v2 and hard-validates the candidate;
5. strict `index --check` passes for an execution-capable finalized candidate;
6. a fresh read-only auditor validates the exact candidate revision/tree associated with the operation;
7. audit failure leaves canonical state unchanged and suspends further writes according to policy;
8. publication fails when canonical state moved after preparation/candidate construction;
9. a stale candidate is not silently rebased or semantically reinterpreted;
10. exact idempotent retries retain existing MemoryService semantics;
11. canonical search/get never silently returns unaudited pending candidates;
12. the implementation remains usable without an MCP adapter and does not make Core depend on GitHub or the future Orchestrator.
