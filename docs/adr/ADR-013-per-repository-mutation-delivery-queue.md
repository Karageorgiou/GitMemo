# ADR-013: Per-repository serialized mutation-delivery queue

Status: **Proposed**
Date: 2026-09-05
Tracking issue: #20

## Context

Runethread memory repositories may eventually receive overlapping writes from multiple AI clients, browser/desktop sessions, delayed jobs, local tools, and remote execution paths. Git-revision optimistic concurrency already protects canonical writes from silent last-writer-wins behavior, but an external delivery layer still needs to coordinate which prepared operations are allowed to proceed and how failures affect later work.

This problem is broader than ordinary Git file conflicts. Two operations can write different canonical files yet still be semantically dependent, and all normal memory mutations can affect shared derived Index v2 state. Git mergeability is therefore not proof that two previously prepared semantic decisions remain valid when executed in parallel.

The project has also considered batching neighboring queued operations to amortize indexing/audit cost. That optimization would require a trustworthy model of semantic read/write dependencies based partly on what the semantic caller actually retrieved and relied on, not merely the final Git diff. Phase 2.6 should not block on solving that deeper problem.

## Decision

Phase 2.6 will use a **durable logical mutation-delivery queue per canonical memory repository** and a **single canonical publication lane per repository**.

### Queue scope

Serialization is per memory repository, not global across Runethread users.

Independent repositories may finalize/audit/publish concurrently. Within one repository, only one operation may occupy the canonical publication critical section at a time.

### Preparation versus publication

Semantic preparation may happen concurrently against the same or different observed revisions.

Canonical publication is serialized and remains subject to ADR-004's exact Git-revision optimistic-concurrency rule. Queue admission or local serialization does not replace the expected-revision check.

### Conceptual operation lifecycle

The delivery layer must represent a deterministic state machine equivalent in meaning to:

```text
ACCEPTED
   |
QUEUED
   |
FINALIZING
   |
CANDIDATE_READY
   |
AUDIT_PENDING
   |
AUDITED
   |
PUBLISHING
   |
COMMITTED
```

and must also represent non-success outcomes equivalent to:

```text
NEEDS_REPREPARE
FINALIZATION_FAILED
AUDIT_FAILED
CANCELLED
```

The exact persisted enum names may be refined before a public API is frozen, but impossible boolean combinations such as independently mutable `queued=true`, `audit_pending=true`, and `complete=false` are prohibited. Convenience flags such as `incomplete` should be derived from one authoritative operation state.

### Stale operations

If an operation's prepared/base revision no longer matches the canonical publication precondition, it MUST NOT be silently rebased or applied as though the original semantic decision were still current.

The operation transitions to a state equivalent to `NEEDS_REPREPARE`. Phase 2.6 does not automatically invoke an AI/model to reinterpret the user's intent. A caller or later orchestration layer must re-run semantic preparation against current canonical state.

An exact idempotent retry of an operation already committed retains the existing ADR-003/MemoryService exception and returns the original committed result rather than being treated as a new stale operation.

### Failure and repository suspension

A deterministic finalization failure affects that operation and does not publish canonical state.

An independent-audit failure or finalizer/auditor disagreement is more severe: the repository write lane enters a suspended/fail-closed state until an authorized recovery procedure resolves the discrepancy. Canonical reads from the last audited revision remain available.

The scheduler must preserve evidence for the failed exact candidate/revision rather than repeatedly mutating it until a check happens to pass.

### Exclusive barriers

Changes that alter the interpretation or safety boundary of ordinary memory operations act as **exclusive repository barriers**. At minimum this includes changes to:

- operational contract semantics;
- memory schema;
- repository format/layout;
- trust/lock semantics;
- bootstrap protocol/workflow semantics;
- supported migration state;
- managed validation/finalization workflow semantics when those workflows are part of the delivery boundary.

A barrier cannot be coalesced with ordinary memory operations. Work admitted before the barrier must reach a safe terminal point before the barrier executes. Operations prepared under pre-barrier semantics must not be published after the barrier without re-preparation under the new state.

### Batching/coalescing is deferred

Phase 2.6 v1 deliberately uses **singleton execution groups**: one logical operation per finalization/audit/promotion transaction.

The queue/scheduler abstraction should leave room for a future batch/group concept, but correctness MUST NOT depend on batching being enabled.

Future neighboring-operation coalescing requires a separate accepted design that addresses at least:

- semantic read/write/dependency sets rather than changed-path overlap alone;
- operation ordering and fairness;
- preservation of each operation's identity, idempotency, provenance, and result;
- all-or-nothing batch publication;
- failure isolation and bounded batch size;
- explicit dependency groups versus provably commuting operations;
- the effect of retrieval context on semantic dependence.

Until that proof exists, unknown semantic dependence means operations are not coalesced.

### No automatic queue-wide semantic repair

The scheduler may perform deterministic state transitions, exact retries, and fail-closed publication checks. It must not invent semantic corrections, silently rewrite a proposed memory, or automatically reinterpret user intent merely to keep the queue moving.

## Consequences

- Runethread gains one clear mutation serialization boundary per repository without requiring a global distributed lock.
- Independent users/repositories can scale horizontally even while each repository retains strict publication ordering.
- Concurrent preparation can still produce stale operations; the queue makes that outcome explicit rather than unsafe.
- A stale head operation may require caller/model participation before it can become runnable again.
- Global Index v2 derived-state races disappear from the publication critical section because only one operation publishes at a time.
- Control-plane transitions become explicit scheduling barriers rather than surprising reinterpretations of already-prepared work.
- Phase 2.6 can ship without solving semantic dependency quantification or batching.
- Later MCP/Orchestrator work can expose or automate the same state machine without changing the underlying canonical-memory rules.

## Alternatives considered

### Let every caller race directly against Git and rely only on stale-revision failures

Rejected as the delivery-layer design. ADR-004 remains the final correctness boundary, but an explicit queue gives observable operation state, deterministic scheduling, failure suspension, and a place to enforce exclusive barriers.

### Use a global Runethread queue

Rejected. It would serialize unrelated users/repositories, create unnecessary availability coupling, and scale poorly.

### Automatically rebase any Git-mergeable stale candidate

Rejected. File-level mergeability is not proof that the semantic decision remains valid after another operation changed canonical context.

### Enable diff-based batching immediately

Rejected for Phase 2.6. Non-overlapping changed paths do not prove semantic independence, and Index v2 is shared derived state.

### Put queue flags inside memory sidecars

Rejected. Operational scheduling state is not semantic memory state and would violate the canonical-state ownership boundary.

## Decisions intentionally left open before acceptance

The following scheduling/runtime choices are tracked in issue #20 and must be resolved before implementation is finalized:

1. **Durable queue-state owner/backend** before the Orchestrator exists.
2. **Head-of-line `NEEDS_REPREPARE` behavior:** strict blocking versus a deterministic rule that can park the stale operation while later independently-current work proceeds without unsafe reordering.
3. **Crash/lease recovery:** how a `FINALIZING`, `AUDIT_PENDING`, or `PUBLISHING` operation is recovered after runner/process death without double publication.
4. **Cancellation boundary:** at which states cancellation is allowed and what happens when a candidate has already been audited but not published.
5. **Queue ordering key:** accepted time, sealed/submitted time, or another deterministic ordering field.

No choice above may weaken the exact-revision publication boundary or introduce automatic semantic re-interpretation in Phase 2.6.

## Verification

An implementation complies with this ADR only if tests/integration evidence demonstrate:

1. two repositories can execute independently while one repository's publication lane remains single-writer;
2. two operations prepared from the same revision cannot both publish unchanged assumptions after the first advances canonical state;
3. the stale operation becomes `NEEDS_REPREPARE` (or equivalent) without modifying canonical state;
4. exact committed idempotent retry still returns the original operation result;
5. deterministic finalization failure leaves later canonical state unchanged;
6. audit disagreement suspends the repository write lane while reads remain available;
7. exclusive barriers prevent pre-barrier semantic operations from publishing under post-barrier semantics without re-preparation;
8. Phase 2.6 v1 works correctly with batch size effectively fixed at one;
9. disabling all future batching/coalescing logic cannot change correctness;
10. no scheduler path silently rebases, rewrites, or semantically repairs a stale proposed mutation;
11. queue/operation state remains outside canonical memory sidecars and generated indexes;
12. the queue does not replace ADR-004's exact Git revision check at publication.
