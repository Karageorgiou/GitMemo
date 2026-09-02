# ADR-006: Capability-based worker abstraction

Status: **Accepted**
Date: 2026-09-02

## Context

Runethread must coordinate different execution backends such as Codex, general research agents, and future providers. Routing directly by product/brand name would hard-code current vendors into the architecture and make policy/testing brittle.

## Decision

Workers are selected through capabilities and policy, not brand names alone.

A task describes required capabilities such as:

```text
repo_read
repo_write
shell
tests
web
files
long_running
```

Each worker adapter advertises supported capabilities. A deterministic router selects a compatible worker subject to policy and approvals.

Initial worker classes:

- `codex` for repository mutation, shell/build/test/debugging work;
- `general` for substantial research, file/document analysis, and long-running non-code work;
- `chat` as a no-delegation result when the active conversational frontend can handle the task directly.

A model may help interpret ambiguous task requirements later, but capability compatibility and permission policy remain deterministic.

## Consequences

- Providers can be added/replaced without changing task semantics.
- Routing decisions become inspectable and testable.
- Task requests must express execution requirements explicitly enough for deterministic policy.
- Provider-specific thread IDs remain adapter/runtime metadata.

## Alternatives considered

### Route directly to named products
Rejected because it couples architecture and stored tasks to current vendors.

### Fully model-driven routing
Rejected for the baseline because it makes routing nondeterministic and harder to test/audit.

### One universal worker
Rejected because coding, web research, and conversational reasoning have different privilege and isolation needs.

## Verification

1. router tests select workers from capability sets without inspecting provider marketing names;
2. unsupported capability requests fail clearly rather than silently degrading;
3. provider adapters can be replaced behind the same task contract;
4. policy checks are deterministic and testable without model calls.
