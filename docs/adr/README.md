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
