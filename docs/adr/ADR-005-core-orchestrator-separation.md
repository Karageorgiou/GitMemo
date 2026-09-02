# ADR-005: Core and Orchestrator are separate components and repositories

Status: **Accepted**
Date: 2026-09-02

## Context

Runethread has two fundamentally different responsibilities: preserving/verifying durable user memory and coordinating nondeterministic agent execution. Combining them would couple repository integrity to provider APIs, task runtimes, and high-churn operational concerns.

## Decision

Runethread Core and Runethread Orchestrator are separate components with separate repositories and release lifecycles.

### Runethread Core
Owns deterministic memory repository behavior: initialization, trust verification, parsing, schema enforcement, validation, indexing, search primitives, memory mutation transactions, lifecycle/relationship invariants, compatibility, and local interfaces.

### Runethread Orchestrator
Owns project registration, task lifecycle, worker selection, worktree management, approvals, provider/thread metadata, execution events, result normalization, and runtime persistence.

The Orchestrator consumes Core through stable application/tool boundaries. It must not reimplement Core repository rules or become canonical storage for personal memory.

## Consequences

- Core remains useful without Orchestrator or any AI provider.
- Orchestrator can evolve quickly with changing agent ecosystems without destabilizing memory format/trust.
- Integration contracts between the two components must be explicit and versioned when necessary.
- Deployment can run both locally while retaining logical separation.

## Alternatives considered

### Single monolithic repository/service
Rejected because provider/runtime churn would become entangled with the durable-memory compatibility surface.

### Put orchestration inside the memory repository protocol
Rejected because transient task execution is not canonical long-term memory.

## Verification

1. Core builds/tests without Orchestrator dependencies.
2. Orchestrator does not directly edit canonical memory file formats.
3. Orchestrator provider dependencies do not appear in Core.
4. failure/unavailability of Orchestrator does not prevent local validation/search/upgrade of a memory repository.
