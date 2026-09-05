# ADR-014: Cloud-hosted Phase 2.6 memory-delivery control plane

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-012 and ADR-013 where they describe the initial GitHub-Actions-backed execution profile

## Context

ADR-012 and ADR-013 established the durable Phase 2.6 invariants: a semantic caller submits a structured MemoryService-compatible operation; deterministic Core constructs a noncanonical candidate; a fresh independent audit binds to the exact candidate; publication is an exact expected-revision fast-forward; and each canonical memory repository has one logical mutation-delivery lane with stale work parked as `NEEDS_REPREPARE` rather than silently rebased.

Their first implementation profile assumed GitHub Actions would be the initial execution adapter and that a managed GitHub ruleset/branch policy would prevent ordinary writers from bypassing the audited path. Pre-implementation verification invalidated two assumptions in that profile:

1. private GitHub repositories on GitHub Free cannot use the required private-repository ruleset/branch-protection features, while Runethread personal memory repositories are intentionally private; and
2. making GitHub Actions the primary finalization/audit runtime would retain runner-startup latency, duplicate execution plumbing, and a workflow-specific transitional state model that the project already intended to replace with hosted execution.

The same review found that Cloudflare now provides primitives that map closely to the accepted provider-independent design: Workers for API/internal services, Durable Objects for per-repository stateful coordination, Workflows for durable multi-step execution, and Containers for the real Linux/Go/Git Runethread runtime. Containers require Workers Paid; this is an infrastructure requirement of the hosted Runethread service account, not an end-user Cloudflare-plan requirement.

The current `MemoryService.ApplyMutation` already creates an isolated transaction, regenerates Index v2, hard-validates, creates the exact mutation commit, and only fast-forwards the caller's local canonical branch. It does not push GitHub. A hosted finalizer can therefore reuse today's deterministic MemoryService implementation in an ephemeral clone without making remote `main` canonical prematurely.

Project `current-state.md` files are orientation/current-state views, not the authoritative atomic-memory transaction. They must not force Phase 2.6 to become a generic prose dual-write transaction.

## Decision

Phase 2.6 uses a **Cloudflare-hosted remote memory-delivery control plane** as its primary hosted execution profile. GitHub remains the user-owned canonical Git store. GitHub Actions remains useful for independent repository-health, recovery, migration, and control-plane validation, but is not the normal interactive mutation executor or queue.

This ADR supersedes only conflicting GitHub-Actions-first implementation details of ADR-012/ADR-013. Their candidate-before-canonical, independent-audit, exact-revision publication, idempotency, stale-reprepare, fail-closed behavior, and per-repository serialization invariants remain accepted unless explicitly amended below.

### Hosted component and release boundary

Provider-specific hosted implementation lives outside `runethread/core`. The target component/repository is `runethread/hosted` unless implementation preflight finds a concrete topology conflict before code is created.

`runethread/core` continues to own deterministic memory semantics and local interfaces. Hosted execution consumes Core through an immutable verified runtime/release or an exact explicitly verified development build. Production hosted mutation MUST NOT execute floating public `main`.

A hosted delivery release has its own explicit release/protocol identity and records the exact Core/runtime binary digest plus Container image/delivery identity used for a mutation. Finalizer and auditor use the same pinned deterministic Runethread runtime semantics but different fresh privilege contexts.

Cloudflare Worker activation and Container rollout are not assumed atomic. Breaking Worker/Workflow/Container protocol changes are control-plane barriers: admission must be drained/placed in maintenance or a versioned blue/green deployment must keep incompatible execution paths separate until in-flight work terminates. One operation MUST NOT silently cross incompatible hosted delivery/runtime semantics.

### One hosted architecture for GitHub Free and paid users

Runethread does not maintain separate Free-versus-paid mutation implementations.

```text
semantic client
      |
authenticated Runethread delivery API
      |
repository-runtime coordinator
      |
durable operation Workflow
      |
Runethread finalizer -> exact candidate -> fresh auditor
      |
private GitHub gateway/publisher
      |
user-owned private GitHub repository
```

GitHub paid branch/ruleset protection is optional defense-in-depth. It is not a correctness dependency. On an unprotected private repository, unexpected out-of-band `main` movement is detected by exact ref observation and puts the repository into fail-closed reconciliation before later hosted writes.

### Component and credential ownership

The hosted profile has five distinct responsibilities.

1. **Public Worker/API edge** — authenticates/authorizes callers, validates request envelopes and resource limits, and routes to the repository coordinator. It implements no memory semantics and MUST NOT hold the GitHub App private key.
2. **Per-repository Durable Object / repository runtime** — the sole hosted lane authority. It owns admission/serialization, lane state, operation metadata/status, active Workflow identity, suspension/maintenance/reconciliation, last accepted canonical revision, and final publication authorization. It also manages its attached finalizer Container rather than introducing another per-repository coordinator solely for Container lifecycle.
3. **Cloudflare Workflow** — owns retryable execution checkpoints for one admitted operation. It does not own the repository lane and cannot independently authorize canonical publication.
4. **Container runtime** — runs the real Runethread Go runtime and Git. The repository runtime's attached Container finalizes; a separate fresh reduced-privilege Container audits.
5. **Private internal GitHub gateway/publisher** — holds the long-lived GitHub App private key and exposes only narrow internal repository-read/token/publication operations through a private capability boundary such as a Service Binding.

The Durable Object answers *which operation may use this repository lane and whether publication is currently permitted*. The Workflow answers *which retryable execution step this operation has durably reached*. They MUST NOT become two independently mutable authorities for the same transition.

### Canonical and hosted state ownership

Canonical semantic memory, project data, generated indexes, and committed mutation/idempotency evidence remain in the user's Git repository under ADR-001 through ADR-004.

The hosted coordinator may persist only operational delivery metadata such as:

- immutable provider repository identity and authorized App installation binding;
- lane state (`OPEN`, `SUSPENDED`, `MAINTENANCE`, `RECONCILIATION_REQUIRED` or equivalent);
- operation/idempotency identity;
- opaque request reference and cryptographic request digest;
- expected/base revision;
- operation state and deterministic Workflow identity;
- candidate/audit evidence references and digests;
- final committed revision or failure outcome.

This hosted control state is authoritative for the live hosted lane, but it is not canonical memory. A committed memory outcome must remain confirmable from exact Git history and mutation metadata. Hosted state loss MUST NOT make committed history ambiguous.

### Private request/evidence persistence

Full private memory Markdown/JSON or other sealed request bodies MUST NOT be stored in ordinary Durable Object operation metadata, Workflow parameters, Workflow step results, logs, or client-visible status merely because those provider stores are convenient.

The normal hosted envelope stores the private sealed request in a private content-addressed object and passes only an opaque object reference plus digest/size/type metadata through Durable Object and Workflow state. This also prevents Workflow event/result size and retention limits from becoming semantic memory limits.

Candidate evidence follows the same rule. Request/candidate objects MUST use content-addressed keys, cryptographic digest verification, and create-if-absent/no-overwrite semantics (for example conditional object creation) so provider last-writer-wins behavior cannot silently mutate evidence under an existing identity.

Retention and garbage collection for request/candidate plaintext are explicit, short by default, and tested. Integrity incidents may retain evidence deliberately, but not through indefinite silent retention.

### Authentication and repository authorization

The hosted API requires an authenticated Runethread principal plus an explicit authorization binding to an installed GitHub App repository. A caller MUST NOT gain mutation authority merely by supplying or guessing a repository identifier.

Repository identity for coordination and authorization uses immutable provider identity where available, not only mutable owner/name. Repository rename/transfer, App uninstall, repository removal from an installation, or App permission changes require authorization revalidation and fail closed when access is no longer valid.

Dogfood may begin with a deliberately narrow operator/service authentication mechanism, but authentication, repository authorization, and memory semantics remain separate so Phase 3 MCP/OAuth can replace client transport without changing mutation rules.

### GitHub App permissions

The runtime GitHub App requests only the minimum repository permissions needed by the hosted path. The expected baseline is **Contents** access plus Metadata's implicit/read access. It MUST NOT request Administration or Workflows permission merely for ordinary memory delivery.

Short-lived installation tokens are scoped down per repository and role:

- finalizer/auditor: read-only Contents when Git access is required;
- publisher: Contents-write only for the exact publication operation;
- semantic clients/public API: no GitHub canonical-write token.

If a future required GitHub endpoint demonstrably needs another permission, that permission addition is a reviewed security/control-plane change. Lack of Workflows permission is an intentional defense against the runtime publisher modifying `.github/workflows/**` during ordinary data-plane publication.

### Admission and whole-workflow serialization

The API accepts one complete sealed MemoryService-compatible operation or a supported immutable payload reference. It does not watch a caller assemble request files across Git commits.

Phase 2.6 v1 permits concurrent semantic preparation but serializes the **whole heavy finalization/audit/publication workflow per repository**. Different repositories execute independently.

After a known hosted commit advances canonical state, queued operations whose expected revision is now stale may be parked before Container startup **only when hosted state already proves they are distinct operations and not an ambiguous lost-response retry**. The optimization MUST NOT replace ADR-003's ordering rule.

When hosted state is absent, inconsistent, recovered, or otherwise unable to prove that distinction, exact canonical committed idempotency evidence is checked before classifying an expected-revision mismatch as `NEEDS_REPREPARE`. Core's `FindAppliedOperation`/request-fingerprint semantics remain authoritative for committed retry/conflict determination.

Provider code may calculate a transport digest over sealed request bytes for object/evidence binding and exact-envelope deduplication. It MUST NOT invent a second semantic idempotency fingerprint implementation in TypeScript or hosted glue. Same idempotency key with semantically different request handling remains a Core decision unless the exact Core fingerprint logic is invoked.

`NO_OP` is a successful terminal result with no candidate/audit/publication, but it still passes through Core's ordinary request validation, committed-idempotency lookup, repository revision check, and hard repository validation. The coordinator MUST NOT infer semantic `NO_OP` solely from the operation label.

### Crash-safe Workflow start protocol

Workflow instance identity is deterministic for one hosted operation attempt, derived from stable immutable repository identity plus operation identity (using a bounded opaque/hash representation suitable for provider ID limits). Random Workflow IDs are not used for authoritative recovery.

Durable Object storage and Workflow creation are not assumed to be one atomic transaction. Admission therefore uses an idempotent reconciliation protocol:

1. the Durable Object durably records the intended active operation and deterministic Workflow instance ID before releasing the admission transaction;
2. it queries that Workflow ID;
3. if the instance exists, it attaches/reconciles status rather than creating another;
4. if it does not exist, it creates exactly that ID;
5. a crash or error at any point is recovered by repeating `get-or-create/reconcile` for the same deterministic ID;
6. the lane is not released until Workflow/Git evidence establishes a safe terminal outcome.

A duplicate admission of the same operation cannot start a second heavy Workflow. An orphan/phantom state after a crash is reconciled through deterministic instance identity rather than repaired by guessing.

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

### Finalization with existing MemoryService

The finalizer MUST use existing deterministic MemoryService/Core rather than a provider-specific mutation implementation.

For a cold executor, the target is at most one GitHub source clone/fetch for finalization. Reachable commit history required by `FindAppliedOperation` must be preserved. A shallow clone MUST NOT hide historical idempotency evidence. A blobless partial clone may be used only after integration tests prove current checkout, validation, transaction, idempotency lookup, and candidate packaging remain correct.

A warm clone MAY be reused only as an untrusted cache after exact fetch and hard restoration to clean canonical state. Correctness never depends on warm disk. Repository-controlled Git hooks, submodules, external filters, credential helpers, unsafe config/includes, and related execution surfaces must be disabled/refused under an explicit hosted Git hardening policy.

Finalization first lets Core resolve committed idempotency before stale revision. For a runnable non-noop operation it invokes `ApplyMutation` once. `ApplyMutation` performs semantic mutation, one Index v2 write, hard validation, mutation commit, and local-only fast-forward. Remote GitHub `main` remains `H0`.

A successful prepublication candidate `C` is bound to that exact finalization attempt. If a step is retried from scratch and produces a different commit, it is a new exact candidate and requires new candidate evidence and audit.

### Candidate evidence and transport

The exact candidate is exported as private immutable/content-addressed evidence bound at least to:

- repository and operation/idempotency identity;
- expected/base revision `H0`;
- exact candidate commit `C` and tree;
- sealed-request transport digest plus authoritative Core mutation metadata/request fingerprint when available;
- exact runtime/container-image/hosted-delivery identity and pinned contract identity;
- package digest and package format version.

Candidate transport optimizes **total transferred bytes/runtime and exactness**, not a ceremonial zero-clone count. It may use a full candidate package or an exact Git-native delta/object package over `H0`. A delta package may require the fresh auditor to obtain exact `H0` through a bounded read-only fetch/clone or separately verified immutable cache.

Partial-clone/promisor state MUST NOT produce candidate evidence with missing required objects. Completeness is proven before the auditor trusts the package.

### Independent audit

A fresh auditor executes in a separate reduced-privilege environment with no publication credential. It reconstructs exact `C` from immutable candidate evidence and, when needed, a bounded exact-base read.

It verifies at least:

- package digest/manifest binding and package completeness;
- candidate commit/tree/parent identity;
- operation/idempotency/request bindings and Core mutation metadata;
- runtime/delivery/contract identity;
- trust/control-plane state;
- hard repository validation;
- strict `index --check` freshness;
- expected mutation diff/scope and absence of unauthorized control-plane/unrelated user-data changes.

The auditor is observational: no repair/index-write step and no canonical publication credential. This independence protects against workspace/candidate/transport/privilege/race errors, not deterministic bugs shared by the same Core implementation.

### DO-mediated publication authorization and cancellation boundary

A fresh audit does **not** itself grant the Workflow permission to publish. After audit succeeds, the Workflow records/reports the exact audit evidence and returns control to the repository-runtime Durable Object.

The Durable Object is the sole hosted component allowed to authorize `AUDITED -> PUBLISHING`. In one serialized state transition it verifies:

- the operation is still the active lane owner;
- lane state still permits publication (`OPEN`, not suspended/maintenance/reconciliation);
- cancellation has not already won;
- audit evidence binds exact authorized repository/operation/`H0`/`C`/request/runtime identities;
- no incompatible hosted/control-plane barrier became active.

Only after that transition wins may the private GitHub gateway receive a publication request/capability. The gateway requires the exact repository ID, operation ID, `H0`, `C`, audit-evidence identity/digest, and delivery/runtime identity that the Durable Object authorized. A Workflow or auditor cannot call publication as an independently authoritative actor.

Cancellation is permitted only before the Durable Object successfully transitions the active operation to `PUBLISHING`. Once `PUBLISHING` wins, cancellation is no longer a correctness mechanism; the system resolves the exact CAS outcome as committed, not committed/currently H0, or unexpected/reconciliation.

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

Before heavy execution and again at publication, hosted Runethread compares direct observed canonical state to accepted/expected state. Unexpected movement not explained by the one active hosted workflow or authorized maintenance/recovery yields `RECONCILIATION_REQUIRED`.

Recovery independently inspects the exact out-of-band revision and either adopts it after applicable trust/validation/index/repository-health checks establish a new accepted canonical base, or repairs/restores through an explicit authorized recovery procedure. Owner permission alone does not make arbitrary movement audited.

### Control-plane barriers

Contract/schema/trust/repository-format/bootstrap/migration/managed-delivery changes remain exclusive barriers. The repository coordinator represents the barrier as `MAINTENANCE` or equivalent; new data-plane publication is refused/parked and pre-barrier prepared operations cannot publish under post-barrier semantics without re-preparation.

Installing or materially changing Phase 2.6 is itself a barrier and is rolled out through the full Core/hosted release/downstream process rather than through the data-plane path it is creating.

### Resource, abuse, and privacy limits

The hosted adapter enforces explicit limits for at least authenticated request rate/queued work, inline request size, repository/current-tree size, candidate evidence size/retention, Container CPU/memory/disk/wall time, Workflow retry/lifetime, and log/event size.

Large legitimate requests use private immutable referenced payloads rather than inheriting provider event-size caps as semantic memory limits. Resource/quota/provider failure leaves canonical Git unchanged and remains distinguishable from deterministic semantic/finalization failure.

Hosted private-memory processing means Cloudflare infrastructure temporarily processes authorized plaintext repository/candidate data. Public rollout therefore requires an explicit data-handling/threat-model review covering private object storage, Workflow/DO metadata, logs, credential isolation, retention/deletion, and provider access boundaries.

### Project current-state views are projections

Atomic memory mutation does not become a generic writer for `projects/<slug>/current-state.md` or other orientation prose. Canonical atomic memories and authoritative project source repositories remain sources of truth. Project overview/current-state files are orientation/materialized views whose future refresh mechanism is separate and has explicit freshness semantics.

### MCP relationship

Phase 2.6 establishes provider-neutral hosted delivery operations before MCP transport. Phase 3 is a thin MCP transport over MemoryService and this hosted delivery lifecycle. Local/offline Runethread continues to use MemoryService directly without Cloudflare.

## Consequences

- GitHub Free and paid users share one hosted mutation architecture; paid protection is defense-in-depth.
- Hosted Runethread requires Workers Paid for Containers; end users do not need Cloudflare accounts/plans.
- Provider-specific code/release lifecycle stays outside Core.
- One repository-runtime Durable Object owns the lane and attached finalizer Container.
- One deterministic Workflow identity per operation attempt makes DO/Workflow startup recoverable without duplicate heavy execution.
- Workflow/DO state stores opaque request/evidence references, not private memory bodies.
- Existing MemoryService remains the single mutation engine and preserves ADR-003 idempotency-before-stale semantics.
- Heavy workflow serialization rejects known-stale queued work before expensive execution without misclassifying ambiguous exact retries.
- Fresh audit remains a distinct trust boundary.
- The DO, not the Workflow/auditor, authorizes the publication critical section and defines cancellation ordering.
- Long-lived GitHub App authority is isolated; baseline App permissions exclude Administration/Workflows.
- Candidate/request evidence uses content-addressed, digest-verified, no-overwrite storage semantics.
- Webhooks accelerate observation but cannot mutate accepted canonical state without a direct ref read.
- Publisher can be clone-free only if exact object identity is proven; exact candidate push remains the safe fallback.
- Per-write post-publication full validation is removed from normal synchronous delivery.
- Breaking hosted deployments require version/barrier handling because provider rollout is not assumed atomic.
- Project orientation prose is not an atomic-memory dual-write requirement.

## Alternatives considered

### Keep GitHub Actions as primary executor
Rejected: runner latency/workflow plumbing would remain in the interactive critical path and would be replaced by the hosted path anyway.

### Separate Free and paid GitHub implementations
Rejected: branch protection availability should not fork mutation semantics.

### Put hosted code in Core
Rejected: provider runtime/deployment/credentials must not contaminate the deterministic memory compatibility surface.

### Store the App key in user workflows or public API Worker
Rejected: canonical publisher authority belongs behind a private internal capability boundary.

### Rewrite MemoryService for Workers
Rejected: it creates a second mutation engine solely to avoid Containers.

### Use persistent cloud Git as canonical
Rejected: the user-owned Git repository remains canonical.

### Let Workflow publish directly after audit
Rejected: it races lane suspension/cancellation/maintenance and splits publication authority from the repository coordinator.

### Store full sealed requests in Workflow event/state
Rejected: it unnecessarily retains private memory in provider execution history and couples semantic request size to Workflow payload/state limits.

### Use random Workflow IDs
Rejected: a crash between DO state and Workflow creation would make duplicate/orphan recovery ambiguous.

### Treat webhook payloads as canonical ref truth
Rejected: duplicate/delayed/out-of-order delivery would create false reconciliation or stale state.

### Require zero GitHub reads after finalization
Rejected: minimize total exact transferred work rather than an arbitrary clone counter.

### Run multiple heavy mutation workflows per repo in v1
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
5. public API/finalizer/auditor never receive the long-lived App private key or canonical-write authority;
6. one repository-runtime DO is the sole lane/publication authority and different repositories progress independently;
7. deterministic Workflow IDs plus get-or-create reconciliation survive crashes before/after Workflow creation without duplicate heavy workflows or phantom lane release;
8. callback loss cannot release the lane without Workflow/Git reconciliation;
9. known distinct stale queued work is parked before Container startup, while ambiguous recovery checks committed idempotency before stale;
10. provider transport digests do not replace Core's semantic idempotency fingerprint/conflict rules;
11. `NO_OP` still executes Core's request/revision/idempotency/repository validation but skips candidate/audit/publication;
12. a cold finalizer performs no more than one source clone/fetch and preserves historical idempotency evidence;
13. warm clone reuse and Git execution are hardened against dirty/stale/repository-controlled execution surfaces;
14. `ApplyMutation` constructs exact `C`, writes Index v2 once, hard-validates, and leaves remote `main` at `H0`;
15. private request/candidate objects are content-addressed, digest-verified, no-overwrite, short-retention, and absent from ordinary Workflow/DO/log/client plaintext;
16. candidate packaging is complete under the selected clone mode and cannot omit required promisor objects;
17. a fresh auditor verifies exact `C`, hard validation, strict Index v2 freshness, and scope without repair or publication authority;
18. audit failure leaves Git canonical state unchanged and suspends the lane;
19. after audit, only the DO can transition `AUDITED -> PUBLISHING`, and cancellation/maintenance/suspension races are resolved before publisher authority is granted;
20. publisher requests are exact-operation/audit-bound and publication requires remote `main == H0` before moving only to audited `C`;
21. clone-free Git-object publication is used only if exact object identity is proven; otherwise exact-candidate push fallback works;
22. CAS response loss is resolved from exact ref/operation evidence without duplicate semantic mutation;
23. successful publication gets only cheap exact-ref confirmation in the normal synchronous path;
24. duplicate/delayed/out-of-order/missed signed webhooks remain safe because they trigger direct ref reconciliation rather than directly mutating accepted state;
25. unexpected/out-of-band movement enters reconciliation before later hosted writes;
26. App uninstall/repository removal/permission loss fails closed and requires authorization revalidation;
27. control-plane barriers prevent pre-barrier work from publishing under changed semantics;
28. non-atomic Worker/Workflow/Container deployment cannot mix incompatible semantics inside one operation;
29. resource/quota/provider failures preserve canonical Git state and are distinguishable from deterministic failures;
30. project orientation files are not silently dual-written by hosted provider code;
31. local/offline MemoryService remains usable without Cloudflare and Core imports no hosted provider dependencies;
32. push-on-every-normal-memory GitHub Actions validation is removed through proper managed rollout while distinct recovery/migration/health checks remain possible;
33. Phase 3 MCP can expose the same delivery lifecycle without changing canonical memory format or mutation semantics;
34. end-to-end measurements separate cold/warm source acquisition, bytes transferred, finalization, packaging, audit, publication, and provider startup so optimization is evidence-driven.