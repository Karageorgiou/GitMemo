# ADR-001: Canonical state ownership

Status: **Accepted**
Date: 2026-09-02

## Context

Runethread combines personal continuity memory, project repositories, agent execution, generated indexes, and optional conversation metadata. If the same durable fact can become authoritative in several places, conflicting copies will eventually diverge and the system will no longer know which value is true.

## Decision

Every class of durable state has exactly one canonical owner.

- Personal semantic memories, provenance, lifecycle, and memory relationships are canonical in the user-owned memory Git repository.
- Project source code, build configuration, project architecture, and project ADRs are canonical in the project Git repository.
- Task execution state, worker thread identifiers, approvals, retries, worktrees, and operational event history are canonical in the Orchestrator runtime store.
- Generated indexes, caches, summaries, and context packets are derived views and are never authoritative.
- Chat URLs and provider-specific conversation identifiers are optional navigation/provenance metadata, never continuity state.

Derived views SHOULD identify or be able to resolve their canonical source whenever practical. A memory may summarize or point to a project decision, but it must not silently become a competing source of project truth.

## Consequences

- Conflicting copies have a deterministic resolution rule: consult the canonical owner.
- Project truth stays with the project and personal continuity follows the user.
- Context packets may be regenerated without data loss.
- Cross-system references and provenance become important because duplication is intentionally limited.
- The Orchestrator database may be lost without destroying canonical memory or project code, although operational history may be lost.

## Alternatives considered

### Store all state in the memory repository
Rejected because source code, project ADRs, and high-churn execution state have different ownership, lifecycle, and performance requirements.

### Store all state in an Orchestrator database
Rejected because it would centralize user memory and project truth in an operational service and weaken local/user ownership.

### Allow duplicated authoritative copies
Rejected because reconciliation becomes ambiguous and eventually produces contradictory truth.

## Verification

Implementation and documentation comply when:

1. each persisted state type has a documented canonical owner;
2. derived indexes can be rebuilt from canonical data;
3. project architecture is referenced rather than copied as an independent authority into memory;
4. deleting Orchestrator runtime state does not delete canonical memory or project Git history;
5. context aggregation identifies source locations for durable project facts where practical.
