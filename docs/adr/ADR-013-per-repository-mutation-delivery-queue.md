# ADR-013: Per-repository serialized mutation-delivery queue

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20

## Context

Runethread memory repositories may eventually receive overlapping writes from multiple AI clients, browser/desktop sessions, delayed jobs, local tools, and remote execution paths. Git-revision optimistic concurrency already protects canonical writes from silent last-writer-wins behavior, but an external delivery layer still needs to coordinate which prepared operations are allowed to proceed and how failures affect later work.

This problem is broader than ordinary Git file conflicts. Two operations can write different canonical files yet still be semantically dependent, and all normal memory mutations can affect shared derived Index v2 state. Git mergeability is therefore not proof that two previously prepared semantic decisions remain valid when executed in parallel.

The project has also considered batching neighboring queued operations to amortize indexing/audit cost. That optimization would require a trustworthy model of semantic read/write dependencies based partly on what the semantic caller actually retrieved and relied on, not merely the final Git diff. Phase 2.6 does not block on solving that deeper problem.

## Decision

Phase 2.6 uses a **logical mutation-delivery queue per canonical memory repository** and a **single canonical publication lane per repository**.

The queue/state machine is provider-independent. The Phase 2.6 implementation uses GitHub Actions as the first execution adapter, but correctness does not depend on GitHub's workflow ordering or runner latency. A later hosted backend such as Cloudflare or the future Orchestrator may replace that execution adapter without changing canonical memory format or the queue's logical safety rules.

### Queue scope

Serialization is per memory repository, not global across Runethread users.

Independent repositories may finalize/audit/publish concurrently. Within one repository, only one operation may occupy the canonical publication critical section at a time.

### Preparation versus publication

Semantic preparation may happen concurrently against the same or different observed revisions.

Canonical publication is serialized and remains subject to ADR-004's exact Git-revision optimistic-concurrency rule. Queue admission, GitHub concurrency controls, or local serialization do not replace the expected-revision check.

### Phase 2.6 GitHub execution adapter

A GitHub-only caller submits one sealed structured mutation request through the explicit dispatch boundary defined by ADR-012. The resulting workflow execution is the Phase 2.6 operational envelope for that attempt.

GitHub workflow-run state is transitional operational state, not a new canonical database. The adapter may use provider concurrency grouping or equivalent controls to avoid intentionally running multiple publication transactions at once, but Runethread MUST NOT assume that GitHub dispatch order equals execution order.

Strict FIFO ordering is therefore **not** a Phase 2.6 correctness guarantee. Submission/acceptance time may be retained for observability and best-effort fairness, but every runnable operation independently proves that its expected Git revision is still current before publication.

A later backend may offer stronger ordered scheduling without changing the underlying safety model.

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

### Stale operations and head-of-line behavior

If an operation's prepared/base revision no longer matches the canonical publication precondition, it MUST NOT be silently rebased or applied as though the original semantic decision were still current.

The operation transitions to a state equivalent to `NEEDS_REPREPARE` and is **parked outside the runnable publication lane**. It does not indefinitely block later operations merely because it arrived earlier.

A later operation may proceed only if its own expected revision and all ordinary publication invariants are current. If it was prepared against the same stale revision, it will independently become `NEEDS_REPREPARE` as well.

Phase 2.6 does not automatically invoke an AI/model to reinterpret the parked operation. A caller or later orchestration layer must run semantic preparation again against current canonical state and submit a new current attempt.

An exact idempotent retry of an operation already committed retains the ADR-003/MemoryService exception and returns the original committed result rather than being treated as a new stale operation.

### Crash and retry recovery

Runner/process death before canonical publication must be recoverable through the existing operation/idempotency identity rather than through hidden mutable queue repair.

The caller or scheduler may resubmit the same sealed request with the same stable idempotency identity:

- if the operation already committed, MemoryService returns the existing committed result;
- if it did not commit and the expected revision is still current, finalization may run again safely;
- if canonical state moved, the attempt becomes `NEEDS_REPREPARE`;
- abandoned noncanonical candidates may be garbage-collected after recovery evidence no longer depends on them.

A recovered attempt must never assume an old candidate remains audited merely because a prior runner reached `AUDIT_PENDING`; audit evidence must remain bound to the exact candidate revision actually being published.

### Cancellation boundary

Cancellation is allowed while an operation is noncanonical and before the publication critical section begins. A cancelled candidate is abandoned without modifying canonical state.

Once publication enters `PUBLISHING`, cancellation is not a correctness mechanism. The system must resolve the exact publication outcome as committed, failed, or stale/reprepare rather than pretending a best-effort cancellation reversed an already-started compare-and-swap.

### Failure and repository suspension

A deterministic finalization failure affects that operation and does not publish canonical state.

An independent-audit failure, finalizer/auditor disagreement, or red post-publication independent audit is more severe: the repository write lane enters a suspended/fail-closed state until an authorized recovery procedure resolves the discrepancy. Canonical reads from the last audited/accepted revision remain available as allowed by ADR-012.

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

The Phase 2.6 GitHub adapter may realize the barrier through a maintenance/suspension mechanism appropriate to that repository, but the barrier semantics are provider-independent.

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
- the effect of retrieval context and the agent's interpretation of the user's request on semantic dependence.

Until that proof exists, unknown semantic dependence means operations are not coalesced.

### No automatic queue-wide semantic repair

The scheduler may perform deterministic state transitions, exact retries, fail-closed publication checks, parking, cancellation, and crash recovery. It must not invent semantic corrections, silently rewrite a proposed memory, or automatically reinterpret user intent merely to keep the queue moving.

Automatic semantic re-preparation is explicitly deferred to later MCP/Orchestrator work or another accepted design.

## Consequences

- Runethread gains one clear mutation serialization boundary per repository without requiring a global distributed lock.
- Independent users/repositories can scale horizontally even while each repository retains a single publication lane.
- Concurrent preparation can still produce stale operations; the queue makes that outcome explicit rather than unsafe.
- A stale operation is parked rather than blocking the repository forever, while later work still proves its own exact revision.
- Global Index v2 derived-state races disappear from the publication critical section because only one operation publishes at a time.
- Queue safety does not depend on provider workflow ordering, which makes a future Cloudflare/Orchestrator backend replacement straightforward.
- Crash/lost-response recovery uses the existing idempotency/revision model rather than a second semantic recovery implementation.
- Control-plane transitions become explicit scheduling barriers rather than surprising reinterpretations of already-prepared work.
- Phase 2.6 can ship without solving semantic dependency quantification or batching.
- Later MCP/Orchestrator work can expose or automate the same state machine without changing the underlying canonical-memory rules.

## Alternatives considered

### Let every caller race directly against Git and rely only on stale-revision failures

Rejected as the delivery-layer design. ADR-004 remains the final correctness boundary, but an explicit queue gives observable operation state, deterministic state transitions, failure suspension, and a place to enforce exclusive barriers.

### Use a global Runethread queue

Rejected. It would serialize unrelated users/repositories, create unnecessary availability coupling, and scale poorly.

### Make GitHub Actions ordering the queue correctness model

Rejected. Provider scheduling order is not a semantic guarantee and may change. Exact Git revision checks and the single publication lane are the correctness boundaries.

### Strictly block all later work behind `NEEDS_REPREPARE`

Rejected for Phase 2.6. A stale operation is no longer runnable and may require external semantic judgment. Keeping the entire repository blocked behind it adds availability cost without improving the revision proof for later independently current operations.

### Automatically rebase any Git-mergeable stale candidate

Rejected. File-level mergeability is not proof that the semantic decision remains valid after another operation changed canonical context.

### Enable diff-based batching immediately

Rejected for Phase 2.6. Non-overlapping changed paths do not prove semantic independence, and Index v2 is shared derived state.

### Put queue flags inside memory sidecars

Rejected. Operational scheduling state is not semantic memory state and would violate the canonical-state ownership boundary.

### Add a Phase 2.6 database solely to make queue state durable

Rejected for the first implementation. It would pull future orchestration infrastructure into the memory-delivery milestone. Git/idempotency provide the durable committed/recovery boundary while the GitHub adapter owns replaceable transient execution state.

## Verification

An implementation complies with this ADR only if tests/integration evidence demonstrate:

1. two repositories can execute independently while one repository's canonical publication lane remains single-writer;
2. correctness remains intact when submitted workflow operations execute in a different order from submission order;
3. two operations prepared from the same revision cannot both publish unchanged assumptions after the first advances canonical state;
4. the stale operation becomes `NEEDS_REPREPARE` (or equivalent), leaves the runnable lane, and does not modify canonical state;
5. a later operation proceeds only when its own expected revision is current;
6. exact committed idempotent retry still returns the original operation result;
7. a simulated runner crash before publication can be safely resubmitted without duplicate canonical mutation;
8. cancellation before `PUBLISHING` leaves canonical state unchanged, while cancellation during publication is resolved by checking the exact publication outcome rather than assuming rollback;
9. deterministic finalization failure leaves later canonical state unchanged;
10. audit disagreement or red independent audit suspends the repository write lane while reads remain available according to policy;
11. exclusive barriers prevent pre-barrier semantic operations from publishing under post-barrier semantics without re-preparation;
12. Phase 2.6 v1 works correctly with batch size effectively fixed at one;
13. disabling all future batching/coalescing logic cannot change correctness;
14. no scheduler path silently rebases, rewrites, or semantically repairs a stale proposed mutation;
15. queue/operation state remains outside canonical memory sidecars and generated indexes;
16. the queue does not replace ADR-004's exact Git revision check at publication;
17. replacing the GitHub Actions execution adapter with another backend does not require a memory schema/repository-format change solely for queue state.
