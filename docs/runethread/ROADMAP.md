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

Goal: make normal hosted external memory delivery fast, deterministic, race-safe, auditable, and ready for later MCP transport without turning GitHub Actions or another redundant orchestration layer into a temporary primary execution architecture.

Governing decisions:

- ADR-012 — audited candidate promotion for external memory delivery;
- ADR-013 — per-repository serialized mutation-delivery queue;
- ADR-014 — Cloud-hosted Phase 2.6 memory-delivery control plane;
- ADR-015 — project current-state documents become asynchronous orientation projections only in the next operational contract;
- ADR-016 — hosted contract eligibility, evidence capability/retention, and exact-publication hardening.

### Contract prerequisite

Contract v8 remains immutable and requires relevant project current-state synchronization as part of memory-write completion. Current `MemoryService.ApplyMutation` does not implement that project-view semantic write. Therefore, after the architecture-freeze gate passes, the first implementation slice is the ADR-015 contract transition (target contract v9), exact v8 fixture/migration/release verification, template migration, and private-repository migration.

Normal hosted Phase 2.6 mutation is admitted only after the repository is explicitly migrated to the projection-capable contract implementing ADR-015, or a later contract explicitly supported by the hosted release. Contract-v8 repositories may be read, verified, reconciled, and upgraded, but the normal hosted mutation path does not silently bypass the v8 synchronization rule.

Target topology:

```text
runethread/core
  deterministic MemoryService/CLI/runtime

runethread/hosted
  public API + webhook edge
  per-repository runtime Durable Objects + SQLite + alarms
  attached finalizer Containers
  private role-scoped evidence boundary/storage
  fresh auditor Container/DO contexts
  private GitHub App gateway
  minimal exact-Git publisher executor
```

`runethread/hosted` is the target provider-specific component unless implementation preflight finds a concrete topology/naming conflict. Hosted provider churn must not become a dependency of Core. Hosted releases pin immutable verified Core/runtime/container/protocol identities rather than floating public `main`.

Architecture:

```text
semantic caller
      |
authenticated sealed mutation request
      |
public Worker/API edge
      |
repository-runtime Durable Object
  SQLite lane/queue/phase-generation + alarm-driven drive()
      |
attached finalizer Container
      |
role-scoped immutable candidate/finalization evidence
      |
fresh reduced-privilege auditor
      |
role-scoped immutable audit evidence
      |
repository DO authorizes PUBLISHING
      |
private GitHub App gateway
      |
short-lived one-repo Contents-write token
      |
minimal Git publisher executor
      |
exact expected-old H0 -> exact audited C
      |
user-owned GitHub canonical branch ref
```

Primary work:

- authenticated request/status/cancel API plus explicit caller-to-App-installation repository authorization;
- explicit repository binding to immutable repository identity + App installation + canonical branch ref + last accepted revision, normally discovering the ref from default branch at adoption but never silently following later ref/default changes;
- private sealed request bodies stored as content-addressed/no-overwrite objects, with only opaque references/digests in DO state/logs/status;
- role-scoped evidence creation so admission/finalizer/auditor can create only their exact artifact class/key for the current repository/attempt/phase/generation under digest/size/create-if-absent constraints; finalizer cannot manufacture authoritative audit evidence and auditor cannot replace finalization evidence;
- referenced evidence remains live for queued/active/retrying/audited/publishing/reconciling operations; ordinary provider TTL/GC applies only after the operation no longer depends on it and the bounded recovery/incident window permits deletion;
- separate hosted attempt identity from Core idempotency identity; hosted identity binds repo/canonical-ref/request digest while Core owns semantic committed retry/conflict;
- one repository DO as sole hosted lane/operation-state/publication-authorization authority, with transactional SQLite for bounded queue, one active op, exact phase/execution generation, retry/backoff/deadline state, evidence refs, canonical-ref binding, suspension/maintenance/reconciliation;
- no Cloudflare Workflow in v1; one DO alarm is the at-least-once scheduler/recovery wakeup and the state driver explicitly persists/reschedules prolonged retryable failure;
- explicit async-interleaving protocol: local atomic phase+generation claim before every Container/object-store/GitHub external action, external I/O outside `blockConcurrencyWhile()`, and compare operation/phase/generation before accepting result; stale-generation output cannot advance state;
- durable acceptance handoff: return `ACCEPTED` only after request/operation metadata and a recoverable alarm are established; exact resubmission/status/recovery repairs missing alarms;
- Container/DO relationship used so repository runtime manages attached finalizer lifecycle rather than another coordinator;
- private internal GitHub gateway retains the long-lived App private key; public API, finalizer, auditor, evidence storage, and publisher executor never receive that key; ordinary runtime App baseline is Contents/Metadata without Administration/Workflows;
- full hosted operation serialized per repo while preserving ADR-003 committed-idempotency-before-stale ordering; stale work may require cold source preflight but stops before candidate/Index/package/audit once proven uncommitted;
- finalizer runs exact pinned Runethread Go/Core + Git; cold source target at most one clone/fetch with historical idempotency reachable, optional warm cache only after direct canonical-ref reset;
- hardened Git policy disabling/refusing hooks, recursive submodules, filters, unsafe config/includes, credential helpers, and similar repository-controlled execution;
- every fresh finalization starts from directly observed bound canonical ref/revision, never surviving unpromoted local candidate history;
- immutable generation-bound finalization receipt: candidate evidence first, receipt last, create-if-absent/no-overwrite, valid receipt wins retries while missing receipt causes restart from canonical state;
- finalization via existing `MemoryService.ApplyMutation`, preserving committed-retry-before-stale, one Index v2 write, hard validation, commit creation, local-only fast-forward;
- Core-validated `NO_OP` terminal path with no candidate/audit/publication;
- canonical repository/trust/compatibility/ref-binding failure separated from request-local mutation failure;
- candidate evidence bound to repo/canonical-ref/H0/C/tree/Core-idempotency/request fingerprint/hosted attempt+generation/runtime/container/delivery/contract identities/digests;
- candidate transport selected by measured exact bytes/runtime; prototype Git-native H0->C delta against self-contained package and prove completeness under partial-clone/promisor modes;
- fresh reduced-privilege audit of exact C with bounded exact-base acquisition, hard validation, strict Index v2 freshness, binding/scope checks, no repair/publication privilege;
- immutable generation-bound audit receipt, with deterministic audit failure durably suspending/reconciling lane before release;
- DO-only `AUDITED -> PUBLISHING` authorization after current generation, cancellation, lane, evidence, App/repo/canonical-ref auth, direct bound-ref==H0, and barrier checks;
- cancellation-vs-publication linearized by local atomic state transition; once PUBLISHING wins, cancellation is too late and exact Git outcome decides;
- only after durable `PUBLISHING`, private App gateway mints a short-lived one-repository minimum Contents-write installation token to a minimal trusted publisher executor;
- publisher executor receives only exact audited Git objects + H0 + C + bound ref + narrow token, performs no source clone/semantic mutation/repair/audit, never constructs C2, and attempts only an expected-old Git-protocol update from exact H0 to exact C;
- the currently documented GitHub REST reference update with `force=false` is not accepted as proof of exact expected-old CAS; clone-free/API publication may replace the publisher executor only after authoritative documentation and integration tests prove exact C identity and atomic old-ref==H0 semantics;
- ambiguous publication response resolved by bound-ref state: C = committed, H0 = retry same exact authorized publication, anything else = reconciliation;
- cheap post-CAS bound-ref==C confirmation, no redundant full validation;
- signed push webhooks only as hints; always reread bound canonical ref and never mutate accepted state from payload alone;
- proven uncommitted stale work -> `NEEDS_REPREPARE`; unexpected active-operation ref movement -> reconciliation;
- one hosted implementation for Free/paid private GitHub, with paid branch/ruleset protection optional defense-in-depth;
- explicit release/protocol versioning and non-atomic Worker/DO/finalizer/auditor/evidence/publisher/canonical-ref deployment handling using maintenance/drain or versioned blue/green;
- operation outcomes `NO_OP`, `ALREADY_COMMITTED`, `NEEDS_REPREPARE`, finalization/audit/cancel/reconciliation plus lane OPEN/SUSPENDED/MAINTENANCE;
- control-plane barriers for contract/schema/trust/repository-format/bootstrap/delivery/migration/evidence/publication/canonical-ref binding changes;
- request/rate/repository/artifact/Container/retry/operation-history/log/private-data retention limits and threat model;
- project current-state/overview prose treated as materialized orientation views only after the ADR-015 contract migration, not as an implicit contract-v8 relaxation;
- remove push-on-every-normal-memory full Actions validation through managed rollout, retaining only distinct health/recovery/migration/PR/control-plane checks;
- measure cold/warm source, bytes, idempotency/stale preflight, finalization, package, audit, publication, publisher-executor startup, alarm/retry/interleaving, provider startup, total latency/cost.

### Pre-implementation architecture-freeze gate

Before implementation begins, exact current ADR/planning state must complete a fresh adversarial review covering correctness, contract compatibility, state ownership/component necessity, async interleaving, concurrency, crash/retry/ambiguous response, privilege/evidence-authority boundaries, evidence liveness/retention, exact publication, canonical-ref lifecycle, deployment/version skew, privacy/resource limits, and avoidable duplication/latency.

The review passes only with **zero required architecture/planning edits**. Any material correction/simplification/missing invariant/changed boundary is recorded first and resets the gate; full review repeats against the new exact head. Prototype questions may remain only with accepted safe invariant-preserving fallback.

The attack review completed on 2026-09-05 against pre-amendment head `68549677e0fbb76b0018ce3aaa574c1d1ba4e1bb` found material changes and produced ADR-016 plus synchronized planning edits. It therefore **failed** the zero-edit freeze gate. Per the explicit instruction for that cycle, no fresh attack run starts automatically after the edits; implementation remains blocked until a later explicitly requested full review of the new exact head passes with zero edits.

Phase 2.6 v1 deliberately uses singleton operations. Deferred: semantic dependency quantification, batching/coalescing, automatic semantic re-preparation, general AI task orchestration, and project-orientation refresh machinery beyond the ADR-015 correctness semantics.

Exit criteria are tracked in issue #20. At minimum:

- zero-edit architecture-freeze review recorded on the exact final planning head before implementation;
- contract-v8 repository is not admitted for normal hosted mutation and supported migration to the ADR-015 contract is verified first;
- remote authenticated caller submits sealed mutation without canonical file editing/GitHub write credential;
- repository/App/canonical-ref auth cannot be gained by guessed identifiers;
- Free/protected paid private repos use same hosted mutation path;
- hosted provider code outside Core, exact pinned runtime/container/protocol identities;
- one repository DO is sole lane/state/publication-authorization authority, no second Workflow state machine;
- DO state + alarms recover accepted work without client traffic;
- alarm/request/cancel/webhook interleaving is safe through atomic phase/generation claims and stale result rejection;
- accepted response cannot strand work without durable alarm/recovery schedule;
- committed idempotency resolved before stale classification;
- cold finalizer preserves historical idempotency, fresh retry never treats unpromoted local C as committed;
- finalization evidence+receipt ordering is deterministic/no-overwrite;
- wrong-role/wrong-generation evidence cannot advance operation state and finalizer cannot create authoritative audit receipt;
- live referenced evidence survives nominal retention boundaries, while orphan/safely-terminal evidence is deleted under explicit policy;
- MemoryService builds exact C, writes Index once, hard-validates, leaves remote canonical ref H0 before audit;
- private request/candidate/audit data does not leak in ordinary metadata/log/status;
- fresh auditor verifies exact C/strict index/scope without repair/publication authority;
- only repository DO can authorize publication after exact evidence/cancel/lane/auth/ref checks;
- default/canonical-branch rename/change/deletion/transfer fails closed rather than silently redirects;
- long-lived App key remains gateway-only and publisher receives only a short-lived one-repo minimum write token after PUBLISHING;
- publisher executor performs no second source clone and changes only exact bound ref H0->exact audited C through a true expected-old Git update;
- forward/sideways/backward/deleted/recreated ref races cannot be overwritten as if H0 still held;
- ambiguous publication and webhook/order failures reconcile deterministically;
- deployment/version barriers prevent mixed incompatible finalizer/auditor/evidence/publisher semantics;
- resource/provider/privacy failures leave canonical Git unchanged;
- local/offline MemoryService remains Cloudflare-independent;
- push-on-every-memory Actions validation removed from normal hosted data plane through managed rollout;
- real private-repo rollout and measured latency/cost prove redundant work removed without deleting distinct trust boundaries.

Hosted Runethread infrastructure account requires Workers Paid because Containers are needed; this is not an end-user Cloudflare-plan requirement.

**Phase 3 is blocked until Phase 2.6 exits green.**

## Phase 3 — MCP adapters over the established memory boundary

Goal: expose the already-proven local and hosted memory operations to compatible AI clients without duplicating business logic.

Phase 3 begins only after Phase 2.6 is implemented and independently verified. MCP is transport over established MemoryService/delivery semantics; it does not own mutation, queueing, auditing, concurrency, or Git publication.

```text
local client                 remote client
    |                            |
MCP stdio/local            MCP authenticated HTTP
    |                            |
MemoryService              Runethread hosted delivery API
    |                            |
local Git                    MemoryService/Core -> user GitHub
```

Initial memory tools:

```text
memory.search
memory.get
memory.prepare
memory.apply
memory.withdraw
memory.status
```

Start with smallest transports needed for local/offline and hosted remote use. Re-check current MCP protocol/SDK/auth when Phase 3 becomes current.

Exit criteria:

- compatible AI client retrieves/mutates without repo-layout knowledge;
- local clients work without Cloudflare when local Git/exec available;
- remote clients use Phase 2.6 lifecycle without canonical Git write credentials;
- MCP code is transport translation, not repository/queue/audit rules;
- MCP integration changes do not alter canonical memory or Phase 2.6 publication invariants.

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

- fake tasks traverse full lifecycle deterministically;
- restart/resume preserves task state;
- invalid transitions rejected;
- no AI provider required for tests.

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
CLI -> Orchestrator -> capability router -> isolated worktree -> Codex
    -> build/tests -> structured result -> SQLite task history
```

Exit criteria:

- Codex cannot modify normal checkout by default;
- changed files/commands/tests/branch/worktree/unresolved verification reported;
- execution failure distinct from verification failure;
- cancellation/approval pauses tested.

## Phase 6 — general agent worker

Goal: support substantial non-coding delegated work.

Implement provider adapter under provider-independent worker interface.

Capabilities may include web research, file/document analysis, long-running reasoning, structured artifacts, and project-repo read access without automatic mutation.

Exit criteria:

- router distinguishes `chat`, `codex`, `general` capability sets;
- provider thread IDs confined to adapter/runtime metadata;
- paid API usage can be disabled without breaking Core/Codex/local workflows.

## Phase 7 — project context aggregation

Goal: make chat/session replacement cheap and reliable.

Assemble execution/context packet from relevant memory, project repo metadata/instructions/ADR pointers, active/recent tasks, blockers/next actions. It is not a new canonical database.

Exit criteria:

- fresh AI conversation recovers project continuity without prior chat URL;
- project source truth referenced rather than duplicated;
- stale runtime metadata distinguished from canonical project Git state.

## Phase 8 — broader remote project/task integration

Goal: extend established remote memory/MCP surface to project/task operations where product/platform support permits it.

Expose authenticated later-Orchestrator capabilities such as `project.*` and `task.*`. Memory remote transport is already Phase 2.6 + Phase 3.

Exit criteria:

- remote project/task access explicit auth/least privilege;
- writes retain Core/Orchestrator invariants;
- changing AI-client APIs does not require canonical-storage or Phase 2.6 architecture changes.

## Phase 9 — hardening and public readiness

Goal: make Runethread-native system safe for external users before compatibility becomes expensive.

Work includes release/migration fixtures, fuzz/property coverage, cross-platform CI/release, threat model, install/doctor UX, signing/SBOM/dependency scanning where practical, public docs, org/repo protection policy.

Exit criteria:

- new users install without GitMemo knowledge;
- supported release compatibility policy stated;
- recovery/migration tested before external adoption raises breaking-change cost.

## Later, only when justified

Possible future work:

- notifications/webhooks beyond canonical-movement webhook;
- collaboration/team policies;
- richer UI;
- multiple parallel workers;
- additional provider adapters;
- remote synchronization conveniences;
- alternative hosted execution providers;
- PostgreSQL/additional queue infrastructure only when measured scale requires it;
- container orchestration/Kubernetes only when measured requirements justify it.

## Immediate next engineering milestone

The immediate milestone is **Phase 2.6 — Memory Write Delivery Pipeline**.

> Finish the pre-implementation architecture freeze, then implement the ADR-015 contract migration followed by the Cloudflare-hosted, audited, per-repository mutation-delivery path defined by ADR-012 through ADR-016. Normal hosted writes begin only on an explicitly compatible migrated contract, independent audit uses role-separated immutable evidence, and canonical publication uses exact audited candidate objects plus a true expected-old Git update. This attack cycle found material edits, so implementation remains blocked until a later zero-edit full review of the new exact planning head passes.

Only after Phase 2.6 exits green should Phase 3 MCP implementation begin.