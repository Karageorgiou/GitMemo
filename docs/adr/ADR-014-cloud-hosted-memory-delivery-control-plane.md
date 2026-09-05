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

The same review found that Cloudflare now provides primitives that map closely to the already-accepted provider-independent design: Workers for the API edge and internal services, Durable Objects for per-repository stateful coordination, Workflows for durable multi-step execution, and Containers for the real Linux/Go/Git Runethread runtime. Containers require the Workers Paid plan; this is an infrastructure requirement of the hosted Runethread service account, not an end-user Cloudflare-plan requirement.

The review also confirmed that the current `MemoryService.ApplyMutation` already creates an isolated transaction, regenerates Index v2, hard-validates, creates the exact mutation commit, and only fast-forwards the caller's *local* canonical branch. It does not push GitHub. A hosted finalizer can therefore reuse today's deterministic MemoryService implementation in an ephemeral clone without making remote `main` canonical prematurely.

Finally, project `current-state.md` files are orientation/current-state views, not the authoritative atomic-memory transaction. They must not force Phase 2.6 to become a generic prose dual-write transaction. Their refresh can be designed as a separate projection/materialized-view concern.

## Decision

Phase 2.6 uses a **Cloudflare-hosted remote memory-delivery control plane** as its primary hosted execution profile. GitHub remains the user-owned canonical Git store. GitHub Actions is retained for independent repository-health, recovery, migration, and control-plane validation where useful, but is not the normal interactive memory-write executor or queue.

This ADR supersedes only the conflicting GitHub-Actions-first implementation details of ADR-012 and ADR-013. Their semantic/canonical invariants remain accepted unless this ADR explicitly changes them.

### Hosted component and release boundary

Cloudflare/provider-specific hosted implementation lives outside `runethread/core`. The target component/repository is `runethread/hosted` unless implementation preflight identifies a concrete repository-topology conflict before code is created.

`runethread/core` continues to own deterministic memory semantics and local interfaces. The hosted component consumes Core through an immutable verified Runethread runtime/release or exact explicitly verified development build. It MUST NOT execute floating public `main` as the production memory engine.

A hosted delivery release has its own explicit release/protocol identity and records the exact Core/runtime binary digest plus Container image/delivery identity used for a mutation. Finalizer and auditor use the same pinned deterministic Runethread runtime/image identity but different fresh privilege contexts.

Cloudflare Worker activation and Container image rollout are not treated as one atomic deployment. Breaking Worker/Workflow/container protocol changes are control-plane barriers: admission must be drained/placed in maintenance or a versioned blue/green deployment must keep old and new execution paths separate until in-flight work terminates. Compatible rollouts must deliberately support the platform's temporary old/new Worker/container overlap. One operation MUST NOT silently cross incompatible hosted delivery/runtime semantics.

### One hosted architecture for GitHub Free and paid users

Runethread does not maintain separate Free-versus-paid memory-delivery implementations.

The normal hosted path is the same for every supported private GitHub memory repository:

```text
semantic client
      |
authenticated Runethread delivery API
      |
Cloudflare per-repository coordinator
      |
durable operation workflow
      |
Runethread finalizer + independent auditor
      |
least-privilege Runethread GitHub App publisher
      |
user-owned private GitHub repository
```

GitHub paid branch/ruleset protection is optional defense-in-depth. When available, it SHOULD be configured to prevent accidental or unauthorized direct canonical movement. It is not a correctness dependency of the hosted Runethread protocol.

On an unprotected private repository, out-of-band `main` movement is detectable because the hosted coordinator retains the last accepted canonical revision and every mutation still requires exact expected-revision publication. Unexpected canonical movement puts the repository into a fail-closed reconciliation/suspension state before later hosted writes are allowed. The repository owner retains ultimate administrative authority over their own Git repository.

### Component and credential ownership

The Phase 2.6 hosted profile has five distinct responsibilities.

1. **Public Worker/API edge** — authenticates/authorizes requests, resolves the target authorized repository, enforces request/rate/resource limits, and routes to the repository coordinator. It does not implement memory semantics and MUST NOT hold the GitHub App private key.
2. **Per-repository Durable Object / repository runtime** — owns hosted lane state, admission/serialization, operation identity/status mapping, suspension/maintenance/reconciliation state, the active Workflow identity, and the last accepted canonical revision for that repository. It also manages the repository's attached finalizer Container through Cloudflare's Durable-Object/Container relationship rather than introducing a second per-repository coordinator solely for Container lifecycle.
3. **Cloudflare Workflow** — owns resumable execution checkpoints and retries for one admitted operation. Workflow steps are individually retryable and MUST be idempotent. Workflow execution state does not replace canonical Git evidence for whether a memory mutation committed.
4. **Container runtime** — runs the real Runethread Go binary/Core and Git in Linux. The repository runtime's attached Container performs finalization; a separate fresh Container instance performs independent audit. Core remains provider-independent and MUST NOT import Cloudflare or GitHub SDK dependencies merely because the hosted adapter uses those providers.
5. **Private internal GitHub gateway/publisher service** — holds the GitHub App private key and exposes only narrow internal repository-auth/publication operations through a Cloudflare Service Binding or equivalent non-public capability boundary. The public API Worker and semantic clients cannot read the App private key or mint arbitrary installation credentials.

The coordinator and Workflow have deliberately different roles: the Durable Object answers *which operation may use this repository lane and what hosted lane state applies*; the Workflow answers *which retryable execution step this one operation has durably reached*. Their state models must not become two independently mutable sources of truth for the same transition.

The Durable Object records the one active Workflow identity. It MUST NOT release the repository lane merely because a completion callback was lost or delayed. Recovery must query/reconcile the active operation/Workflow state and exact Git evidence before another heavy hosted mutation is admitted.

### Canonical and hosted state ownership

Canonical semantic memory, project data, generated indexes, and committed mutation/idempotency evidence remain in the user's Git repository as governed by ADR-001 through ADR-004.

The hosted coordinator may persist operational delivery state such as:

- immutable provider repository identity and authorized GitHub App installation binding;
- lane state such as `OPEN`, `SUSPENDED`, `MAINTENANCE`, or `RECONCILIATION_REQUIRED`;
- operation identity/idempotency identity;
- expected/base revision;
- operation state and active Workflow identity;
- candidate identity when one exists;
- exact audit evidence identity;
- final committed revision or failure outcome.

This is authoritative hosted control state while the service is operating, but it is not canonical memory. A committed memory outcome must always be recoverable/confirmable from exact Git history and mutation metadata. Loss of hosted pending-state storage must not cause committed canonical history to become ambiguous; resubmitting the same sealed operation/idempotency identity remains the recovery fallback.

This explicitly replaces ADR-012/ADR-013's GitHub-workflow-run-as-transitional-state implementation profile. It does not change the canonical memory schema or repository format solely to store delivery state.

### Authentication and repository authorization

The hosted API requires an authenticated Runethread principal and an explicit authorization binding to a GitHub App installation/repository. A caller MUST NOT gain mutation authority merely by supplying an arbitrary repository owner/name or numeric repository ID.

The dogfood/Phase 2.6 implementation may start with a deliberately narrow operator/service authentication mechanism, but the hosted API must keep authentication, repository authorization, and memory semantics as separate boundaries so Phase 3 MCP/OAuth integration can replace the client-auth transport without changing canonical storage or mutation rules.

Repository identity for coordination and authorization is based on immutable provider identity where available, not only mutable owner/name strings. Repository rename/transfer and App-installation changes require revalidation of the authorization binding.

### Admission, whole-workflow serialization, and operation states

The API accepts one complete sealed MemoryService-compatible operation or an explicitly supported immutable payload reference. A caller must not construct a watched request tree through multiple Git commits.

Phase 2.6 v1 allows concurrent semantic preparation but serializes the **heavy hosted mutation workflow** per repository, not merely the final compare-and-swap. Only one operation per repository may occupy finalization/audit/publication execution at a time.

This is an efficiency rule layered on top of ADR-004, not a replacement for exact revision checks. After one operation commits and advances canonical state, queued operations whose expected revision is now stale are parked as `NEEDS_REPREPARE` **before** a Container is started. This avoids knowingly paying to clone, rebuild Index v2, and audit candidates that cannot publish. Later independently-current operations may proceed after stale queued work is parked.

Strict FIFO remains an availability/fairness choice rather than a semantic correctness assumption. The scheduler may choose a deterministic queued operation order, but correctness never assumes arrival order. No hosted scheduler may semantically rewrite or automatically reinterpret stale work merely to keep the lane moving.

The hosted state machine must represent meanings equivalent to:

```text
ACCEPTED
QUEUED
FINALIZING
CANDIDATE_READY
AUDIT_PENDING
AUDITED
PUBLISHING
COMMITTED
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

`ALREADY_COMMITTED` is the hosted mapping of MemoryService's exact committed idempotent retry behavior. `NO_OP` is a successful terminal outcome with no candidate, audit, or publication step.

Exact persisted enum names may still be refined before a public API is frozen, but impossible combinations of independently mutable booleans are prohibited.

### Finalization with the existing MemoryService

The normal finalizer MUST use the existing deterministic MemoryService/Core implementation rather than a Cloudflare- or workflow-specific mutation reimplementation.

For a cold executor, the target is at most **one GitHub source clone/fetch for finalization** in the normal operation path. The clone MUST preserve reachable commit history required by `FindAppliedOperation` while avoiding eager historical blob transfer where Git partial-clone support is safe, for example a blobless partial clone. A shallow clone MUST NOT be used if it could hide historical idempotency evidence.

A warm container MAY reuse its existing clone as an untrusted cache by fetching and restoring a clean exact canonical state before the operation. Cache reuse must never be a correctness dependency; container disk is ephemeral and may disappear between requests. Untrusted repository content, Git configuration, hooks, submodules, filters, and credential helpers must be handled under an explicit hardened Git execution policy so repository data cannot obtain hosted service credentials or code-execution authority merely by being cloned.

The finalizer receives only repository-scoped read authority sufficient to clone/fetch the authorized memory repository. It MUST NOT receive the App private key or a general canonical-write token.

Finalization first checks exact Git state and committed idempotency evidence. If the request is already committed under the same idempotency identity, the operation returns `ALREADY_COMMITTED`. If the request is not already committed and current canonical state differs from the prepared expected revision, it becomes `NEEDS_REPREPARE` before expensive candidate construction.

For a runnable operation, the finalizer calls `ApplyMutation` in the isolated hosted clone. The local fast-forward performed by `ApplyMutation` yields exact candidate commit `C` inside that disposable clone; remote GitHub `main` remains at expected revision `H0`.

The finalizer MUST NOT redundantly run another Index v2 write or another equivalent repair pass after successful `ApplyMutation`. `ApplyMutation` already performs Index v2 generation and hard validation as part of candidate construction.

A prepublication candidate commit is evidence from one successful finalization attempt, not a value Phase 2.6 assumes can always be byte-for-byte reproduced by rerunning Git commit creation. The Workflow therefore persists/binds the exact candidate produced by the successful finalized step. If a finalization step never durably completed and is retried from scratch, a new exact candidate may be produced and must receive its own artifact/audit binding before publication.

### Exact candidate evidence and transport

After finalization, the exact candidate is exported as private immutable/content-addressed candidate evidence suitable for a fresh auditor and, if required, exact later publication. Private object storage such as R2 may hold this evidence, but the physical package format is an implementation detail subject to an early prototype.

The manifest must bind at least:

- operation identity and idempotency identity;
- expected/base revision `H0`;
- candidate commit `C` and candidate tree identity;
- request fingerprint;
- mutation metadata;
- exact Runethread runtime/container-image/hosted-delivery identity and pinned contract identity used for finalization;
- a cryptographic digest of the candidate evidence/package.

Candidate transport is optimized for **total transferred work and exactness**, not for a ceremonial zero-clone count. A full current-tree package can be wasteful for large repositories, while a small delta package requires the auditor to obtain exact base `H0`. Phase 2.6 therefore permits either:

- a self-contained full candidate package; or
- an exact candidate delta/object package over `H0`, with the auditor obtaining exact `H0` through a bounded read-only fetch/clone or a separately verified immutable cache.

The implementation should prototype a Git-native delta/object representation such as a bundle/pack over `H0 -> C`, but MUST prove that all required objects are available even when the finalizer used partial-clone/promisor mechanics. No missing promisor object may be discovered only after the auditor begins trusting the package.

The candidate evidence may contain private memory data. It therefore MUST be private, inaccessible to semantic clients by default, excluded from ordinary logs, encrypted in transit and at rest through the platform's supported storage controls, and retained only as long as required for audit, publication, retry/recovery evidence, or an integrity incident. Retention/garbage-collection behavior must be explicit and tested; integrity-incident retention must not become indefinite silent storage of private memory.

### Independent audit with minimum redundant transfer

A fresh auditor executes in a separate fresh container/environment with no canonical publication credential. It consumes the exact immutable candidate evidence and reconstructs/verifies exact `C`.

The auditor SHOULD avoid a redundant full-history clone. If the chosen candidate representation is a delta over `H0`, a bounded read-only fetch of exact `H0` is permitted when that is simpler/cheaper than transferring a full current-repository snapshot through object storage. Clone/fetch count is therefore a performance metric, not a correctness invariant.

The auditor uses the same deterministic Runethread implementation/runtime identity as the finalizer and verifies the exact candidate, including at minimum:

- artifact/package digest and manifest binding;
- candidate commit/tree identity and expected parent/base relationship;
- operation/idempotency/request-fingerprint mutation metadata;
- applicable trust/control-plane state;
- hard repository validation;
- strict `index --check` freshness/integrity;
- expected mutation diff/scope constraints;
- absence of unauthorized control-plane or unrelated user-data changes.

The auditor is observational: it MUST NOT run repair/index-write steps to make a candidate green and MUST NOT possess the publisher's canonical-write credential. If it requires a base `H0` fetch, it receives only repository-scoped read authority.

This independent audit remains intentionally limited: a deterministic bug shared by the same Runethread implementation can affect both finalizer and auditor. Independence protects against workspace contamination, candidate mix-ups, transport corruption, wrong-SHA publication, privilege mistakes, and environment/race errors; it is not a second semantic implementation.

### Publication and GitHub App authority

The Runethread GitHub App is installed only on repositories explicitly authorized by the user. Long-lived App private-key material lives only in the private internal GitHub gateway/publisher service and is never copied into user memory repositories, public API Workers, finalizer/auditor containers, or semantic clients.

The gateway mints repository-scoped/permission-narrowed short-lived installation credentials only for an authorized repository/operation. Read credentials may be passed to the finalizer/auditor when required. Canonical-write credentials SHOULD remain internal to the publisher gateway whenever the chosen GitHub API path permits it. If the exact-push fallback requires a dedicated publisher container, only that minimal privileged environment receives a short-lived repository-scoped write credential; it never receives semantic-client authority or the long-lived App key.

Only an independently audited candidate is eligible for publication.

Publication must preserve ADR-012's exact invariant:

```text
remote main = H0
candidate   = C

audit exact C
atomically require main == H0
fast-forward main H0 -> exact C
```

No force push, merge commit, squash rewrite, or last-writer-wins update is allowed.

The implementation SHOULD first prototype a clone-free publisher using GitHub's Git object/ref APIs. That optimization is accepted only if it can reproduce/upload the exact already-audited Git objects and make remote `main` point to the exact candidate commit `C`. GitHub's atomic `updateRefs` expected-old-object (`beforeOid`) semantics are an appropriate candidate for the final compare-and-swap boundary.

If the Git object API cannot preserve the exact candidate commit identity reliably, the fallback is a minimal privileged publication environment that imports the audited candidate evidence and pushes exact `C`; it still must not perform a new semantic mutation or reconstruct a different commit.

Because Phase 2.6 v1 serializes the heavy hosted workflow per repository, a canonical mismatch at the *active operation's* final publication boundary cannot be explained by another normal hosted operation in the same lane. Unless a known maintenance/recovery action accounts for the movement, it is treated as unexpected/out-of-band movement and drives the lane to `RECONCILIATION_REQUIRED`. By contrast, queued operations that become stale after a preceding known hosted commit are ordinary `NEEDS_REPREPARE` outcomes and can be parked before heavy execution.

### Publication completion, webhooks, and post-publication verification

A successful normal operation becomes `COMMITTED` after:

1. the exact audited candidate wins the expected-revision publication compare-and-swap; and
2. a cheap independent read confirms canonical `main == C`.

Phase 2.6 MUST NOT run another full clone/Index rebuild/full validation synchronously after every successful exact publication merely to re-prove the same immutable candidate. That would recreate the duplicated ceremony this milestone exists to remove.

The GitHub App SHOULD subscribe to the repository `push` event for fast out-of-band/ref-movement detection. A dedicated webhook ingress verifies GitHub's webhook signature before routing the immutable repository identity and exact `before`/`after`/`ref` evidence to the repository coordinator. A push matching the expected hosted publication may confirm/accelerate observation; an unexpected `main` push places the lane into reconciliation/suspension.

Webhook delivery is observability/acceleration, **not a correctness dependency**. Webhooks may be delayed, retried, or missed. Admission and publication still re-read exact canonical Git state directly. Duplicate webhook delivery is handled idempotently by delivery ID/event identity.

Read-only GitHub Actions or another independent verifier may still perform explicit repository-health audits, recovery/reconciliation checks, and validation around control-plane migrations. The existing memory-repository `validate.yml` push-on-every-change behavior must not remain as a redundant full runner for every normal audited data-plane publication once the hosted path is rolled out. Any workflow-trigger change is itself a managed control-plane/bootstrap change and must pass the appropriate migration/downstream barrier rather than being silently patched by the data-plane service.

### Out-of-band canonical movement and reconciliation

The repository owner remains able to change their own Git repository, particularly when their GitHub plan does not offer private branch/ruleset enforcement.

Before starting heavy execution and again before publication, Runethread compares observed canonical state to the coordinator's accepted/expected revision. Unexpected movement that was not produced/adopted through the hosted audited path yields `RECONCILIATION_REQUIRED` or equivalent fail-closed lane state.

Recovery must independently inspect the exact out-of-band revision and either:

- adopt it only after applicable trust/validation/index/repository-health checks establish a new accepted canonical base; or
- restore/repair through an explicit authorized recovery procedure.

The hosted service must not silently label arbitrary owner movement as audited merely because the owner had permission to push it. A GitHub App webhook can accelerate detection but cannot replace the exact direct ref checks above.

### Control-plane barriers

Contract/schema/trust/repository-format/bootstrap/migration/managed-delivery changes remain exclusive barriers as defined by ADR-013.

The per-repository coordinator represents a barrier through a state equivalent to `MAINTENANCE`. New data-plane publication is refused/parked while the barrier is active. Pre-barrier semantic operations that have not committed cannot publish under post-barrier semantics without re-preparation.

Installing or materially changing the Phase 2.6 hosted delivery mechanism itself is a control-plane barrier and must be rolled out through the full development/release/downstream process rather than through the new normal data-plane path it is creating. Hosted deployment machinery must account for non-atomic Worker/Container rollout as defined by the hosted component/release boundary above.

### Resource, abuse, and privacy limits

Hosted execution introduces service-level resource and abuse boundaries that do not belong in the semantic memory schema.

The hosted adapter MUST enforce explicit limits for at least:

- authenticated request rate and concurrent queued work;
- inline request size, with a supported immutable referenced-payload path for larger legitimate operations rather than inheriting an unrelated provider transport cap as a semantic memory limit;
- repository/current-tree and candidate-package size supported by the hosted profile;
- Container wall time, memory, CPU, and ephemeral disk use;
- candidate artifact retention/size;
- Workflow retry count/backoff and total execution lifetime;
- log/event payload size and redaction of memory content/secrets.

Resource exhaustion or quota failure must fail an operation explicitly and preserve canonical Git state. Retryable provider failures must remain distinguishable from deterministic semantic/finalization failures.

Hosted private-memory processing means Cloudflare infrastructure temporarily processes the authorized repository and candidate data. Public rollout therefore requires an explicit data-handling/threat-model review covering private candidate storage, logs, credential isolation, retention/deletion, and provider access boundaries. Hosted convenience must not be described as end-to-end private from the infrastructure operator when the service necessarily executes against plaintext memory content.

### Project current-state views are projections, not atomic-memory dual writes

Phase 2.6's normal atomic-memory transaction does not become a generic writer for `projects/<slug>/current-state.md` or other orientation prose solely to keep those summaries synchronized with every memory commit.

Canonical atomic memories and authoritative project source repositories remain the durable sources of truth for their respective facts. Project overview/current-state files are orientation/materialized views. A future deterministic projection/refresh mechanism may update them asynchronously from authoritative evidence and must clearly represent freshness/verification state.

Temporary staleness of such an orientation view is degradation, not corruption of an otherwise valid atomic-memory transaction. Phase 2.6 implementation must not introduce a second semantic mutation engine in workflow/service code merely to rewrite project-summary Markdown.

### MCP relationship

Phase 2.6 establishes provider-neutral hosted delivery operations such as submit/apply-status/cancel semantics before MCP transport is added.

Phase 3 remains a thin MCP transport over the same MemoryService and hosted delivery lifecycle. MCP does not become the queue, mutation engine, audit engine, or canonical store. A local/offline Runethread path continues to use MemoryService directly without requiring Cloudflare.

The hosted Cloudflare implementation must therefore be replaceable without changing canonical memory format, and local Core behavior must remain usable without the hosted service.

## Consequences

- GitHub Free and paid users share one normal hosted Runethread mutation architecture.
- Paid GitHub branch/ruleset protection becomes defense-in-depth rather than a hidden product prerequisite.
- The hosted Runethread service requires a Cloudflare Workers Paid account for Containers; end users do not need their own Cloudflare account or paid Cloudflare plan.
- Provider-specific hosted code/release lifecycle stays outside Core; the target implementation repository is `runethread/hosted`.
- GitHub Actions runner startup latency leaves the normal interactive mutation critical path.
- The Phase 2.6 implementation now owns a small hosted operational state store per repository, while canonical semantic state and committed evidence remain Git-owned.
- Existing MemoryService mutation semantics can be reused rather than split/reimplemented solely to produce a hosted candidate.
- The repository runtime Durable Object can manage its attached finalizer Container without a second per-repository coordination object.
- Phase 2.6 v1 serializes the expensive finalization/audit/publication workflow per repository so known-stale queued operations can be rejected before Container work.
- A cold finalizer targets one source clone/fetch; warm cache reuse may reduce this to fetch-only without becoming a correctness dependency.
- Independent audit consumes exact immutable candidate evidence and may use a bounded read-only `H0` fetch when that is cheaper/simpler than transporting a full candidate snapshot. Transfer volume and exactness, not clone count alone, drive optimization.
- Publisher implementation can be clone-free if GitHub Git-object APIs preserve exact candidate identity; otherwise exact candidate push is the safe fallback.
- Long-lived GitHub App key material is isolated from the public API and execution containers behind a private internal publisher capability boundary.
- Per-write post-publication full validation is removed from the normal synchronous path; cheap exact-ref confirmation plus webhook/reconciliation/health checks provide distinct evidence without re-proving the same candidate.
- Breaking hosted deployments require explicit version/barrier handling because provider Worker/Container rollout is not assumed atomic.
- Project current-state prose is prevented from becoming an authoritative dual-write requirement in the memory transaction.
- The Cloudflare control plane can later absorb richer remote execution/orchestration capabilities without changing Core's deterministic memory semantics, while the future general Orchestrator remains a separate concern under ADR-005.

## Alternatives considered

### Keep GitHub Actions as the primary Phase 2.6 executor

Rejected as the final hosted profile. It preserves provider runner latency and workflow-specific state plumbing while the project already intends to offer remote MCP/hosted execution. GitHub Actions remains useful as an independent read-only audit/recovery/control-plane surface.

### Maintain separate Free and paid GitHub delivery implementations

Rejected. Branch protection availability should not fork Runethread's normal mutation semantics. One hosted publisher path plus optional GitHub protection is simpler and easier to verify.

### Put hosted Cloudflare/provider code in `runethread/core`

Rejected. Provider runtimes, deployment state, credentials, TypeScript/Worker dependencies, and cloud release churn are operational concerns separate from the durable deterministic memory compatibility surface. The hosted component consumes Core rather than making Core consume the hosted provider.

### Put the GitHub App private key in each user's memory-repository Actions secrets or public API Worker

Rejected. Long-lived canonical-publisher authority belongs in a private hosted capability boundary, not user-editable repository workflow configuration or the Internet-facing request handler.

### Rewrite MemoryService for Workers to avoid Containers

Rejected. Core already owns deterministic Git/filesystem mutation semantics. Reimplementing them in JavaScript/TypeScript to avoid the Workers Paid Container requirement would create a second mutation engine and increase correctness risk.

### Use a persistent cloud Git mirror as canonical memory

Rejected. The user-owned GitHub repository remains canonical. Hosted caches/artifacts are execution aids and must not become a second canonical memory database.

### Require zero GitHub reads after finalization

Rejected as a hard invariant. For large repositories, shipping a full current-tree candidate package can cost more than a bounded fresh read of exact base `H0`. The auditor must be fresh and exact, but the implementation should minimize total bytes/runtime rather than optimize an arbitrary clone counter.

### Reclone full Git history separately for finalizer, auditor, publisher, and post-publication validation

Rejected. The normal finalizer is the only component that needs historical idempotency evidence. The auditor needs an exact candidate/current base, the publisher needs exact Git objects, and normal completion only needs the canonical ref. Their data acquisition should match those distinct needs.

### Run multiple finalizer/auditor workflows concurrently within one repository in v1

Rejected. Concurrent semantic preparation remains allowed, but singleton Phase 2.6 publication means most same-base candidates would become stale after the first commit. Serializing the heavy workflow lets the coordinator park known-stale work before paying for Containers/index/audit. Future batching/dependency-aware concurrency requires a separate accepted design.

### Merge finalizer and auditor to save one container

Rejected. The fresh reduced-privilege auditor is a distinct trust boundary. Saving that container would remove meaningful independent evidence rather than remove ceremony.

### Make project current-state prose part of every atomic memory commit

Rejected for Phase 2.6. That creates a semantic dual-write requirement and expands the critical mutation surface. Project orientation views should be treated as projections/materialized views with separate freshness semantics.

## Verification

An implementation complies with this ADR only if tests/integration evidence demonstrate at least:

1. hosted mutation execution works for a private GitHub repository without requiring private-repository branch protection;
2. the same normal hosted mutation path works when stronger GitHub branch/ruleset protection is enabled;
3. provider-specific hosted code remains outside Core and the hosted runtime uses an exact verified Core/runtime identity rather than floating `main`;
4. the semantic client and public API Worker do not receive canonical GitHub write credentials or the App private key;
5. the App private key exists only in a private internal GitHub gateway/publisher capability boundary and token issuance is repository/permission scoped;
6. the repository-runtime Durable Object manages lane state and its attached finalizer Container without another per-repository coordination authority;
7. the per-repository coordinator permits at most one heavy mutation Workflow at a time in v1 while independent repositories can progress independently;
8. after a known hosted commit advances `main`, queued stale work becomes `NEEDS_REPREPARE` before Container startup;
9. Workflow retry/resume cannot duplicate a canonical mutation, and loss of a completion callback cannot cause the coordinator to release the lane without reconciliation;
10. a cold normal finalization path performs no more than one GitHub source clone/fetch for finalization, while any warm clone reuse is discarded/refreshed safely when stale or dirty;
11. the finalizer clone strategy preserves historical idempotency evidence required by `FindAppliedOperation` and rejects unsafe/untrusted Git execution features;
12. the hosted finalizer invokes the existing MemoryService/Core mutation implementation and does not perform a redundant second Index write;
13. remote GitHub `main` remains at `H0` after finalization and before audit/publication;
14. candidate evidence is cryptographically bound to `H0`, exact `C`, tree identity, request fingerprint, runtime/container-image/hosted-delivery/contract identity, and operation identity;
15. candidate packaging works correctly with the chosen clone mode and cannot omit required promisor objects;
16. a fresh auditor reconstructs and verifies exact `C` from immutable candidate evidence plus, when needed, a bounded read-only exact-`H0` source without obtaining publication authority;
17. the fresh auditor performs no repair and passes hard validation plus strict Index v2 freshness on exact `C`;
18. audit failure leaves GitHub canonical state unchanged and suspends the hosted repository lane;
19. publication requires the remote canonical ref to still equal `H0` and changes it only to exact audited `C` without force, merge, or semantic reconstruction;
20. an unexpected canonical mismatch during the one active hosted workflow enters reconciliation, while known queued stale work is parked as ordinary `NEEDS_REPREPARE`;
21. the preferred clone-free GitHub-object publication path is used only if an integration test proves GitHub receives the exact candidate commit/tree identity; otherwise the exact-candidate push fallback is used;
22. successful publication is followed only by cheap exact-ref confirmation in the normal synchronous path, not another full clone/validation cycle;
23. signed GitHub App push webhooks detect normal and out-of-band `main` movement idempotently but are not required for correctness when delivery is delayed/missed;
24. unexpected/out-of-band canonical movement enters a fail-closed reconciliation state before later hosted writes;
25. the existing push-on-every-memory GitHub Actions validation is removed from the normal hosted data-plane critical/background path through the proper managed workflow/control-plane rollout, while recovery/migration/health checks remain available;
26. `NO_OP`, `ALREADY_COMMITTED`, stale reprepare, crash retry, cancellation, finalization failure, audit failure, publication race, provider retry, and reconciliation outcomes are deterministic and tested;
27. a control-plane barrier prevents pre-barrier prepared operations from publishing under changed semantics;
28. breaking hosted Worker/Workflow/container deployments cannot mix incompatible execution semantics inside one operation despite non-atomic provider rollout;
29. authentication and repository authorization prevent a caller from targeting a repository solely by guessing/supplying its identifier;
30. request/repository/artifact/runtime/retry limits fail explicitly without changing canonical Git state;
31. private candidate/repository content is excluded from ordinary logs/client responses and deleted according to a tested retention/recovery policy;
32. project current-state orientation files are not silently dual-written by provider code as part of the atomic memory transaction;
33. local/offline MemoryService behavior remains usable without Cloudflare and Core does not import provider-specific hosted dependencies;
34. Phase 3 MCP can call the same hosted delivery boundary without changing canonical memory format or mutation semantics;
35. end-to-end latency/cost measurements distinguish cold/warm source acquisition, bytes transferred, finalization, candidate packaging, audit, publication, and provider startup components so further optimization is evidence-driven.
