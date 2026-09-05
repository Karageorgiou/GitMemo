# Architecture Decision Records

Runethread uses Architecture Decision Records (ADRs) for durable architectural choices that affect compatibility, trust, security, state ownership, or component boundaries.

The architecture documents describe the target system. Accepted decisions govern implementation until they are explicitly superseded.

## Status values

- **Proposed** — under review; implementation must not assume it is final.
- **Accepted** — current architectural decision.
- **Superseded** — replaced by a newer ADR that explains why.
- **Rejected** — considered and intentionally not adopted.

## Decision catalog

| ADR | Decision | Status |
| --- | --- | --- |
| [ADR-001](ADR-001-canonical-state-ownership.md) | Canonical state ownership | Accepted |
| [ADR-002](ADR-002-deterministic-core-semantic-ai.md) | Deterministic Core and semantic AI boundary | Accepted |
| [ADR-003](ADR-003-two-phase-memory-mutations.md) | Two-phase memory mutation protocol | Accepted |
| [ADR-004](ADR-004-git-optimistic-concurrency.md) | Git-revision optimistic concurrency | Accepted |
| [ADR-005](ADR-005-core-orchestrator-separation.md) | Core and Orchestrator are separate components/repositories | Accepted |
| [ADR-006](ADR-006-capability-based-workers.md) | Capability-based worker abstraction | Accepted |
| [ADR-007](ADR-007-isolated-git-worktrees.md) | Isolated Git worktrees for delegated code mutation | Accepted |
| [ADR-008](ADR-008-permissions-and-approvals.md) | Least-privilege permission and approval model | Accepted |
| [ADR-009](ADR-009-controlled-runethread-cutover.md) | Controlled GitMemo-to-Runethread cutover | Accepted |
| [ADR-010](ADR-010-organization-repository-topology.md) | Runethread organization and repository topology | Accepted |
| [ADR-011](ADR-011-runtime-contract-release-separation.md) | Runtime and contract release separation | Accepted |
| [ADR-012](ADR-012-audited-candidate-memory-delivery.md) | Audited candidate promotion for external memory delivery | Accepted |
| [ADR-013](ADR-013-per-repository-mutation-delivery-queue.md) | Per-repository serialized mutation-delivery queue | Accepted |
| [ADR-014](ADR-014-cloud-hosted-memory-delivery-control-plane.md) | Cloud-hosted Phase 2.6 memory-delivery control plane | Accepted |
| [ADR-015](ADR-015-project-current-state-projection-semantics.md) | Project current-state documents are asynchronous orientation projections in the next contract | Accepted |
| [ADR-016](ADR-016-phase-2-6-hosted-trust-boundary-hardening.md) | Phase 2.6 hosted trust-boundary hardening | Accepted |
| [ADR-017](ADR-017-reconciliation-privacy-and-publication-fencing.md) | Phase 2.6 reconciliation, repository privacy, and publication fencing | Accepted |

ADR-014 amends the initial GitHub-Actions-backed implementation profile described in ADR-012/ADR-013. Their candidate-before-canonical, independent-audit, exact-revision publication, idempotency, stale-reprepare, and per-repository serialization invariants remain accepted.

ADR-015 records a required **next-contract** semantic change. Contract v8 remains immutable and continues to require relevant project current-state synchronization until a repository is explicitly migrated to the contract release implementing ADR-015.

ADR-016 hardens the Phase 2.6 hosted boundary. It makes the ADR-015 projection-capable contract a normal hosted-write admission prerequisite, separates finalizer/auditor evidence-write capabilities, pins live evidence against premature retention/GC, and replaces the unproven Worker-only REST-ref-CAS assumption with a concrete minimal Git publisher executor unless a future GitHub API proves true expected-old ref CAS.

ADR-017 further hardens the same hosted profile. Ordinary out-of-band adoption must preserve the last accepted canonical revision in ancestry so committed Git/idempotency evidence is not erased; directly observed private repository visibility becomes a live hosted-write eligibility condition; `PUBLISHING` remains fenced until all issued publisher capabilities for the generation are quiesced; and the managed memory validation-workflow trigger transition must ship through the released upgrader/downstream migration rather than ordinary provider writes. Where older ADR-014/ADR-016 recovery/publication wording is less strict, ADR-017 controls.

## ADR format

Each ADR contains:

```text
# ADR-NNN: title

Status: Proposed | Accepted | Superseded | Rejected
Date: YYYY-MM-DD

## Context
What problem or constraint requires a durable decision?

## Decision
What are we choosing?

## Consequences
What becomes easier, harder, required, or intentionally unsupported?

## Alternatives considered
What credible alternatives were rejected and why?

## Verification
How can implementation/tests demonstrate compliance?
```

## Rules

1. ADRs record architectural decisions, not ordinary implementation details.
2. Accepted ADRs should not be silently edited to reverse their meaning; supersede them with a new ADR.
3. Compatibility- or security-affecting code changes should cite the relevant ADR in the PR/commit description.
4. Released Runethread contracts remain immutable historical authority for repositories pinned to them until an explicit supported migration changes that repository's contract state.