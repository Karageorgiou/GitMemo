# Runethread implementation roadmap

Status: **Proposed**

This roadmap sequences the work so identity is made coherent while the project is still pre-v1, then each engineering stage proves one architectural assumption before adding another dependency.

## Phase 0 — architecture freeze

Goal: define boundaries before implementation changes.

Deliverables:

- system architecture;
- clean GitMemo -> Runethread cutover strategy;
- ADR catalog for canonical ownership, deterministic Core, two-phase mutation, concurrency, orchestration, worker abstraction, worktree isolation, permissions, migration, and repository topology;
- explicit non-goals;
- security/trust boundary carried forward from GitMemo.

Exit criteria:

- architecture is reviewable as documentation-only changes;
- no code/module/repository-format rename has occurred yet;
- unresolved architecture questions are recorded rather than hidden in implementation;
- ADRs define the target Runethread-native identity and migration invariants.

## Phase 1 — controlled Runethread identity cutover

Goal: remove avoidable GitMemo naming before substantial new APIs and integrations are built.

Target identity:

```text
implementation       runethread/core
Go module            github.com/runethread/core
CLI                  runethread
native metadata      .runethread/
private repo naming  <user>/runethread-memory
first RT release     v0.6.0
```

Primary work:

- preserve Git history while transferring/renaming the implementation repository;
- update Go module/import identity;
- rename CLI entry point and user-facing commands;
- convert setup/bootstrap/release/template identity to Runethread;
- define the native `.runethread/` config/lock format;
- implement/test deterministic migration from the known GitMemo v0.5.0 source state;
- migrate the known private memory repository without replacing canonical UUID identities;
- change project slug/metadata from `gitmemo` to `runethread` where semantically appropriate;
- review the small project-memory set for product-name references rather than global search/replace;
- regenerate derived indexes;
- preserve historical GitMemo tags/releases/commits as immutable history;
- retain an explicit pre-migration recovery point.

Exit criteria:

- current implementation identity is consistently Runethread;
- `go test ./...` and all release gates pass under the new module identity;
- newly initialized repositories are Runethread-native;
- migrated private memory repository passes full validation;
- migrated memory count and UUID set match the pre-migration snapshot exactly;
- no unintended GitMemo operational identifiers remain in current native state;
- historical GitMemo releases remain reachable and unmodified.

## Phase 2 — deterministic Core mutation service

Goal: make the memory engine usable through a small deterministic application boundary under the final Runethread identity.

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

Suggested CLI:

```text
runethread get --json
runethread prepare --json
runethread apply --json
runethread withdraw --json
runethread status --json
```

Exit criteria:

- an AI/tool no longer needs to know how canonical Markdown/JSON/index files are physically edited;
- stale-revision writes are rejected;
- a failed hard validation cannot produce a successful commit/result;
- tests verify rollback and concurrency failure modes.

## Phase 3 — local MCP adapter

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

## Phase 4 — Orchestrator skeleton

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

## Phase 5 — Codex worker

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

## Phase 6 — general agent worker

Goal: support substantial non-coding delegated work.

Implement a provider adapter initially using an appropriate agent/API surface while keeping the worker interface provider-independent.

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

## Phase 7 — project context aggregation

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

## Phase 8 — remote MCP / normal-chat integration

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

## Phase 9 — hardening and public readiness

Goal: make the now-Runethread-native system safe for external users before compatibility becomes expensive.

Work includes:

- release regression fixtures beginning with the first public Runethread-native format;
- migration fixtures for the one historical GitMemo v0.5.0 cutover source state;
- broader fuzz/property coverage;
- cross-platform CI/release verification;
- threat-model review;
- installation/doctor UX;
- release signing/SBOM/dependency scanning where practical;
- public documentation under `runethread.dev`;
- organization/repository protection policy.

Exit criteria:

- new users install Runethread without requiring GitMemo knowledge;
- a supported release compatibility policy is stated for Runethread going forward;
- recovery and migration behavior is tested before external adoption raises the cost of breaking changes.

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

After Phase 0 ADR review, the next milestone is **not** a new memory API. It is the clean identity cutover:

> Preserve Git and canonical memory history while moving the implementation to `runethread/core`, converting current product/module/CLI/native-repository identifiers to Runethread, and deterministically migrating the known GitMemo v0.5.0 private memory repository with identical UUID membership and full post-migration validation.

Only after that cutover is verified should new Core APIs, MCP surfaces, and Orchestrator components be built.
