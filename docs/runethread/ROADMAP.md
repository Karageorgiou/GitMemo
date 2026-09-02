# Runethread implementation roadmap

Status: **Proposed**

This roadmap sequences the work so each stage proves one architectural assumption before adding another dependency.

## Phase 0 — architecture freeze

Goal: define boundaries before implementation changes.

Deliverables:

- system architecture;
- GitMemo -> Runethread migration strategy;
- ADR catalog for canonical ownership, deterministic Core, two-phase mutation, concurrency, orchestration, worker abstraction, worktree isolation, permissions, migration compatibility, and repository topology;
- explicit non-goals;
- security/trust boundary carried forward from GitMemo.

Exit criteria:

- architecture is reviewable as documentation-only changes;
- no code/module/repository-format rename has occurred;
- unresolved architecture questions are recorded rather than hidden in implementation.

## Phase 1 — deterministic Core mutation service

Goal: make the current memory engine usable through a small deterministic application boundary.

Primary work:

```text
MemoryService
  Search
  Get
  PrepareMutation
  ApplyMutation
  Withdraw
  Status
```

Supporting work:

- repository abstraction;
- expected Git revision in prepared mutations;
- local write serialization where needed;
- mutation transaction staging/rollback;
- validator integration as a hard write gate;
- relationship/lifecycle invariant enforcement;
- machine-readable result/error contracts;
- JSON CLI output suitable for automation.

Suggested CLI evolution while still under GitMemo compatibility:

```text
gitmemo get --json
gitmemo prepare --json
gitmemo apply --json
gitmemo withdraw --json
gitmemo status --json
```

Exit criteria:

- an AI/tool no longer needs to know how canonical Markdown/JSON/index files are physically edited;
- stale-revision writes are rejected;
- a failed hard validation cannot produce a successful commit/result;
- tests verify rollback and concurrency failure modes.

## Phase 2 — local MCP adapter

Goal: expose Core operations to compatible AI clients without duplicating business logic.

Architecture:

```text
MCP adapter
    |
MemoryService
```

Initial tools should remain small:

```text
memory.search
memory.get
memory.prepare
memory.apply
memory.withdraw
memory.status
```

Start with local/stdio transport where supported.

Exit criteria:

- a compatible external AI client can retrieve and mutate memory through Core without understanding repository layout;
- MCP-specific code contains transport translation, not repository business rules.

## Phase 3 — Orchestrator skeleton

Goal: prove the execution control plane without paying for or depending on real model workers.

Create separate repository:

```text
runethread/orchestrator
```

Implement:

- project registry;
- task model/state machine;
- SQLite persistence and migrations;
- append-only task events;
- capability router;
- policy/approval model;
- CLI;
- fake worker implementation.

Suggested CLI:

```text
runethread project add
runethread task dispatch
runethread task status
runethread task result
runethread task cancel
```

Exit criteria:

- fake tasks traverse the full lifecycle deterministically;
- restart/resume behavior preserves task state;
- invalid state transitions are rejected;
- no AI provider is required for tests.

## Phase 4 — Codex worker

Goal: prove safe delegated code execution.

Implement:

- Codex App Server adapter;
- per-task Git worktree creation;
- task branch naming;
- streamed worker events;
- approval bridge;
- cancellation;
- normalized result contract;
- build/test verification capture.

Target proof:

```text
CLI
 -> Orchestrator
 -> capability router
 -> isolated worktree
 -> Codex
 -> build/tests
 -> structured result
 -> SQLite task history
```

Exit criteria:

- Codex cannot modify the user's normal checkout by default;
- changed files, commands, test results, branch/worktree, and unresolved verification are reported;
- failed execution is distinguishable from failed verification;
- task cancellation and approval pauses are tested.

## Phase 5 — general agent worker

Goal: support substantial non-coding delegated work.

Implement a provider adapter initially using an appropriate OpenAI agent/Responses surface, while keeping the worker interface provider-independent.

Capabilities may include:

- web research;
- file/document analysis;
- long-running reasoning;
- structured reports/artifacts;
- project-repository read access without automatic mutation.

Exit criteria:

- router can distinguish `chat`, `codex`, and `general` capability sets;
- provider-specific conversation/thread identifiers remain confined to adapter/runtime metadata;
- paid API usage can be disabled without breaking Core or Codex/local workflows.

## Phase 6 — project context aggregation

Goal: make chat/session replacement cheap and reliable.

Introduce a project-context operation that assembles:

```text
relevant Runethread memory
+ current project repository metadata
+ AGENTS.md / project instructions
+ project ADR pointers
+ active tasks
+ recent completed tasks
+ blockers / next actions
```

The result is an execution/context packet, not a new canonical database.

Exit criteria:

- a fresh AI conversation can recover current project continuity without depending on the previous chat URL;
- project source truth is referenced rather than duplicated into personal memory;
- stale task/runtime metadata is clearly distinguished from canonical project Git state.

## Phase 7 — remote MCP / normal-chat integration

Goal: make ordinary AI chat the front door where product/platform support permits it.

Expose a remote authenticated surface for:

```text
memory.*
project.*
task.*
```

This phase intentionally comes late because consumer AI integration capabilities and plan restrictions can change without affecting Core/Orchestrator architecture.

Exit criteria:

- remote access uses explicit authentication and least privilege;
- writes retain the same Core/Orchestrator invariants as local calls;
- changing ChatGPT/other-client integration APIs does not require changing canonical storage.

## Phase 8 — public Runethread migration

Goal: move from GitMemo identity to Runethread without breaking supported repositories.

Work includes:

- bridge GitMemo release;
- historical compatibility fixtures;
- transfer/preservation of implementation Git history;
- `runethread/core` establishment;
- new CLI/module identity;
- new generated memory template;
- documentation/domain transition;
- legacy validation and upgrade support.

Exit criteria:

- new users install Runethread directly;
- existing GitMemo repositories remain supported;
- historical GitMemo releases/tags remain immutable and meaningful;
- compatibility tests cover every officially supported legacy repository format.

## Later, only when justified

Possible future work:

- hosted Runethread Cloud;
- notifications/webhooks;
- collaboration/team policies;
- richer UI;
- multiple parallel workers;
- additional provider adapters;
- remote synchronization conveniences;
- PostgreSQL/queues for hosted multi-instance runtime;
- container orchestration/Kubernetes only when measured operational requirements justify it.

## Immediate next engineering milestone

After Phase 0 review, the first code milestone should be:

> A deterministic `MemoryService` can prepare and apply a memory mutation against an expected Git revision, validate the final repository, and reject/roll back unsafe or stale writes without an AI model participating in repository integrity.
