# ADR-014: Cloud-hosted Phase 2.6 memory-delivery control plane

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-012 and ADR-013 where they describe the initial GitHub-Actions-backed execution profile

## Context

ADR-012 and ADR-013 established the durable Phase 2.6 invariants: a semantic caller submits a structured MemoryService-compatible operation; deterministic Core constructs a noncanonical candidate; a fresh independent audit binds to the exact candidate; publication is an exact expected-revision fast-forward; and each canonical memory repository has one logical mutation-delivery lane with stale work parked as `NEEDS_REPREPARE` rather than silently rebased.

Pre-implementation verification invalidated the original GitHub-Actions-first/private-ruleset profile for the normal hosted path. Private GitHub Free repositories cannot rely on private branch/ruleset protection, while retaining Actions as the primary executor would preserve runner-startup latency and workflow-specific plumbing that hosted execution was intended to remove.

Cloudflare provides primitives that fit the provider-independent architecture: Workers for API/internal services, Durable Objects for per-repository stateful coordination, Durable Object alarms for at-least-once recovery wakeups, and Containers for the real Linux/Go/Git Runethread runtime. Containers require Workers Paid; this is an infrastructure requirement of the hosted Runethread service account, not an end-user Cloudflare-plan requirement.

The current `MemoryService.ApplyMutation` already owns committed-idempotency lookup, stale checking, mutation semantics, Index v2 generation, hard validation, mutation commit creation, and a local fast-forward. It does not push GitHub. Hosted Phase 2.6 therefore reuses this implementation rather than creating a second mutation engine.

Adversarial review found three additional implementation-boundary requirements:

1. Cloudflare Workflows would duplicate operation-state/retry ownership already required of the per-repository Durable Object, so Phase 2.6 v1 does not use them.
2. A local unpromoted candidate already contains Runethread mutation trailers, so a retry must never search that local candidate history and mistake it for canonical committed evidence.
3. Durable Objects are single-threaded but asynchronous external I/O allows requests/alarms to interleave. Correctness therefore requires explicit durable phase/generation linearization rather than assuming one Durable Object means one uninterrupted external transaction.

The existing Core transaction also operates on the currently checked-out named branch; the released memory contract does not establish a literal branch name such as `main` as a semantic invariant. The hosted profile therefore binds each authorized repository to an explicit **canonical branch ref** rather than hard-coding a branch name.

Project `current-state.md` files remain orientation/materialized views, not part of the authoritative atomic-memory transaction.

## Decision

Phase 2.6 uses a **Cloudflare-hosted remote memory-delivery control plane** as its primary hosted execution profile. GitHub remains the user-owned canonical Git store. GitHub Actions remains useful for independent repository-health, PR, recovery, migration, and control-plane validation, but is not the normal interactive mutation executor or queue.

This ADR supersedes only conflicting implementation-profile details of ADR-012/ADR-013. Their candidate-before-canonical, independent-audit, exact-revision publication, idempotency, stale-reprepare, fail-closed behavior, and per-repository serialization invariants remain accepted unless explicitly amended below.

### Hosted component and release boundary

Provider-specific hosted implementation lives outside `runethread/core`. The target component/repository is `runethread/hosted` unless implementation preflight finds a concrete topology conflict before code is created.

`runethread/core` continues to own deterministic memory semantics and local interfaces. Hosted execution consumes Core through an immutable verified runtime/release or exact explicitly verified development build. Production hosted mutation MUST NOT execute floating public `main`.

A hosted delivery release has its own explicit release/protocol identity and records the exact Core/runtime binary digest plus Container image/delivery identity used by an operation. Finalizer and auditor use the same pinned deterministic Runethread semantics but separate fresh privilege contexts.

Cloudflare Worker/Container/control-path rollout is not assumed atomic. Breaking changes are control-plane barriers: admission is drained/placed in maintenance or versioned blue/green execution keeps incompatible paths separate until in-flight work terminates. One operation MUST NOT cross incompatible hosted protocol/runtime generations.

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
repository-runtime DO authorizes publication
      |
private GitHub gateway/publisher
      |
user-owned private GitHub repository / canonical branch ref
```

GitHub paid branch/ruleset protection is optional defense-in-depth, not a correctness dependency. On an unprotected private repository, unexpected out-of-band canonical-ref movement is detected through exact ref observation and fails closed into reconciliation before later hosted writes.

### Canonical branch-ref binding

Hosted Runethread binds each authorized repository to:

- immutable GitHub repository identity;
- authorized GitHub App installation identity;
- an explicit canonical branch ref, normally discovered from the repository's default branch at onboarding/adoption;
- last accepted canonical commit revision for that ref.

The canonical branch name is hosted configuration/authorization state, not a new field in canonical memory and not a hard-coded `main` invariant.

Every preparation/finalization/publication/ref-observation operation targets the bound canonical ref. Repository rename may preserve the immutable repository identity, but default-branch rename/change, canonical-ref deletion, transfer, installation change, or loss of access requires revalidation. The hosted service MUST NOT silently follow a changed default branch or begin publishing to a new ref merely because GitHub metadata changed.

Examples below use `H0`/`C` and “canonical ref”; a literal `main` branch is only an example deployment name.

### Component and credential ownership

The hosted profile has four responsibilities.

1. **Public Worker/API edge** — authenticates/authorizes callers, validates request envelopes/resource limits, stores/references sealed request objects, and routes to the repository coordinator. It implements no memory semantics and MUST NOT hold the GitHub App private key or publication-capable binding.
2. **Per-repository Durable Object / repository runtime** — sole hosted lane/operation-state authority. It owns queue admission/serialization, durable phase/generation state, retry/backoff/alarms, attached finalizer lifecycle, suspension/maintenance/reconciliation, canonical-ref binding/last accepted revision, audit handoff, and final publication authorization.
3. **Fresh auditor Container runtime** — executes the same pinned deterministic Runethread validator/index checker in a separate fresh reduced-privilege Container/DO context. It owns no repository-lane state and no publication authority.
4. **Private internal GitHub gateway/publisher** — holds the long-lived GitHub App private key and exposes narrow repository-read/token/publication operations through a private capability boundary such as a Service Binding. Publication capability is reachable only from the repository-runtime control path.

No Cloudflare Workflow is required for Phase 2.6 v1. Durable Object SQLite is the one hosted operation-state owner. Alarms provide at-least-once wakeups; external side effects are idempotent and phase/generation-bound.

### Durable Object state machine, interleaving, and alarms

One repository-runtime Durable Object exists per immutable GitHub repository identity. Its transactional SQLite state contains the bounded queue, one active operation, canonical-ref binding, exact operation phase, execution generation, retry/backoff/deadline state, and evidence references.

The DO uses one alarm as scheduler/recovery wakeup for the active operation and due queue/cleanup work. Multiple logical deadlines are multiplexed in stored state and the alarm is set to the earliest due event. Alarms are at-least-once; retryable downstream failure is caught, explicit backoff is persisted, and another alarm is scheduled so correctness does not depend solely on the provider's finite automatic retries.

**Single DO identity does not mean external calls execute as one uninterrupted critical section.** Alarms and requests may interleave while a handler awaits Container, R2/object-store, GitHub, or other external I/O.

For every externally visible step, the DO uses this protocol:

1. in a local atomic SQLite transition, verify the active operation/phase and claim the next external action with an exact `execution_generation` (or equivalent monotonic phase token), expected evidence inputs, and retry identity;
2. persist that claim before starting external I/O;
3. perform external I/O without holding `blockConcurrencyWhile()` across the network/container wait;
4. external work is idempotent and bound to repository + hosted attempt + phase + execution generation and immutable input digests;
5. after the external result returns, re-read/atomically compare active operation, phase, and generation before accepting the result;
6. a late/stale result from an obsolete generation cannot advance state; any produced unreferenced evidence is nonauthoritative and is garbage-collected under policy.

`blockConcurrencyWhile()` is reserved for short initialization/migration/local-only critical setup. It MUST NOT wrap normal GitHub/R2/Container I/O.

Status, admission, cancel, webhook, alarm, and recovery requests may all invoke the same bounded idempotent `drive()` logic. A second driver seeing an already-claimed in-flight generation does not start a different authoritative action; recovery may retry the **same generation/action identity** only under that phase's idempotency rules.

### Acceptance and alarm handoff

A request is not reported as durably `ACCEPTED` until its sealed request reference/digest and operation metadata are durably stored **and** the repository DO has confirmed a due alarm/recovery wakeup for pending work.

If the process fails before that confirmation/response, the caller has not received durable acceptance. Exact resubmission is safe and repairs any stored-but-unscheduled attempt. Admission, status, cancel, webhook, and DO construction/recovery paths also verify that queued/active work has an alarm when one is required and repair a missing schedule.

The implementation must verify Cloudflare storage/alarm output-gate behavior in integration tests; it MUST NOT assume an undocumented cross-provider transaction. The safety fallback is exact resubmission plus deterministic schedule repair, not loss of an accepted operation.

### Canonical and hosted state ownership

Canonical semantic memory, project data, generated indexes, and committed mutation/idempotency evidence remain in the user's Git repository under ADR-001 through ADR-004.

Hosted coordinator state may contain only operational metadata such as:

- immutable repository/App-installation/canonical-ref binding;
- lane state (`OPEN`, `SUSPENDED`, `MAINTENANCE`, `RECONCILIATION_REQUIRED` or equivalent);
- Core idempotency identity plus separate hosted attempt identity;
- opaque request reference/digest;
- expected/base revision;
- active phase/execution generation/retry/backoff/deadline;
- finalization/candidate/audit evidence references/digests;
- final committed revision/failure outcome.

This state is authoritative for the live hosted lane, not for semantic memory. A committed memory result remains confirmable from exact canonical Git history and mutation metadata. Hosted state loss MUST NOT make committed history ambiguous.

### Hosted attempt identity versus Core idempotency identity

Core's idempotency key remains the semantic committed-retry identity interpreted by MemoryService/Git mutation metadata. Hosted attempt identity additionally binds immutable repository identity, canonical-ref identity, and exact sealed-request transport digest/reference.

Byte-different sealed envelopes reusing one Core key MUST NOT alias the same hosted attempt accidentally. They may create separate hosted attempts while Core remains authoritative for semantic equivalence/conflict. Exact resubmission of the same sealed request maps back to the same hosted attempt/status while recoverable.

Transport digests are evidence-binding/deduplication aids only; they do not replace Core's normalized semantic request fingerprint.

### Private request/evidence persistence

Full private memory Markdown/JSON or other request bodies MUST NOT be stored in ordinary DO operation metadata, logs, or client-visible status.

The normal hosted envelope stores private request data in private content-addressed/no-overwrite object storage and keeps only an opaque reference plus digest/size/type in DO state. Candidate/finalization/audit evidence follows the same rule.

Request/candidate/audit objects and immutable receipts use content-addressed or deterministic attempt/phase/generation-bound keys, cryptographic digest verification, and create-if-absent/no-overwrite semantics. Retention/garbage collection is explicit and short by default; integrity incidents may retain evidence deliberately but not through indefinite silent retention.

### Authentication, authorization, and GitHub App permissions

The hosted API requires an authenticated Runethread principal plus explicit authorization to an installed GitHub App repository/canonical-ref binding. Supplying or guessing a repository ID is never sufficient.

Repository transfer, App uninstall/removal, permission change, canonical/default-branch change, or ref deletion requires revalidation and fails closed when authorization/binding is no longer valid.

Dogfood may begin with narrow operator/service auth, but authentication, repository authorization, and memory semantics remain separate so Phase 3 MCP/OAuth can replace client transport without changing mutation rules.

The runtime GitHub App requests minimum permissions. Expected ordinary data-plane baseline is **Contents** plus implicit/read Metadata. It MUST NOT request Administration or Workflows permission merely for memory delivery.

Short-lived installation tokens are narrowed per repository/role:

- finalizer/auditor: Contents-read when Git access is required;
- publisher gateway: Contents-write only for exact authorized publication;
- semantic clients/public API: no canonical Git write credential.

Any later permission addition is a reviewed security/control-plane change.

### Admission and whole-operation serialization

Phase 2.6 v1 permits concurrent semantic preparation but serializes the **whole hosted finalization/audit/publication operation per repository**. Different repositories execute independently. Waiting operations are bounded DO metadata/request references and allocate no separate Workflow/executor.

Serialization is an availability/cost optimization; it does not permit stale classification before ADR-003 committed-idempotency semantics are satisfied.

An attempt whose expected revision no longer equals current canonical state still passes through canonical Core/Git idempotency lookup first. If its key already committed, Core returns committed/conflict semantics; only a proven-uncommitted stale attempt becomes `NEEDS_REPREPARE`.

Therefore stale queued work may still require cold Container/source acquisition. The guaranteed optimization boundary is before semantic candidate construction, Index v2 write, candidate packaging, and audit.

`NO_OP` remains Core-validated. It still performs request validation, committed-idempotency lookup, exact revision check, and hard canonical repository validation, but produces no candidate/audit/publication.

### Operation states

Hosted lifecycle meanings are equivalent to:

```text
ACCEPTED -> QUEUED -> FINALIZING -> CANDIDATE_READY
         -> AUDIT_PENDING -> AUDITED -> PUBLISHING -> COMMITTED
```

with terminal/non-success outcomes equivalent to:

```text
NO_OP
ALREADY_COMMITTED
NEEDS_REPREPARE
FINALIZATION_FAILED
AUDIT_FAILED
CANCELLED
RECONCILIATION_REQUIRED
```

Exact enum names may be refined before public API freeze. Impossible independently mutable boolean combinations are prohibited.

### Finalizer execution protocol

The finalizer MUST use existing deterministic MemoryService/Core.

Cold target is at most one GitHub source clone/fetch. Reachable commit history required by `FindAppliedOperation` must be preserved. Shallow history MUST NOT hide historical idempotency evidence. Blobless partial clone is allowed only after integration tests prove checkout, validation, transaction, idempotency lookup, and candidate packaging correctness.

A warm clone is an untrusted cache only. Every fresh finalization invocation fetches and hard-restores/reconstructs to the directly observed canonical branch ref/revision and verifies named branch/cleanliness/revision before calling Core. Correctness never depends on warm disk.

Hosted Git execution disables/refuses repository-controlled hooks, recursive submodules, external filters, credential helpers, unsafe config/includes, and similar execution surfaces.

A fresh invocation NEVER starts from a local branch containing an earlier unpromoted candidate. `ApplyMutation` locally fast-forwards to candidate `C`; that local C already contains operation metadata even though remote canonical state does not. Searching that history on retry could falsely yield `ALREADY_COMMITTED`.

Finalizer start/result calls are generation-bound and idempotent. A valid immutable finalization receipt prevents a new authoritative semantic finalization.

### Idempotent finalization receipt

Before semantic work, finalizer checks the deterministic immutable finalization receipt for hosted attempt/generation. If valid, it verifies receipt/evidence digests and returns that exact result.

If no valid receipt exists:

1. restore clone to direct canonical ref/revision;
2. call `ApplyMutation`, preserving canonical committed-idempotency-before-stale behavior;
3. on candidate success, persist complete exact candidate evidence first;
4. create immutable attempt/generation-bound finalization receipt **last** using create-if-absent semantics, binding result class, H0, C when present, request/evidence digests, Core mutation metadata/request fingerprint, and runtime/delivery identity;
5. only after receipt is durably readable/verified may the DO accept the result for the currently claimed generation.

If overlapping retries somehow produce different results before a receipt exists, only one receipt wins. Losing outputs are nonauthoritative. Missing/corrupt receipt evidence is integrity failure; never silently regenerate a different result under an existing receipt.

If crash occurs after local ApplyMutation but before receipt creation, retry discards/resets local candidate and begins from remote canonical state. Newly generated C may differ and must receive fresh evidence before receipt creation.

`ALREADY_COMMITTED` comes only from canonical Git evidence or hosted terminal state already reconciled to that evidence. For `NO_OP`, no Git commit is invented; loss of hosted no-op receipt means Core re-evaluates against the then-current expected revision.

### Finalization failure classification

- invalid proposed mutation/post-write candidate validation failure with healthy canonical base is operation-local;
- failure to verify/validate the **canonical pre-mutation repository**, trust/lock, supported contract/repository state, or bound canonical ref drives `SUSPENDED`/`RECONCILIATION_REQUIRED` rather than endless request-local failures;
- disposable-cache dirtiness/provider/network failures are execution failures and do not redefine canonical health.

Core error/result codes and exact validation evidence remain the classification basis.

### Candidate evidence and transport

Exact candidate evidence binds at least:

- repository/canonical-ref, hosted attempt/generation, Core idempotency identity;
- H0, exact C, tree, and parent relation;
- sealed-request digest plus Core request fingerprint/mutation metadata;
- exact runtime/container/delivery/contract identities;
- package digest/format version.

Transport optimizes total exact bytes/runtime, not a zero-clone slogan. It may use a self-contained package or exact Git-native delta/object package over H0. A delta may require fresh auditor to obtain exact H0 through bounded read-only acquisition.

Partial-clone/promisor state MUST NOT yield missing required objects. Package completeness is proven before auditor trust.

### Fresh auditor and audit receipt

Auditor executes in a separate fresh reduced-privilege Container/DO context with no publication credential/lane authority. Its work is attempt/candidate/generation-bound and idempotent.

It reconstructs exact C from immutable candidate evidence plus bounded exact-base read when necessary and verifies:

- finalization receipt/package digest/manifest;
- candidate commit/tree/parent/canonical-ref identity;
- hosted attempt/Core idempotency/request bindings/mutation metadata;
- runtime/delivery/contract identity;
- trust/control-plane state;
- hard repository validation;
- strict `index --check` freshness;
- expected diff/scope and absence of unauthorized control-plane/unrelated user-data changes.

Auditor performs no repair/index-write and has no publication credential.

Before pass/fail is accepted, auditor writes immutable attempt/candidate/generation-bound audit receipt with create-if-absent/no-overwrite semantics. Deterministic audit failure leaves canonical state unchanged and durably suspends/reconciles the lane. Provider/network failure creates no successful audit receipt and is retried under explicit DO backoff.

### DO-mediated publication and cancellation linearization

A valid audit receipt does not itself authorize publication. The repository DO alone may authorize `AUDITED -> PUBLISHING`.

In one local atomic state transition, the DO verifies:

- active operation/attempt and current execution generation;
- lane is publishable (`OPEN`);
- cancellation has not already won;
- audit evidence binds exact repository/canonical-ref/attempt/idempotency/H0/C/request/runtime identities;
- repository/App/canonical-ref authorization still valid;
- direct canonical ref still equals H0;
- no incompatible control-plane barrier is active.

If this transition wins, `PUBLISHING` plus exact publication request identity is durably recorded before publisher I/O. A concurrent cancellation arriving afterward is too late. If cancellation's atomic terminal transition wins first, publication is not authorized.

Publisher I/O occurs outside DO local critical sections and is idempotent/generation-bound. After any response or timeout, the DO rechecks active operation/generation before accepting a result.

### Exact GitHub publication

Only independently audited, DO-authorized candidate C is eligible.

```text
canonical ref = H0
candidate     = C

audit exact C
DO authorizes exact publication
atomically require canonical ref == H0
fast-forward canonical ref H0 -> exact C
```

No force push, merge commit, squash rewrite, semantic reconstruction, or last-writer-wins update.

Preferred prototype uses GitHub Git-object APIs plus atomic expected-old-SHA ref update. It is accepted only if integration tests prove exact C/tree identity. Otherwise a minimal privileged environment imports exact candidate evidence and pushes exact C.

After ambiguous publication response:

- canonical ref == C -> COMMITTED;
- canonical ref == H0 -> same exact authorized publication may be retried;
- any other value -> `RECONCILIATION_REQUIRED`.

Successful normal completion does only cheap exact-ref confirmation; no redundant full clone/index/validation cycle.

### Webhook observation

GitHub App SHOULD subscribe to signed `push` events for fast observation. Signature/delivery/repository identity are verified and handled idempotently.

Webhook payload never directly advances accepted canonical state. It triggers authoritative read of the **bound canonical ref**. Stale/out-of-order payload is ignored after current-ref read; expected hosted movement may confirm/reconcile the active op; unexpected current movement enters fail-closed reconciliation.

Correctness remains safe under duplicate, delayed, reordered, or missed webhooks.

### Out-of-band movement and reconciliation

Before semantic candidate construction and again at publication, Runethread compares directly observed bound canonical ref/revision to accepted/expected state. Unexpected movement not explained by active hosted operation or explicit maintenance/recovery yields `RECONCILIATION_REQUIRED`.

Recovery independently inspects exact out-of-band revision and either adopts it after applicable trust/validation/index/repository-health checks establish a new accepted canonical base or repairs/restores through explicit authorized recovery. Owner permission alone does not make arbitrary movement audited.

### Control-plane barriers

Contract/schema/trust/repository-format/bootstrap/migration/managed-delivery/canonical-ref-binding changes are exclusive barriers. Repository coordinator represents barrier as `MAINTENANCE` or equivalent; new publication is refused/parked and pre-barrier prepared operations cannot publish under changed semantics without re-preparation.

Installing/materially changing Phase 2.6 itself is a barrier and uses full Core/hosted release/downstream process.

### Resource, abuse, and privacy limits

Hosted adapter enforces explicit limits for authenticated request rate/queued work, inline/request-object size, repository/current-tree size, candidate/audit evidence size/retention, DO operation-history retention, Container CPU/memory/disk/wall time, alarm retry/backoff/operation lifetime, and log/event size.

Large legitimate requests use private immutable referenced payloads rather than provider event-size caps as semantic limits. Resource/quota/provider failure leaves canonical Git unchanged and remains distinguishable from deterministic semantic/finalization failure.

Hosted private-memory processing means Cloudflare temporarily handles authorized plaintext repository/candidate data. Public rollout requires explicit data-handling/threat-model review covering private object storage, DO metadata, logs, credential isolation, retention/deletion, and provider access boundaries.

### Project current-state views are projections

Atomic memory mutation does not become a generic writer for `projects/<slug>/current-state.md` or other orientation prose. Canonical atomic memories and authoritative project source repositories remain sources of truth. Project overview/current-state files are orientation/materialized views with separate future refresh/freshness semantics.

### MCP relationship

Phase 2.6 establishes provider-neutral hosted delivery operations before MCP transport. Phase 3 is thin MCP transport over MemoryService and this hosted delivery lifecycle. Local/offline Runethread continues to use MemoryService directly without Cloudflare.

## Consequences

- GitHub Free and paid users share one hosted mutation architecture; paid protection is defense-in-depth.
- Hosted Runethread requires Workers Paid for Containers; end users do not need Cloudflare accounts/plans.
- Provider-specific code/release lifecycle stays outside Core.
- One repository-runtime DO owns lane/queue state, phase generations, retries/alarms, attached finalizer lifecycle, audit handoff, and publication authorization.
- Phase 2.6 v1 has no Cloudflare Workflow state machine.
- DO asynchronous interleaving is handled explicitly through durable atomic phase/generation claims; no external I/O is protected by a long `blockConcurrencyWhile()` lock.
- Acceptance includes durable operation storage plus recoverable alarm scheduling/repair semantics.
- Canonical branch is an explicit authorized ref binding, not a hard-coded `main` assumption.
- Existing MemoryService remains the single semantic mutation engine and preserves ADR-003 committed-idempotency-before-stale semantics.
- Stale classification may require cold source acquisition, but stale uncommitted work stops before candidate/index/package/audit work.
- Immutable finalization/audit receipts make repeated alarm/request/container execution recoverable without trusting disposable local clones.
- Fresh audit remains a distinct trust boundary.
- Long-lived GitHub App authority is isolated and ordinary runtime permissions exclude Administration/Workflows.
- Webhooks accelerate observation but do not own state.
- Per-write redundant postpublication full validation is removed from normal delivery.
- Project orientation prose is not an atomic-memory dual-write requirement.

## Alternatives considered

### Keep GitHub Actions as primary executor
Rejected: runner latency/workflow plumbing remains in the interactive path and duplicates the hosted runtime.

### Add Cloudflare Workflows on top of the repository Durable Object
Rejected for Phase 2.6 v1. The DO already owns serialized lane, durable operation state, finalizer lifecycle, audit handoff, cancellation, publication authorization, and crash reconciliation. A second Workflow state machine duplicates state/retry ownership without a distinct singleton-v1 invariant.

### Assume one Durable Object prevents all operation races
Rejected. Async external I/O opens interleaving opportunities. Explicit durable phase/generation claims and compare-before-accept logic are required.

### Hold `blockConcurrencyWhile()` across external I/O
Rejected. It turns slow provider I/O into a DO-wide lock, reduces availability/throughput, and has runtime timeout/reset risk. Local transactional state claims provide the required linearization instead.

### Hard-code canonical branch as `main`
Rejected. Core operates on a named branch and the memory semantics do not require a literal branch name. Hosted Runethread binds an explicit canonical ref and revalidates branch changes.

### Separate Free and paid GitHub implementations
Rejected: branch protection availability should not fork mutation semantics.

### Put hosted code in Core
Rejected: provider runtime/deployment/credentials must not contaminate deterministic memory compatibility surface.

### Store App key in user workflows or public API Worker
Rejected: canonical publisher authority belongs behind a private internal capability boundary.

### Rewrite MemoryService for Workers
Rejected: creates a second mutation engine solely to avoid Containers.

### Add a hosted complete idempotency database solely to reject stale work before source acquisition
Rejected: Git/Core remains canonical committed-idempotency authority; duplicating all history adds another recovery authority for a performance optimization.

### Re-run ApplyMutation blindly after lost finalizer response
Rejected: local candidate history can falsely look committed. Exact receipt or canonical reset is required.

### Let auditor publish directly
Rejected: races cancellation/suspension/maintenance and splits publication authority.

### Treat webhook payload as canonical truth
Rejected: duplicate/delayed/out-of-order delivery would create false state changes.

### Merge finalizer and auditor
Rejected: removes intentional fresh reduced-privilege trust boundary.

### Make project current-state prose part of every mutation
Rejected: creates semantic dual-write requirement.

## Verification

Implementation evidence must demonstrate at least:

1. one hosted path works for private GitHub Free and protected paid repositories;
2. provider-specific code remains outside Core and pins exact Core/runtime/container identities;
3. caller auth + repository/App/canonical-ref authorization prevents guessed identifiers from granting mutation access;
4. default/canonical-branch rename/change/deletion/transfer is detected and fails closed instead of silently redirecting publication;
5. runtime App baseline excludes Administration/Workflows and role tokens are repository/permission narrowed;
6. public API/finalizer/auditor never receive long-lived App private key/direct publication capability;
7. one repository DO is sole lane/operation/publication authority and different repositories progress independently;
8. no Cloudflare Workflow is required for v1 correctness/recovery;
9. DO SQLite + alarms resume accepted active operation after API response loss, eviction/restart, and transient failure without client traffic;
10. accepted response is not returned before durable operation state and a recoverable alarm schedule are established; exact resubmission/status/recovery repairs missing schedules;
11. alarm/request/cancel/webhook interleaving cannot duplicate or reorder authoritative external actions because each action is atomically phase/generation-claimed;
12. external result from stale/obsolete generation cannot advance current operation;
13. `blockConcurrencyWhile()` is not held across normal GitHub/R2/Container I/O;
14. alarm driver explicitly reschedules prolonged retryable failure and cannot release lane without evidence reconciliation;
15. hosted attempt identity binds repository/canonical-ref/Core-idempotency/request digest so byte-different envelopes do not alias accidentally;
16. stale classification that could overlap committed retry runs canonical Core/Git idempotency lookup first;
17. stale uncommitted work stops before candidate/Index/package/audit even when cold source acquisition was required;
18. `NO_OP` executes Core validation/idempotency/revision checks and skips candidate/audit/publication;
19. cold finalizer performs no more than one source clone/fetch and preserves historical idempotency evidence;
20. warm clone reuse/Git execution are hardened against stale/dirty/repository-controlled execution surfaces;
21. every fresh finalization starts from directly observed bound canonical ref, never surviving unpromoted candidate history;
22. after ApplyMutation, candidate evidence is persisted and immutable finalization receipt is created last before DO accepts result;
23. overlapping retries cannot select two authoritative finalization results; stale-generation outputs are nonauthoritative;
24. lost finalizer response with local C cannot cause false ALREADY_COMMITTED;
25. receipt/evidence mismatch/missing evidence fails closed;
26. ApplyMutation constructs exact C, writes Index v2 once, hard-validates, and leaves remote canonical ref at H0;
27. canonical pre-mutation trust/validation/compatibility failure suspends/reconciles lane while request-local mutation failure does not unnecessarily block unrelated work;
28. private request/candidate/audit objects are digest-verified, no-overwrite, short-retention, and absent from ordinary DO/log/status plaintext;
29. candidate packaging is complete under selected clone mode and cannot omit required promisor objects;
30. fresh auditor verifies exact C/hard validation/strict Index v2/scope without repair/publication authority;
31. auditor retries are generation-bound/idempotent and one immutable audit result is authoritative;
32. deterministic audit failure cannot release lane without durable suspension/reconciliation;
33. only repository DO can transition AUDITED -> PUBLISHING after cancellation/lane/evidence/auth/canonical-ref checks;
34. cancellation-vs-publication race is resolved by one atomic DO state transition;
35. publisher requests bind exact repository/canonical-ref/attempt/generation/audit/H0/C and publication requires ref == H0;
36. duplicate publisher calls/response loss resolve from exact ref/evidence without duplicate semantic mutation;
37. clone-free publication is used only if exact C identity is proven; exact-push fallback works otherwise;
38. successful publication gets only cheap exact-ref confirmation;
39. duplicate/delayed/out-of-order/missed webhooks remain safe because direct bound-ref reconciliation owns truth;
40. out-of-band movement enters reconciliation before later writes;
41. App uninstall/repository removal/permission loss fails closed;
42. control-plane/canonical-ref-binding barriers invalidate incompatible prepared work;
43. non-atomic hosted deployment cannot mix incompatible semantics inside one operation;
44. resource/quota/provider failure preserves canonical Git and is distinguishable from deterministic failure;
45. project orientation files are not silently dual-written;
46. local/offline MemoryService remains usable without Cloudflare and Core imports no hosted provider dependencies;
47. push-on-every-normal-memory Actions validation is removed through managed rollout while distinct recovery/migration/health checks remain possible;
48. Phase 3 MCP can expose same lifecycle without canonical-format change;
49. measurements separate cold/warm acquisition, bytes, idempotency/stale preflight, finalization, packaging, audit, publication, alarm/retry, and provider startup.