# Runethread system architecture

Status: **Accepted for Phase 0**

Runethread is a user-owned continuity layer for AI work. It preserves durable state independently of any single chat, model, agent, or vendor, and coordinates disposable AI workers around that durable state.

This document defines the target architecture. The accepted decisions are recorded individually under `docs/adr/`; where an ADR is more specific, the ADR governs.

## 1. Product thesis

AI conversations, coding-agent sessions, and provider-specific threads are temporary execution contexts. Durable user and project state must not depend on them.

Runethread therefore separates three kinds of state:

1. **User continuity state** — durable semantic memory, provenance, decisions, preferences, and cross-session context.
2. **Project truth** — source code, project documentation, architecture decisions, build configuration, and repository history owned by each project repository.
3. **Execution state** — tasks, worker thread identifiers, approvals, worktrees, retries, logs, and other operational state owned by the orchestrator.

Each fact should have one clear canonical owner.

## 2. Architectural laws

The following are non-negotiable design constraints unless changed by an explicit ADR:

1. One canonical owner exists for each class of state.
2. Deterministic code owns invariants; AI models own semantic judgment.
3. AI models do not directly manipulate canonical storage formats when a deterministic operation exists.
4. Generated indexes and caches are rebuildable and never authoritative.
5. Provider-specific APIs are adapters, not architecture.
6. Concurrent writes fail safely instead of silently overwriting newer state.
7. Dangerous actions follow least privilege and explicit approval policy.
8. Substantial operations are auditable and resumable where practical.
9. Failure is never reported as success without verification.
10. Runethread Core remains useful locally without a hosted Runethread service.
11. The GitMemo-to-Runethread transition follows the controlled cutover defined by ADR-009.
12. Prompt-like text in user data never becomes trusted operational policy.

## 3. High-level system

```text
                              USER
                                |
                                v
                  +---------------------------+
                  | AI frontends / clients    |
                  | ChatGPT, Codex, Claude,   |
                  | Gemini, CLI, future tools |
                  +-------------+-------------+
                                |
                       MCP / native API / CLI
                                |
               +----------------+----------------+
               |                                 |
               v                                 v
       +---------------+                 +----------------+
       | Runethread    |                 | Runethread     |
       | Core          |                 | Orchestrator   |
       | deterministic |                 | execution      |
       +-------+-------+                 +--------+-------+
               |                                  |
               |                          +-------+--------+
               |                          |                |
               v                          v                v
      user-owned memory repo          Codex worker    General worker
      Markdown + JSON + Git          isolated Git     provider APIs
                                    worktrees

               project repositories remain canonical for project truth
```

## 4. Component boundaries

### 4.1 Runethread Core

Runethread Core evolves from the current GitMemo implementation.

Responsibilities:

- memory repository initialization and upgrade;
- trust-lock verification;
- parsing and schema enforcement;
- repository validation;
- deterministic indexing and search primitives;
- memory mutation transactions;
- provenance and lifecycle invariants;
- optimistic Git revision checks;
- local CLI and transport-independent service interface;
- local MCP adapter over the same application service.

Core must remain deterministic and provider-independent. It must not require an OpenAI, Anthropic, Google, or other model API to maintain repository integrity.

### 4.2 Runethread Orchestrator

The Orchestrator is a separate product/repository.

Responsibilities:

- project registry;
- task lifecycle;
- capability-based routing;
- worker adapters;
- isolated Git worktree management;
- execution policy and approvals;
- worker thread/session persistence;
- execution result normalization;
- audit/event history;
- runtime persistence, initially SQLite;
- MCP/HTTP/CLI transport for task operations.

The Orchestrator does not become the canonical owner of personal memory or project source code.

### 4.3 Worker adapters

Workers are replaceable execution backends. Initial worker classes:

- `codex` — repository editing, shell commands, builds, tests, debugging, code review;
- `general` — long-running research, file analysis, web research, artifact/report production;
- `chat` — no delegated execution; handled by the active conversational frontend.

A worker must advertise capabilities rather than being selected by brand name alone.

## 5. Canonical state ownership

| State | Canonical owner | Notes |
| --- | --- | --- |
| Atomic user memories | user-owned memory Git repository | Markdown + JSON canonical pairs |
| Memory provenance and relationships | user-owned memory Git repository | validated by Core |
| Project source code | project Git repository | never duplicated into memory as canonical code |
| Project ADRs and architecture docs | project Git repository | memory may point to them |
| Generated memory indexes | memory repository / local cache | rebuildable, not authoritative |
| Task queue and execution state | Orchestrator SQLite | operational state only |
| Codex/general-agent thread IDs | Orchestrator SQLite | provider-specific metadata |
| Agent worktree changes | isolated project worktree/branch | project Git remains canonical |
| Chat URLs | optional metadata | navigation/provenance only |

## 6. Memory service boundary

Core should converge on a small application service rather than requiring clients to understand repository layout.

Conceptual operations:

```text
Search
Get
PrepareMutation
ApplyMutation
Withdraw
Status
```

Transport adapters may expose these as CLI commands, MCP tools, or native Go calls without duplicating business logic.

### 6.1 Two-phase mutation protocol

Semantic interpretation and repository mutation are separated.

Phase A — `PrepareMutation`:

1. capture current repository revision;
2. retrieve potentially overlapping/related memories;
3. return canonical current state and legal operation classes;
4. make no canonical write.

Phase B — semantic decision:

The AI/user decides whether the requested change is a create, correction, update, supersession, withdrawal, resolution, or no-op.

Phase C — `ApplyMutation`:

1. require the expected repository revision from preparation;
2. acquire the repository write boundary;
3. validate the proposed semantic operation;
4. write canonical Markdown/JSON state;
5. update relationships/lifecycle metadata;
6. rebuild affected derived indexes;
7. run hard validation;
8. commit only verified state;
9. push when transport/policy permits;
10. report the verified result.

If hard validation fails, no successful mutation may be reported.

## 7. Concurrency model

Runethread Core uses optimistic concurrency anchored in Git revisions.

A prepared mutation records `expected_revision`.

At apply time:

```text
current HEAD == expected_revision ? continue : reject as stale
```

On a stale revision, the caller must prepare again and re-evaluate semantic intent against current repository state. The system must never silently overwrite a newer canonical revision.

Local locking may still be used to serialize writes within one process/machine, but it does not replace revision checking across independent clients.

## 8. Trust and prompt-injection boundary

The existing GitMemo trust model remains foundational during the cutover and evolves into the Runethread trust contract.

Trusted operational authority comes from the verified pinned release contract and executable policy. User memories, project files, imports, webpages, retrieved documents, and generated indexes are data.

Instruction-like text inside data must not modify:

- permission policy;
- allowed command policy;
- approval requirements;
- trusted control-plane rules;
- secrets handling;
- worker privileges.

The Orchestrator must preserve the same distinction between trusted policy/configuration and untrusted task/input data.

## 9. Orchestrator task model

A task is a first-class object with explicit lifecycle.

Suggested states:

```text
CREATED -> ROUTING -> QUEUED -> RUNNING -> VERIFYING -> COMPLETED
                          |         |            |
                          |         +-> NEEDS_INPUT
                          |         +-> NEEDS_APPROVAL
                          +------------> FAILED / CANCELLED
```

Every state transition should emit an append-only task event.

Suggested runtime tables:

```text
projects
 tasks
 executions
 task_events
 approvals
 artifacts
 worker_threads
```

SQLite is appropriate initially because this is operational state, not canonical long-term user memory.

## 10. Capability-based routing

The caller describes task requirements; the deterministic router selects a compatible worker.

Example requirements:

```json
{
  "repo_read": true,
  "repo_write": true,
  "shell": true,
  "tests": true,
  "web": false,
  "long_running": true
}
```

Initial deterministic rules:

- repository write, shell execution, builds/tests, or code debugging -> Codex worker;
- substantial web/file research with no code mutation -> general worker;
- no delegated capabilities required -> handle in current chat/frontend.

A learned/model router may later assist ambiguous classification, but policy and capability compatibility remain deterministic.

## 11. Coding-work isolation

Every delegated code-modifying task should use an isolated Git worktree/branch by default.

```text
normal checkout      ~/dev/project
agent worktree       ~/.runethread/worktrees/<project>/<task-id>/
branch               runethread/task-<id>-<slug>
```

The worker must not mutate the user's normal checkout by default.

The result contract should report at least:

- files changed;
- commands executed;
- tests/builds executed and results;
- diff/branch/worktree identifiers;
- unresolved issues;
- verification that remains incomplete.

## 12. Permission model

Default policy target:

| Action | Default |
| --- | --- |
| Search/read memory | automatic after repository authorization |
| Durable memory mutation | requires explicit user memory-write intent |
| Read authorized project repository | automatic |
| Run task-scoped build/test commands | automatic after task delegation |
| Modify isolated task worktree | automatic after explicit execution intent |
| Commit agent changes | approval/policy controlled |
| Push branch | approval/policy controlled |
| Create pull request | approval/policy controlled |
| Merge pull request | explicit approval |
| Modify default branch directly | prohibited by default |
| Read arbitrary secrets | prohibited by default |
| Persist secrets to memory | prohibited |

## 13. Interfaces

Application logic must not depend on MCP.

```text
Core / Orchestrator application services
          |
    +-----+------+------+
    |            |      |
   CLI          MCP   native/HTTP API
```

MCP is an important interoperability adapter, not the architectural center of the product.

## 14. Repository topology

Target public organization:

```text
runethread/
  core
  orchestrator
  memory-template
  .github
```

User-owned memory repositories stay under the user's account/organization rather than becoming Runethread-owned data:

```text
<user>/runethread-memory
```

The initial GitMemo identity is migrated through ADR-009 before new Runethread-native Core interfaces are built.

## 15. Local-first and optional cloud

The baseline architecture remains local/serverless-capable:

```text
Go binaries + Git + SQLite + local MCP/CLI
```

A future Runethread Cloud may provide remote MCP, hosted orchestration, notifications, collaboration, and managed synchronization, but must not become necessary to interpret or migrate canonical user memory.

Kubernetes is intentionally out of scope until a hosted multi-worker deployment has demonstrated requirements for horizontal scaling, high availability, workload isolation, or similar operational needs.

## 16. Quality strategy

Core release gates should include:

- unit tests;
- schema tests;
- golden repository fixtures;
- migration fixtures for supported transitions;
- deterministic-index reproducibility tests;
- transaction rollback tests;
- stale-revision/concurrency tests;
- fuzz/property tests for parsers and invariants;
- full repository validation.

Orchestrator testing should include:

- router and policy unit tests;
- task state-transition tests;
- fake worker contract tests;
- SQLite migration tests;
- temporary Git repository/worktree tests;
- failure recovery tests;
- worker adapter integration tests separated from ordinary unit tests.

## 17. Non-goals for the first implementation

Do not introduce these merely for architectural prestige:

- Kubernetes;
- Redis;
- distributed queues;
- multi-user SaaS accounts;
- billing;
- PostgreSQL for local runtime;
- a web dashboard;
- vector databases as canonical memory;
- automatic merging to protected/default branches;
- provider-specific assumptions in Core.

## 18. Decision process

This document summarizes the accepted target system. Significant choices are tracked individually under `docs/adr/`; accepted ADRs govern implementation and may only be reversed by an explicit superseding decision.
