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

Goal: make normal hosted external memory delivery fast, deterministic, race-safe, auditable, and ready for later MCP transport without turning GitHub Actions into a temporary primary execution architecture.

Governing decisions:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue;
- ADR-014 — Cloud-hosted Phase 2.6 memory-delivery control plane.

Architecture:

```text
semantic caller
      |
authenticated sealed mutation request
      |
Cloudflare Worker/API edge
      |
per-repository Durable Object coordinator
      |
durable operation Workflow
      |
Runethread finalizer Container
      |
private immutable exact candidate
      |
fresh reduced-privilege audit Container
      |
least-privilege GitHub App exact CAS publication
      |
user-owned audited canonical GitHub main
```

Primary work:

- authenticated transport-neutral hosted request/status/cancel boundary for one complete MemoryService-compatible operation;
- one Durable Object per immutable GitHub repository identity for hosted lane admission/serialization, operation status, suspension/maintenance/reconciliation state, and last accepted canonical revision;
- one Cloudflare Workflow per admitted operation for durable idempotent execution checkpoints and retries;
- Containers running the real Linux/Go/Git Runethread runtime, keeping Core itself provider-independent;
- one cold GitHub clone maximum in the normal finalization path, preserving reachable commit history required for idempotency while avoiding unnecessary historical blob transfer where safe;
- optional warm-clone reuse only as a disposable cache after exact fetch/clean refresh;
- finalization through the existing deterministic `MemoryService.ApplyMutation`, with its existing Index v2 write, hard validation, commit creation, and local-only fast-forward performed once;
- early `ALREADY_COMMITTED`, `NO_OP`, and stale `NEEDS_REPREPARE` resolution before expensive candidate/audit work;
- private immutable/content-addressed candidate packaging bound to expected revision, exact candidate commit/tree, request fingerprint, mutation metadata, runtime/build identity, and pinned contract identity;
- artifact/package design proven self-contained even when the finalizer clone uses partial-clone/promisor mechanics;
- fresh reduced-privilege artifact-based audit of the exact candidate without another GitHub clone, repair step, or publisher credential;
- least-privilege Runethread GitHub App installed only on user-authorized memory repositories, with long-lived App private-key material held only in the hosted secret boundary;
- atomic expected-old-SHA, non-force publication to exact audited candidate;
- early prototype of clone-free Git-object publication, accepted only if exact candidate object identity is reproduced; otherwise exact audited-artifact push fallback;
- cheap post-CAS `main == candidate` confirmation instead of another full synchronous clone/validation cycle;
- one normal hosted mutation architecture for GitHub Free and paid private users, with paid branch/ruleset protection as optional defense-in-depth;
- fail-closed reconciliation after unexpected/out-of-band canonical movement;
- explicit operation outcomes including `NO_OP`, `ALREADY_COMMITTED`, stale reprepare, finalization/audit/cancellation/reconciliation states, plus lane `OPEN`/`SUSPENDED`/`MAINTENANCE` semantics;
- control-plane barriers for contract/schema/trust/repository-format/bootstrap/delivery/migration changes;
- project current-state/overview prose treated as orientation/materialized views rather than a required atomic-memory dual write;
- GitHub Actions retained only where it proves a distinct independent repository-health, recovery, migration, or control-plane invariant;
- end-to-end latency/cost measurement split across cold/warm clone, finalization, packaging, audit, publication, and provider startup.

Phase 2.6 v1 deliberately uses singleton operations. Deferred work includes semantic dependency quantification, neighboring-operation batching/coalescing, automatic semantic re-preparation, general AI task orchestration, and project-orientation projection refresh beyond current correctness needs.

Exit criteria are tracked in issue #20. At minimum:

- a remote semantic caller can submit one authenticated sealed MemoryService mutation without directly editing canonical memory files or receiving GitHub write credentials;
- private GitHub Free and protected paid repositories use the same normal hosted mutation path;
- per-repository hosted publication remains single-writer while independent repositories can progress independently;
- Workflow retry/resume preserves operation/idempotency identity and cannot duplicate canonical mutation;
- a cold normal finalizer performs no more than one GitHub clone and preserves historical idempotency evidence;
- existing MemoryService constructs exact candidate `C`, writes Index v2 once, hard-validates, and leaves remote `main` at expected `H0` before audit;
- a private self-contained candidate package lets a fresh reduced-privilege auditor verify exact `C` without another GitHub clone;
- audit failure leaves canonical state unchanged and suspends later hosted writes;
- publication fails closed if canonical `main` moved and otherwise makes exact audited `C` canonical without force/merge/rewrite;
- clone-free Git-object publication is enabled only if exact object identity is proven, with exact candidate push as the fallback;
- normal completion performs only cheap exact-ref confirmation after the exact audited publication;
- exact committed retries, no-ops, stale work, cancellation, crash recovery, audit disagreement, out-of-band movement, and control-plane barriers are deterministic and tested;
- project orientation files are not silently dual-written by provider/workflow code;
- local/offline MemoryService remains usable without Cloudflare and Core contains no hosted-provider dependency;
- GitHub Actions is removed from the normal interactive write critical path;
- the implementation is rolled through the proper release/downstream process and proven against a real private memory repository;
- measured latency/cost demonstrates that redundant clones/checks have been removed without removing distinct trust boundaries.

The hosted Runethread infrastructure account requires Cloudflare Workers Paid because Containers are needed for the real Linux/Go/Git runtime. This is not an end-user Cloudflare-plan requirement.

**Phase 3 is blocked until Phase 2.6 exits green.**

## Phase 3 — MCP adapters over the established memory boundary

Goal: expose the already-proven local and hosted memory operations to compatible AI clients without duplicating business logic.

Phase 3 begins only after the Phase 2.6 Memory Write Delivery Pipeline is implemented and independently verified. MCP is transport over the established MemoryService/delivery semantics; it does not become a second owner of memory mutation, queueing, auditing, concurrency, or Git publication behavior.

Architecture:

```text
local client                 remote client
    |                            |
MCP stdio/local            MCP authenticated HTTP
    |                            |
MemoryService              Runethread hosted delivery API
    |                            |
local Git                    MemoryService/Core -> user GitHub
```

Initial memory tools should remain small:

```text
memory.search
memory.get
memory.prepare
memory.apply
memory.withdraw
memory.status
```

Start with the smallest transports required to prove both local/offline use and hosted remote memory delivery. Re-check the current MCP protocol/SDK and authentication requirements when Phase 3 becomes current rather than freezing today's transport details into Phase 2.6.

Exit criteria:

- a compatible external AI client can retrieve and mutate memory through Core without understanding repository layout;
- local clients can use Runethread without Cloudflare when local execution/Git are available;
- remote clients can use the same Phase 2.6 hosted mutation lifecycle without receiving canonical GitHub write credentials;
- MCP-specific code contains transport translation, not repository/business/queue/audit rules;
- changing MCP client/protocol integration does not change canonical memory format or Phase 2.6 publication invariants.

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

## Phase 8 — broader remote project/task integration

Goal: extend the already-established remote memory/MCP surface to project/task operations where product/platform support permits it.

Expose authenticated remote capabilities for the later Orchestrator/project layers, such as:

```text
project.*
task.*
```

Memory `memory.*` remote transport is no longer deferred to this phase; it is established by Phase 2.6 + Phase 3. This phase is about broader project/task integration and ordinary-chat entry points after the Orchestrator exists.

Exit criteria:

- remote project/task access uses explicit authentication and least privilege;
- writes retain the same Core/Orchestrator invariants as local calls;
- changing ChatGPT/other-client integration APIs does not require changing canonical storage or the Phase 2.6 memory-delivery architecture.

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

- notifications/webhooks;
- collaboration/team policies;
- richer UI;
- multiple parallel workers;
- additional provider adapters;
- remote synchronization conveniences;
- alternative hosted execution providers;
- PostgreSQL/additional queue infrastructure only when measured hosted scale requires it;
- container orchestration/Kubernetes only when measured operational requirements justify it.

## Immediate next engineering milestone

The immediate milestone is **Phase 2.6 — Memory Write Delivery Pipeline**.

> Build and verify the Cloudflare-hosted, audited, per-repository mutation-delivery path defined by ADR-012, ADR-013, and ADR-014 so a remote caller can submit a sealed MemoryService mutation, receive durable asynchronous operation state, and make only an independently audited exact candidate canonical through race-safe GitHub publication with no redundant clone/validation ceremony.

Only after Phase 2.6 exits green should Phase 3 MCP implementation begin.
