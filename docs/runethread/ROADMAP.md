# Runethread implementation roadmap

Status: **Proposed**

This roadmap sequences the work so identity is made coherent while the project is still pre-v1, then each engineering stage proves one architectural assumption before adding another dependency. `CURRENT_MILESTONE.md` remains the authority for the immediate work boundary.

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

## Phase 2.5 — compatibility hardening and release separation

Goal: harden repository/runtime compatibility before adding new transport or delivery surfaces.

Primary work:

- explicit runtime-release versus contract-release identity;
- contract v8 compatibility matrix and migration rules;
- filesystem/repository-boundary safety hardening;
- deterministic upgrade behavior across supported source states;
- public template and known private-memory migration to the v0.8 contract state;
- engineering-process and cross-platform pipeline hardening.

Exit criteria:

- Runethread v0.8.0 is published as an immutable release;
- contract v8 semantics and release separation are explicit and tested;
- the public template and known private memory repository are migrated and permanently validated;
- historical compatibility and migration evidence are retained.

Status: **Complete**.

## Phase 2.6 — Memory Write Delivery Pipeline

Goal: make normal external memory delivery fast, deterministic, race-safe, and auditable before introducing MCP transport.

Governing decisions:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue.

Architecture:

```text
semantic caller
      |
sealed mutation request
      |
per-repository logical delivery queue
      |
MemoryService finalizer
      |
noncanonical exact candidate
      |
fresh read-only audit
      |
exact expected-revision fast-forward publication
      |
audited canonical main
```

Primary work:

- sealed GitHub workflow-dispatch request carrying one complete MemoryService-compatible operation;
- GitHub Actions as the first replaceable execution adapter, not the architectural queue authority;
- finalization through the existing deterministic MemoryService/Core implementation;
- candidate construction off canonical `main` without premature publication;
- Index v2 regeneration, hard validation, and strict freshness before publication;
- fresh read-only audit of the exact candidate revision/tree;
- exact Git-SHA compare-and-swap fast-forward publication with no force push or intervening merge commit;
- dedicated least-privilege Runethread GitHub App installed only on user-authorized memory repositories as the canonical publisher;
- managed memory-repository policy that prevents routine writers from bypassing the audited path;
- operation state covering queue/finalization/audit/publication/committed outcomes;
- stale-operation parking as `NEEDS_REPREPARE` without automatic semantic rebasing or reinterpretation;
- idempotent crash/lost-response retry using the existing operation identity;
- cancellation before publication and exact outcome resolution once publication begins;
- repository write-lane suspension on audit disagreement or red independent audit;
- exclusive barriers for contract/schema/trust/repository-format/bootstrap/workflow/migration changes;
- rollout through `runethread/memory-template` and then a real private memory repository;
- end-to-end latency measurement against the previous duplicated synchronous workflow.

Phase 2.6 v1 deliberately uses singleton operations. Deferred work includes semantic dependency quantification, neighboring-operation batching/coalescing, and automatic semantic re-preparation.

Exit criteria are tracked in issue #20. At minimum:

- a GitHub-only semantic caller can submit one sealed structured MemoryService mutation without directly editing canonical memory files;
- no half-written request race exists;
- unaudited candidates cannot become canonical;
- Index v2 is rebuilt and strict freshness passes;
- a fresh read-only auditor verifies the exact candidate and cannot publish it;
- publication fails closed if canonical `main` moved;
- successful publication makes the exact audited MemoryService candidate canonical;
- routine writers cannot bypass the managed audited publication path;
- stale operations park as `NEEDS_REPREPARE` without semantic auto-repair;
- crash/lost-response retry cannot duplicate a canonical mutation;
- audit disagreement suspends writes while safe canonical reads remain available;
- control-plane barriers invalidate incompatible queued semantic work;
- correctness is independent of GitHub workflow execution order;
- the delivery mechanism remains independent of MCP and does not make Core depend on GitHub internally;
- the GitHub execution adapter can later be replaced without a memory-format migration solely for queue state;
- template/private rollout is verified end-to-end;
- measured latency demonstrates removal of the previous redundant blocking ceremony without removing distinct correctness gates.

**Phase 3 is blocked until Phase 2.6 exits green.**

## Phase 3 — local MCP adapter

Goal: expose Core operations to compatible AI clients without duplicating business logic.

Phase 3 begins only after the Phase 2.6 Memory Write Delivery Pipeline is implemented and independently verified. MCP is transport over the established MemoryService/delivery semantics; it does not become a second owner of memory mutation, queueing, auditing, concurrency, or Git publication behavior.

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
- MCP-specific code contains transport translation, not repository business rules;
- the accepted Phase 2.6 delivery/concurrency invariants remain unchanged by transport.

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

The immediate milestone is **Phase 2.6 — Memory Write Delivery Pipeline**.

> Build and verify the audited, per-repository mutation-delivery path defined by ADR-012 and ADR-013 so a GitHub/cloud-only caller can submit a sealed MemoryService mutation, receive deterministic asynchronous operation state, and make only an independently audited exact candidate canonical through race-safe publication.

Only after Phase 2.6 exits green should Phase 3 MCP implementation begin.
