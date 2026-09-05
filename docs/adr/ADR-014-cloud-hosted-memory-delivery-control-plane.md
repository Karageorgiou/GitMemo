# ADR-014: Cloud-hosted Phase 2.6 memory-delivery control plane

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-012 and ADR-013 where they describe the initial GitHub-Actions-backed execution profile

## Context

ADR-012 and ADR-013 established the durable Phase 2.6 invariants: a semantic caller submits a structured MemoryService-compatible operation; deterministic Core constructs a noncanonical candidate; a fresh independent audit binds to the exact candidate; publication is an exact expected-revision fast-forward; and each canonical memory repository has one logical mutation-delivery lane with stale work parked as `NEEDS_REPREPARE` rather than silently rebased.

Their first implementation profile assumed GitHub Actions would be the initial execution adapter and that a managed GitHub ruleset/branch policy would prevent ordinary writers from bypassing the audited path. Pre-implementation verification invalidated those assumptions for private GitHub Free repositories and showed that retaining Actions as the primary executor would preserve runner-startup latency and workflow-specific plumbing that hosted execution was intended to remove.

Cloudflare provides primitives that map closely to the accepted provider-independent design: Workers for API/internal services, Durable Objects for per-repository stateful coordination, Durable Object alarms for at-least-once resumable wakeups, and Containers for the real Linux/Go/Git Runethread runtime. Containers require Workers Paid; this is an infrastructure requirement of the hosted Runethread service account, not an end-user Cloudflare-plan requirement.

The current `MemoryService.ApplyMutation` already creates an isolated transaction, regenerates Index v2, hard-validates, creates the exact mutation commit, and only fast-forwards the caller's local canonical branch. It does not push GitHub. A hosted finalizer can therefore reuse today's deterministic MemoryService implementation without making remote `main` canonical prematurely, provided the hosted wrapper never mistakes a local unpromoted candidate for canonical committed history.

A second adversarial review showed that adding Cloudflare Workflows on top of the repository Durable Object would duplicate durable operation-state/retry ownership. Once the Durable Object owns the lane, operation state, finalizer lifecycle, audit handoff, publication authorization, and crash reconciliation, a second Workflow state machine adds creation/reconciliation and retention surfaces without proving a distinct Phase 2.6 v1 invariant. Durable Object transactional storage plus at-least-once alarms and idempotent external receipts are sufficient for the singleton per-repository operation model.

Project `current-state.md` files are orientation/current-state views, not the authoritative atomic-memory transaction. They must not force Phase 2.6 to become a generic prose dual-write transaction.

## Decision

Phase 2.6 uses a **Cloudflare-hosted remote memory-delivery control plane** as its primary hosted execution profile. GitHub remains the user-owned canonical Git store. GitHub Actions remains useful for independent repository-health, recovery, migration, and control-plane validation, but is not the normal interactive mutation executor or queue.

This ADR supersedes only conflicting GitHub-Actions-first implementation details of ADR-012/ADR-013. Their candidate-before-canonical, independent-audit, exact-revision publication, idempotency, stale-reprepare, fail-closed behavior, and per-repository serialization invariants remain accepted unless explicitly amended below.

### Hosted component and release boundary

Provider-specific hosted implementation lives outside `runethread/core`. The target component/repository is `runethread/hosted` unless implementation preflight finds a concrete topology conflict before code is created.

`runethread/core` continues to own deterministic memory semantics and local interfaces. Hosted execution consumes Core through an immutable verified runtime/release or an exact explicitly verified development build. Production hosted mutation MUST NOT execute floating public `main`.

A hosted delivery release has its own explicit release/protocol identity and records the exact Core/runtime binary digest plus Container image/delivery identity used for a mutation. Finalizer and auditor use the same pinned deterministic Runethread runtime semantics but different fresh privilege contexts.

Cloudflare Worker activation and Container rollout are not assumed atomic. Breaking Worker/Container/control-protocol changes are control-plane barriers: admission must be drained/placed in maintenance or a versioned blue/green deployment must keep incompatible execution paths separate until in-flight work terminates. One operation MUST NOT silently cross incompatible hosted delivery/runtime semantics.

### One hosted architecture for GitHub Free and paid users

Runethread does not maintain separate Free-versus-paid mutation implementations.

```text
semantic client
      |
authenticated Runethread delivery API
      |
repository-runtime Durable Object
      |
attached finalizer Container -> exact candidate evidence
      |
fresh auditor Container -> exact audit evidence
      |
repository-runtime DO publication authorization
      |
private GitHub gateway/publisher
      |
user-owned private GitHub repository
```

GitHub paid branch/ruleset protection is optional defense-in-depth. It is not a correctness dependency. On an unprotected private repository, unexpected out-of-band `main` movement is detected by exact ref observation and puts the repository into fail-closed reconciliation before later hosted writes.

### Component and credential ownership

The hosted profile has four distinct responsibilities.

1. **Public Worker/API edge** — authenticates/authorizes callers, validates request envelopes and resource limits, stores/references sealed request objects, and routes to the repository coordinator. It implements no memory semantics and MUST NOT hold the GitHub App private key or a binding that can directly authorize canonical publication.
2. **Per-repository Durable Object / repository runtime** — the sole hosted lane and operation-state authority. It owns queue admission/serialization, operation state, retry/backoff scheduling, alarms, finalizer lifecycle, suspension/maintenance/reconciliation, last accepted canonical revision, audit handoff, and final publication authorization. It also manages its attached finalizer Container.
3. **Fresh auditor Container runtime** — executes the same pinned deterministic Runethread validator/index checker in a separate fresh reduced-privilege container instance/DO context. It owns no repository-lane state and has no publication authority.
4. **Private internal GitHub gateway/publisher** — holds the long-lived GitHub App private key and exposes narrow repository-read/token/publication operations through a private capability boundary such as a Service Binding. The publication-capable binding is available only to the repository-runtime control path, not to the Internet-facing API or semantic client.

No Cloudflare Workflow is required for Phase 2.6 v1. Durable Object SQLite state is the one hosted operation-state owner. Alarms provide at-least-once wakeups; every externally visible step is designed to be idempotent so alarm/request retries are safe.

### Durable Object orchestration and alarms

One repository-runtime Durable Object exists per immutable GitHub repository identity. Its persistent SQLite state contains the bounded queue plus the one active operation and its exact phase/evidence references.

The Durable Object uses a single alarm as a scheduler/recovery wakeup for the one active operation and any due queue/cleanup work. Multiple logical deadlines may be multiplexed in stored state; the alarm is set to the earliest due event.

Alarm execution is at-least-once. The alarm handler therefore performs only idempotent state-machine advancement. It catches transient downstream failures, records explicit retry/backoff state, and schedules another alarm before returning when continued retry is appropriate; correctness does not rely solely on the provider's finite automatic alarm retry count.

Incoming status/cancel/admission calls may opportunistically invoke the same bounded `drive()`/reconciliation logic, but are not required to keep an accepted operation alive. Durable state plus alarms must be sufficient after API response loss or DO eviction/restart.

The active operation is never released merely because a Container callback/status hook was lost. The DO reconciles its persisted phase with exact finalization/audit receipts and Git state before moving to a new operation.

### Canonical and hosted state ownership

Canonical semantic memory, project data, generated indexes, and committed mutation/idempotency evidence remain in the user's Git repository under ADR-001 through ADR-004.

The hosted coordinator may persist only operational delivery metadata such as:

- immutable provider repository identity and authorized App installation binding;
- lane state (`OPEN`, `SUSPENDED`, `MAINTENANCE`, `RECONCILIATION_REQUIRED` or equivalent);
- Core idempotency identity plus separate hosted operation/attempt identity;
- opaque request reference and cryptographic transport digest;
- expected/base revision;
- operation phase, retry count/backoff/deadline, and active execution identity;
- finalization/candidate/audit evidence references and digests;
- final committed revision or failure outcome.

This hosted control state is authoritative for the live hosted lane, but it is not canonical memory. A committed memory outcome must remain confirmable from exact Git history and mutation metadata. Hosted state loss MUST NOT make committed history ambiguous.

### Hosted operation identity versus Core idempotency identity

Core's idempotency key and the hosted execution-attempt identity are related but not interchangeable.

The Core idempotency key remains the semantic committed-retry identity interpreted by `MemoryService` and Git mutation metadata. The hosted operation/attempt identity additionally binds the immutable repository identity and the exact sealed-request transport digest/reference used for that hosted execution.

Two byte-different sealed request envelopes that reuse one Core idempotency key MUST NOT accidentally alias the same hosted operation attempt. They may create separate hosted attempts, while Core remains authoritative for whether the requests normalize to the same semantic fingerprint or are an idempotency conflict.

An exact resubmission of the same sealed request maps back to the same hosted attempt/status while that attempt remains recoverable. Provider transport digests are evidence-binding/deduplication tools only; they do not replace Core's semantic request fingerprint.

### Private request/evidence persistence

Full private memory Markdown/JSON or other sealed request bodies MUST NOT be stored in ordinary Durable Object operation metadata, logs, or client-visible status merely because those provider stores are convenient.

The normal hosted envelope stores the private sealed request in a private content-addressed object and passes only an opaque object reference plus digest/size/type metadata through Durable Object state. This prevents provider metadata limits/retention from becoming semantic memory limits.

Candidate/finalization/audit evidence follows the same rule. Request/candidate objects and immutable receipts MUST use content-addressed or deterministic attempt-bound keys, cryptographic digest verification, and create-if-absent/no-overwrite semantics so provider last-writer-wins behavior cannot silently mutate evidence under an existing identity.

Retention and garbage collection for request/candidate plaintext are explicit, short by default, and tested. Integrity incidents may retain evidence deliberately, but not through indefinite silent retention.

### Authentication and repository authorization

The hosted API requires an authenticated Runethread principal plus an explicit authorization binding to an installed GitHub App repository. A caller MUST NOT gain mutation authority merely by supplying or guessing a repository identifier.

Repository identity for coordination and authorization uses immutable provider identity where available, not only mutable owner/name. Repository rename/transfer, App uninstall, repository removal from an installation, or App permission changes require authorization revalidation and fail closed when access is no longer valid.

Dogfood may begin with a deliberately narrow operator/service authentication mechanism, but authentication, repository authorization, and memory semantics remain separate so Phase 3 MCP/OAuth can replace client transport without changing mutation rules.

### GitHub App permissions

The runtime GitHub App requests only the minimum repository permissions needed by the hosted path. The expected baseline is **Contents** access plus Metadata's implicit/read access. It MUST NOT request Administration or Workflows permission merely for ordinary memory delivery.

Short-lived installation tokens are scoped down per repository and role:

- finalizer/auditor: read-only Contents when Git access is required;
- publisher gateway: Contents-write only for the exact publication operation;
- semantic clients/public API: no GitHub canonical-write token.

If a future required endpoint demonstrably needs another permission, that permission addition is a reviewed security/control-plane change. Lack of Workflows permission is an intentional defense against the runtime publisher modifying `.github/workflows/**` during ordinary data-plane publication.

### Admission and whole-operation serialization

The API accepts one complete sealed MemoryService-compatible operation or a supported immutable payload reference. It does not watch a caller assemble request files across Git commits.

Phase 2.6 v1 permits concurrent semantic preparation but serializes the **whole hosted finalization/audit/publication operation per repository**. Different repositories execute independently.

Queued operations exist only as bounded DO metadata/request references. They do **not** allocate a separate Workflow/executor while waiting. A finalizer/auditor execution begins only when the DO makes that operation active.

Serialization is an availability/cost optimization layered on top of Core correctness; it does **not** authorize the hosted coordinator to classify a stale expected revision before the canonical committed-idempotency rule has been satisfied.

An operation whose `expected_revision` no longer equals current canonical state still passes through a canonical idempotency preflight using Core/Git evidence. `FindAppliedOperation`/Core request-fingerprint behavior remains authoritative and occurs before stale classification, exactly as required by ADR-003. If the key is already committed, Core returns the committed outcome/conflict semantics; only an uncommitted operation becomes `NEEDS_REPREPARE`.

Therefore Phase 2.6 does not promise that every known-stale queued operation can be rejected before cold Container/source acquisition. Without a provably complete canonical idempotency index, doing so would be unsafe for old committed retries. The guaranteed optimization boundary is **before semantic candidate construction, Index v2 write, candidate packaging, and audit**. A warm runtime or future lightweight Core/Git preflight may make this cheaper, but correctness does not depend on it.

`NO_OP` is a successful terminal result with no candidate/audit/publication, but it still passes through Core's ordinary request validation, committed-idempotency lookup, repository revision check, and hard repository validation. The coordinator MUST NOT infer semantic `NO_OP` solely from the operation label.

### Operation states

The hosted lifecycle represents meanings equivalent to:

```text
ACCEPTED -> QUEUED -> FINALIZING -> CANDIDATE_READY
         -> AUDIT_PENDING -> AUDITED -> PUBLISHING -> COMMITTED
```

and terminal/non-success outcomes equivalent to:

```text
NO_OP
ALREADY_COMMITTED
NEEDS_REPREPARE
FINALIZATION_FAILED
AUDIT_FAILED
CANCELLED
RECONCILIATION_REQUIRED
```

Exact enum names may be refined before a public API freezes; impossible independently mutable boolean combinations are prohibited.

### Finalizer execution protocol

The finalizer MUST use existing deterministic MemoryService/Core rather than a provider-specific mutation implementation.

For a cold executor, the target is at most one GitHub source clone/fetch for finalization. Reachable commit history required by `FindAppliedOperation` must be preserved. A shallow clone MUST NOT hide historical idempotency evidence. A blobless partial clone may be used only after integration tests prove current checkout, validation, transaction, idempotency lookup, and candidate packaging remain correct.

A warm clone MAY be reused only as an untrusted cache after exact fetch and hard restoration to the direct remote canonical revision. Correctness never depends on warm disk. Repository-controlled Git hooks, submodules, external filters, credential helpers, unsafe config/includes, and related execution surfaces must be disabled/refused under an explicit hosted Git hardening policy.

A **fresh finalization invocation always begins from the directly observed remote canonical state**, never from a local branch that may contain an earlier unpromoted candidate. Before invoking Core, the hosted wrapper hard-resets/reconstructs the working clone to the exact authorized remote revision and verifies cleanliness/branch/revision.

This is required because `ApplyMutation` locally fast-forwards the clone to candidate `C`; that local candidate contains Runethread operation metadata even though remote GitHub `main` has not committed it. A retry MUST NOT search that unpromoted local history and misclassify it as `ALREADY_COMMITTED`.

The DO starts/observes finalizer work under the deterministic hosted attempt ID. Repeated start/status calls are idempotent. A retry may not start a second authoritative finalization if a valid finalization result receipt already exists.

Core itself performs the canonical committed-idempotency lookup before stale expected-revision classification. If uncommitted and stale, finalization ends before mutation/index/audit work.

### Idempotent finalization result receipt

The finalization execution is wrapped in an immutable result-receipt protocol.

Before doing semantic work, the finalizer checks a deterministic immutable **finalization result receipt** for the hosted attempt. If a valid receipt exists, it verifies the receipt/evidence digests and returns that exact result rather than re-running `ApplyMutation`.

If no receipt exists:

1. refresh/reset the clone to direct remote canonical state;
2. invoke `ApplyMutation` once, preserving Core's committed-idempotency-before-stale behavior against canonical remote history;
3. for candidate success, persist complete exact candidate evidence/package first;
4. create the immutable attempt-bound finalization result receipt **last**, with create-if-absent semantics, binding the result class (`candidate`, `already_committed`, `needs_reprepare`, `no_op`, or deterministic failure as appropriate), exact `H0`, candidate `C` when present, request digest, evidence digest/reference, Core mutation metadata/request fingerprint where applicable, and runtime/delivery identity;
5. only after the receipt is durably readable/verified may the DO advance past `FINALIZING`.

The receipt's conditional create is the linearization point for the authoritative finalization result of that hosted attempt. If overlapping/retried finalizers somehow produce different candidates/results before the receipt exists, only one receipt may win. A loser MUST discard its own unreceipted output, read/verify the winning receipt, and return that result; it cannot overwrite the receipt or continue with losing evidence.

If the process crashes after local `ApplyMutation` but before receipt creation, a retry discards/resets any surviving local candidate and starts again from remote canonical state. A newly generated candidate may differ and must receive its own package before the one attempt receipt is created. Orphan candidate packages without a finalization receipt are nonauthoritative and may be garbage-collected.

If a finalization receipt exists but referenced evidence is missing/corrupt, that is an integrity failure; the system fails closed rather than silently regenerating a different result under the existing receipt.

`ALREADY_COMMITTED` is returned only from canonical Git evidence (or trusted hosted terminal state already reconciled to that canonical evidence), never merely because an unpromoted local clone contains the operation commit.

For `NO_OP`, Core returns no candidate. The hosted result receipt can preserve that response while retained, but no Git commit is invented; if hosted no-op state/receipt is later lost, Core re-evaluates the request against the then-current expected revision rather than pretending Git contains committed no-op evidence.

### Finalization failure classification

Hosted error handling distinguishes operation-local failures from repository-integrity/control-plane failures.

- an invalid proposed mutation or post-write validation failure that leaves canonical state valid is operation-local (`FINALIZATION_FAILED` or equivalent) and need not suspend unrelated later operations;
- failure to verify/validate the **canonical pre-mutation repository**, trust/lock, supported contract/repository state, or another invariant indicating the accepted canonical base itself is unhealthy drives the lane to `SUSPENDED`/`RECONCILIATION_REQUIRED` rather than repeatedly treating each request as independent mutation failure;
- disposable-cache dirtiness or provider/network errors are repaired/retried as execution failures and do not redefine canonical health.

Core error/result codes and exact validation evidence remain the basis for this classification; provider glue does not reinterpret semantic validation rules.

### Candidate evidence and transport

The exact candidate is exported as private immutable evidence bound at least to:

- repository, hosted attempt, and Core idempotency identity;
- expected/base revision `H0`;
- exact candidate commit `C` and tree;
- sealed-request transport digest plus authoritative Core mutation metadata/request fingerprint;
- exact runtime/container-image/hosted-delivery identity and pinned contract identity;
- package digest and package format version.

Candidate transport optimizes **total transferred bytes/runtime and exactness**, not a ceremonial zero-clone count. It may use a full candidate package or an exact Git-native delta/object package over `H0`. A delta package may require the fresh auditor to obtain exact `H0` through a bounded read-only fetch/clone or separately verified immutable cache.

Partial-clone/promisor state MUST NOT produce candidate evidence with missing required objects. Completeness is proven before the auditor trusts the package.

### Fresh auditor execution and audit receipt

A fresh auditor executes in a separate reduced-privilege Container/DO instance with no publication credential. Its instance identity is attempt/candidate-bound but it is **not** another repository coordinator and owns no queue/lane state.

The repository-runtime DO starts/queries the auditor idempotently using exact finalization receipt/candidate evidence. The auditor reconstructs exact `C` from immutable candidate evidence and, when needed, a bounded exact-base read.

It verifies at least:

- finalization receipt and package digest/manifest binding;
- candidate commit/tree/parent identity;
- hosted-attempt/Core-idempotency/request bindings and Core mutation metadata;
- runtime/delivery/contract identity;
- trust/control-plane state;
- hard repository validation;
- strict `index --check` freshness;
- expected mutation diff/scope and absence of unauthorized control-plane/unrelated user-data changes.

The auditor is observational: no repair/index-write step and no canonical publication credential. This independence protects against workspace/candidate/transport/privilege/race errors, not deterministic bugs shared by the same Core implementation.

Before reporting a deterministic pass/fail, the auditor writes an immutable attempt/candidate-bound **audit receipt** with create-if-absent/no-overwrite semantics. The repository DO advances from `AUDIT_PENDING` only after reading and verifying that receipt.

A deterministic audit failure leaves Git canonical state unchanged and drives durable repository-lane suspension. Missing/corrupt/conflicting audit evidence is an integrity failure and is handled fail-closed. Transient provider/network execution failure creates no successful audit receipt and is retried under the DO's explicit backoff/resource policy.

### DO-mediated publication authorization and cancellation boundary

A valid audit receipt does **not** itself authorize publication. The repository-runtime DO is the sole hosted component allowed to authorize `AUDITED -> PUBLISHING`.

In one serialized state transition it verifies:

- the operation/attempt is still the active lane owner;
- lane state still permits publication (`OPEN`, not suspended/maintenance/reconciliation);
- cancellation has not already won;
- audit evidence binds exact authorized repository/attempt/idempotency/`H0`/`C`/request/runtime identities;
- no incompatible hosted/control-plane barrier became active.

Only after that transition wins may the repository-runtime control path invoke the private GitHub gateway. The gateway requires exact repository ID, hosted attempt ID, Core idempotency identity, `H0`, `C`, audit-receipt identity/digest, and delivery/runtime identity that the Durable Object authorized. An auditor, public API Worker, or semantic client cannot independently call the publication capability.

Cancellation is permitted only before the Durable Object successfully transitions the active operation to `PUBLISHING`. Once `PUBLISHING` wins, cancellation is no longer a correctness mechanism; recovery resolves the exact CAS/ref outcome.

If the Durable Object crashes after entering `PUBLISHING`, recovery reads exact Git state/evidence: `main == C` means committed; `main == H0` permits retrying the exact authorized CAS; another value enters reconciliation. Repeating the same exact CAS is safe because the expected-old revision remains part of the publisher request.

### Exact GitHub publication

Only an independently audited and DO-authorized candidate is eligible for publication.

```text
remote main = H0
candidate   = C

audit exact C
DO authorizes exact publication
atomically require main == H0
fast-forward main H0 -> exact C
```

No force push, merge commit, squash rewrite, semantic reconstruction, or last-writer-wins update is allowed.

The implementation SHOULD prototype clone-free publication using GitHub Git-object APIs plus atomic expected-old-SHA ref update. This optimization is accepted only if integration tests prove that GitHub receives the exact audited commit/tree identity. Otherwise a minimal privileged publication environment imports exact candidate evidence and pushes exact `C`.

After successful CAS, normal completion performs a cheap independent ref read and requires `main == C`; it does not repeat full clone/index/validation of the same immutable candidate.

### Webhook observation is a hint, not state authority

The GitHub App SHOULD subscribe to signed `push` events for fast canonical-movement observation. Signature and delivery identity are verified and handled idempotently.

A webhook payload never directly advances the coordinator's accepted canonical revision or independently changes a lane into reconciliation. It is a trigger/hint to perform an authoritative direct ref read.

On delivery:

1. verify signature/delivery/repository identity;
2. read the current canonical ref directly;
3. if the webhook's `after` is no longer current, treat it as stale/out-of-order observation;
4. if current ref matches an expected hosted publication, reconcile/confirm that operation;
5. if current ref is unexpected, enter fail-closed reconciliation.

Correctness therefore remains safe under duplicate, delayed, reordered, or missed webhooks.

### Out-of-band movement and reconciliation

Before heavy semantic candidate construction and again at publication, hosted Runethread compares direct observed canonical state to accepted/expected state. Unexpected movement not explained by the one active hosted operation or authorized maintenance/recovery yields `RECONCILIATION_REQUIRED`.

Recovery independently inspects the exact out-of-band revision and either adopts it after applicable trust/validation/index/repository-health checks establish a new accepted canonical base, or repairs/restores through an explicit authorized recovery procedure. Owner permission alone does not make arbitrary movement audited.

### Control-plane barriers

Contract/schema/trust/repository-format/bootstrap/migration/managed-delivery changes remain exclusive barriers. The repository coordinator represents the barrier as `MAINTENANCE` or equivalent; new data-plane publication is refused/parked and pre-barrier prepared operations cannot publish under post-barrier semantics without re-preparation.

Installing or materially changing Phase 2.6 is itself a barrier and is rolled out through the full Core/hosted release/downstream process rather than through the data-plane path it is creating.

### Resource, abuse, and privacy limits

The hosted adapter enforces explicit limits for at least authenticated request rate/queued work, inline request size, repository/current-tree size, candidate/audit evidence size/retention, Durable Object operation-history retention/garbage collection, Container CPU/memory/disk/wall time, alarm retry/backoff/operation lifetime, and log/event size.

Large legitimate requests use private immutable referenced payloads rather than inheriting provider event-size caps as semantic memory limits. Resource/quota/provider failure leaves canonical Git unchanged and remains distinguishable from deterministic semantic/finalization failure.

Hosted private-memory processing means Cloudflare infrastructure temporarily processes authorized plaintext repository/candidate data. Public rollout therefore requires an explicit data-handling/threat-model review covering private object storage, DO metadata, logs, credential isolation, retention/deletion, and provider access boundaries.

### Project current-state views are projections

Atomic memory mutation does not become a generic writer for `projects/<slug>/current-state.md` or other orientation prose. Canonical atomic memories and authoritative project source repositories remain sources of truth. Project overview/current-state files are orientation/materialized views whose future refresh mechanism is separate and has explicit freshness semantics.

### MCP relationship

Phase 2.6 establishes provider-neutral hosted delivery operations before MCP transport. Phase 3 is a thin MCP transport over MemoryService and this hosted delivery lifecycle. Local/offline Runethread continues to use MemoryService directly without Cloudflare.

## Consequences

- GitHub Free and paid users share one hosted mutation architecture; paid protection is defense-in-depth.
- Hosted Runethread requires Workers Paid for Containers; end users do not need Cloudflare accounts/plans.
- Provider-specific code/release lifecycle stays outside Core.
- One repository-runtime Durable Object owns queue/lane state, retry scheduling, attached finalizer Container, audit handoff, and publication authorization.
- Phase 2.6 v1 has no Cloudflare Workflow state machine; removing it eliminates a redundant operation-state owner, start/reconciliation race, retention surface, and provider dependency.
- At-least-once DO alarms plus explicit persisted phase/backoff and immutable finalization/audit receipts provide crash/retry recovery.
- Queued operations allocate only bounded DO/request-reference state until they become active.
- Workflow/DO plaintext-retention concerns disappear; DO metadata still stores opaque request/evidence references rather than private memory bodies.
- Existing MemoryService remains the single mutation engine and preserves ADR-003 idempotency-before-stale semantics.
- Hosted serialization avoids duplicate heavy work, but stale classification still waits for canonical committed-idempotency preflight; the guaranteed savings are candidate/index/package/audit work, not every cold Container startup.
- The hosted finalization receipt prevents an unpromoted local candidate from being mistaken for canonical `already_applied` state after a lost finalizer response.
- Receipt conditional creation linearizes one authoritative finalization result when overlapping retries occur.
- Canonical precondition/trust failure suspends the lane instead of being treated as an ordinary request-local mutation failure.
- Fresh audit remains a distinct trust boundary and produces immutable audit evidence before the DO may publish.
- The DO, not auditor/public API, authorizes the publication critical section and defines cancellation ordering.
- Long-lived GitHub App authority is isolated; baseline App permissions exclude Administration/Workflows.
- Candidate/request/evidence uses content-addressed/deterministic, digest-verified, no-overwrite storage semantics.
- Webhooks accelerate observation but cannot mutate accepted canonical state without a direct ref read.
- Publisher can be clone-free only if exact object identity is proven; exact candidate push remains the safe fallback.
- Per-write post-publication full validation is removed from normal synchronous delivery.
- Breaking hosted deployments require version/barrier handling because provider rollout is not assumed atomic.
- Project orientation prose is not an atomic-memory dual-write requirement.

## Alternatives considered

### Keep GitHub Actions as primary executor
Rejected: runner latency/workflow plumbing would remain in the interactive critical path and would be replaced by the hosted path anyway.

### Add Cloudflare Workflows on top of the repository Durable Object
Rejected for Phase 2.6 v1. The repository DO already must own the serialized lane, persistent operation state, finalizer lifecycle, audit handoff, cancellation, publication authorization, and crash reconciliation. A second durable Workflow state machine duplicates state/retry ownership and introduces cross-system start/callback/retention coordination. The singleton per-repository operation can be driven by transactional DO state, at-least-once alarms, and idempotent receipts. A future more complex orchestration may justify Workflows separately.

### Separate Free and paid GitHub implementations
Rejected: branch protection availability should not fork mutation semantics.

### Put hosted code in Core
Rejected: provider runtime/deployment/credentials must not contaminate the deterministic memory compatibility surface.

### Store the App key in user workflows or public API Worker
Rejected: canonical publisher authority belongs behind a private internal capability boundary.

### Rewrite MemoryService for Workers
Rejected: it creates a second mutation engine solely to avoid Containers.

### Add a hosted complete idempotency database solely to reject stale work before source acquisition
Rejected: Git history/Core remains canonical committed-idempotency authority. Duplicating all historical operation identity merely to avoid a possible cold clone adds a second recovery authority and complexity disproportionate to the optimization.

### Use persistent cloud Git as canonical
Rejected: the user-owned Git repository remains canonical.

### Re-run `ApplyMutation` blindly after a lost finalizer response
Rejected: the disposable clone may already contain unpromoted candidate `C`, so `FindAppliedOperation` on local HEAD could mistake it for canonical committed evidence. Exact result receipt or canonical reset is required.

### Let auditor publish directly after audit
Rejected: it races lane suspension/cancellation/maintenance and splits publication authority from the repository coordinator.

### Treat webhook payloads as canonical ref truth
Rejected: duplicate/delayed/out-of-order delivery would create false state changes.

### Require zero GitHub reads after finalization
Rejected: minimize total exact transferred work rather than an arbitrary clone counter.

### Run multiple heavy mutation operations per repo in v1
Rejected: singleton publication plus unknown semantic dependence makes most same-base parallel work wasteful/stale.

### Merge finalizer and auditor
Rejected: saving one Container would remove an intentional fresh reduced-privilege trust boundary.

### Make project current-state prose part of every mutation
Rejected: creates a semantic dual-write requirement and expands critical mutation scope.

## Verification

Implementation evidence must demonstrate at least:

1. one normal hosted path works for private GitHub Free and protected paid repositories;
2. provider-specific hosted code remains outside Core and pins exact Core/runtime/image identities;
3. caller auth plus repository-installation authorization prevents repository-ID guessing from granting mutation access;
4. runtime App baseline excludes Administration/Workflows permissions and role tokens are repository/permission narrowed;
5. public API/finalizer/auditor never receive the long-lived App private key or direct publication capability;
6. one repository-runtime DO is the sole queue/lane/operation/publication authority and different repositories progress independently;
7. no Cloudflare Workflow is required for v1 correctness/recovery; queued operations consume only bounded DO/request-reference state;
8. DO SQLite state plus at-least-once alarms resume an accepted active operation after API response loss, DO eviction/restart, and transient downstream failure without incoming client traffic;
9. the alarm driver is idempotent, explicitly reschedules after prolonged retryable failures, and cannot release the lane without exact state/evidence reconciliation;
10. hosted attempt identity binds repository + Core idempotency + sealed-request digest so byte-different envelopes cannot alias one hosted attempt accidentally;
11. any stale classification that could overlap a committed retry runs canonical Core/Git idempotency lookup first; no incomplete hosted cache can override ADR-003;
12. stale uncommitted work stops before candidate construction/Index write/package/audit even when cold source acquisition was required to prove it;
13. provider transport digests do not replace Core's semantic idempotency fingerprint/conflict rules;
14. `NO_OP` still executes Core's request/revision/idempotency/repository validation but skips candidate/audit/publication;
15. a cold finalizer performs no more than one source clone/fetch and preserves historical canonical idempotency evidence;
16. warm clone reuse and Git execution are hardened against dirty/stale/repository-controlled execution surfaces;
17. every fresh finalization invocation starts from direct remote canonical state, not surviving unpromoted local candidate history;
18. after `ApplyMutation`, candidate evidence is persisted and an immutable finalization result receipt is created last before the DO can advance;
19. concurrent/overlapping finalizer retries cannot select two authoritative results: conditional receipt creation chooses one winner and losers return/discard accordingly;
20. a lost finalizer response with surviving local C cannot produce false `ALREADY_COMMITTED`; retry either uses the valid receipt or resets to remote canonical state and re-finalizes;
21. receipt/evidence mismatch or missing evidence fails closed rather than silently regenerating under the old receipt;
22. `ApplyMutation` constructs exact `C`, writes Index v2 once, hard-validates, and leaves remote `main` at `H0`;
23. canonical pre-mutation trust/validation/compatibility failure suspends/reconciles the repository, while request-local proposed-mutation failure does not unnecessarily block unrelated later work;
24. private request/candidate/finalization/audit receipt objects are digest-verified, no-overwrite, short-retention, and absent from ordinary DO/log/client plaintext;
25. candidate packaging is complete under the selected clone mode and cannot omit required promisor objects;
26. a fresh auditor verifies exact `C`, hard validation, strict Index v2 freshness, and scope without repair/publication authority;
27. auditor retries are idempotent and exactly one immutable audit receipt/result is authoritative for an attempt/candidate;
28. deterministic audit failure leaves Git canonical state unchanged and cannot release the lane without durable suspension/reconciliation;
29. after audit, only the repository DO can transition `AUDITED -> PUBLISHING`, and cancellation/maintenance/suspension races are resolved before publisher authority is granted;
30. publisher requests are exact-repository/attempt/idempotency/audit-bound and publication requires remote `main == H0` before moving only to audited `C`;
31. duplicate publisher calls and CAS response loss are resolved from exact ref/evidence without duplicate semantic mutation;
32. clone-free Git-object publication is used only if exact object identity is proven; otherwise exact-candidate push fallback works;
33. successful publication gets only cheap exact-ref confirmation in the normal synchronous path;
34. duplicate/delayed/out-of-order/missed signed webhooks remain safe because they trigger direct ref reconciliation rather than directly mutating accepted state;
35. unexpected/out-of-band movement enters reconciliation before later hosted writes;
36. App uninstall/repository removal/permission loss fails closed and requires authorization revalidation;
37. control-plane barriers prevent pre-barrier work from publishing under changed semantics;
38. non-atomic Worker/Container/control deployment cannot mix incompatible semantics inside one operation;
39. resource/quota/provider failures preserve canonical Git state and are distinguishable from deterministic failures;
40. project orientation files are not silently dual-written by hosted provider code;
41. local/offline MemoryService remains usable without Cloudflare and Core imports no hosted provider dependencies;
42. push-on-every-normal-memory GitHub Actions validation is removed through proper managed rollout while distinct recovery/migration/health checks remain possible;
43. Phase 3 MCP can expose the same delivery lifecycle without changing canonical memory format or mutation semantics;
44. end-to-end measurements separate cold/warm source acquisition, bytes transferred, idempotency/stale preflight, finalization, packaging, audit, publication, alarm/retry overhead, and provider startup so optimization is evidence-driven.