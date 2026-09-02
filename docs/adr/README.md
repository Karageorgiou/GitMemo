# Architecture Decision Records

Runethread uses Architecture Decision Records (ADRs) for durable architectural choices that affect compatibility, trust, security, state ownership, or component boundaries.

The Phase 0 architecture documents describe the proposed target system. The ADRs below should be accepted individually before implementation relies on them.

## Status values

- **Proposed** — under review; implementation must not assume it is final.
- **Accepted** — current architectural decision.
- **Superseded** — replaced by a newer ADR that explains why.
- **Rejected** — considered and intentionally not adopted.

## Phase 0 decision catalog

| ADR | Decision | Initial status |
| --- | --- | --- |
| ADR-001 | Canonical state ownership | Proposed |
| ADR-002 | Deterministic Core and semantic AI boundary | Proposed |
| ADR-003 | Two-phase memory mutation protocol | Proposed |
| ADR-004 | Git-revision optimistic concurrency | Proposed |
| ADR-005 | Core and Orchestrator are separate components/repositories | Proposed |
| ADR-006 | Capability-based worker abstraction | Proposed |
| ADR-007 | Isolated Git worktrees for delegated code mutation | Proposed |
| ADR-008 | Least-privilege permission and approval model | Proposed |
| ADR-009 | GitMemo-to-Runethread compatibility strategy | Proposed |
| ADR-010 | Runethread organization and repository topology | Proposed |

## ADR format

Each ADR should contain:

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
4. The current implementation remains governed by existing GitMemo release contracts until a Runethread migration decision is actually implemented and released.
