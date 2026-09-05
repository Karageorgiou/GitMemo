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

The same review found that Cloudflare now provides primitives that map closely to the already-accepted provider-independent design: Workers for the API edge, Durable Objects for per-repository stateful coordination, Workflows for durable multi-step execution, and Containers for the real Linux/Go/Git Runethread runtime. Containers require the Workers Paid plan; this is an infrastructure requirement of the hosted Runethread service account, not an end-user Cloudflare-plan requirement.

The review also confirmed that the current `MemoryService.ApplyMutation` already creates an isolated transaction, regenerates Index v2, hard-validates, creates the exact mutation commit, and only fast-forwards the caller's *local* canonical branch. It does not push GitHub. A hosted finalizer can therefore reuse today's deterministic MemoryService implementation in an ephemeral clone without making remote `main` canonical prematurely.

Finally, project `current-state.md` files are orientation/current-state views, not the authoritative atomic-memory transaction. They must not force Phase 2.6 to become a generic prose dual-write transaction. Their refresh can be designed as a separate projection/materialized-view concern.

## Decision

Phase 2.6 uses a **Cloudflare-hosted remote memory-delivery control plane** as its primary hosted execution profile. GitHub remains the user-owned canonical Git store. GitHub Actions is retained for independent repository-health, recovery, migration, and control-plane validation where useful, but is not the normal interactive memory-write executor or queue.

This ADR supersedes only the conflicting GitHub-Actions-first implementation details of ADR-012 and ADR-013. Their semantic/canonical invariants remain accepted unless this ADR explicitly changes them.

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

### Component ownership

The Phase 2.6 hosted profile has four distinct responsibilities.

1. **Worker/API edge** — authenticates/authorizes requests, resolves the target authorized repository, validates request envelope limits, and routes to the repository coordinator. It does not implement memory semantics.
2. **Per-repository Durable Object** — owns hosted lane state, admission/serialization, operation identity/status mapping, suspension/maintenance state, and the last accepted canonical revision for that repository. It is the provider-specific durable coordinator for pending delivery state, not a second copy of canonical memory.
3. **Cloudflare Workflow** — owns resumable execution checkpoints and retries for one admitted operation. Workflow steps are individually retryable and MUST be idempotent. Workflow execution state does not replace canonical Git evidence for whether a memory mutation committed.
4. **Container runtime** — runs the real Runethread Go binary/Core and Git in Linux. Core remains provider-independent and MUST NOT import Cloudflare or GitHub SDK dependencies merely because the hosted adapter uses those providers.

The coordinator and Workflow have deliberately different roles: the Durable Object answers *which operation may use this repository lane and what hosted lane state applies*; the Workflow answers *which retryable execution step this operation has durably reached*. Their state models must not become two independently mutable sources of truth for the same transition.

### Canonical and hosted state ownership

Canonical semantic memory, project data, generated indexes, and committed mutation/idempotency evidence remain in the user's Git repository as governed by ADR-001 through ADR-004.

The hosted coordinator may persist operational delivery state such as:

- repository identity and authorized GitHub App installation binding;
- lane state such as `OPEN`, `SUSPENDED`, `MAINTENANCE`, or `RECONCILIATION_REQUIRED`;
- operation identity/idempotency identity;
- expected/base revision;
- operation state;
- candidate identity when one exists;
- exact audit evidence identity;
- final committed revision or failure outcome.

This is authoritative hosted control state while the service is operating, but it is not canonical memory. A committed memory outcome must always be recoverable/confirmable from exact Git history and mutation metadata. Loss of hosted pending-state storage must not cause committed canonical history to become ambiguous; resubmitting the same sealed operation/idempotency identity remains the recovery fallback.

This explicitly replaces ADR-012/ADR-013's GitHub-workflow-run-as-transitional-state implementation profile. It does not change the canonical memory schema or repository format solely to store delivery state.

### Admission and operation states

The API accepts one complete sealed MemoryService-compatible operation or an explicitly supported immutable payload reference. A caller must not construct a watched request tree through multiple Git commits.

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

### Per-repository serialization

One logical publication lane exists per canonical memory repository. The coordinator identity is derived from the immutable provider repository identity, not only a mutable owner/name string.

Independent repositories execute independently. Within one repository, semantic preparation may occur concurrently, but hosted mutation admission/publication remains serialized and every operation independently proves its expected Git revision.

Strict FIFO remains an availability/fairness choice rather than a semantic correctness assumption. Stale operations transition to `NEEDS_REPREPARE` and leave the runnable lane. No hosted scheduler may semantically rewrite or automatically reinterpret them merely to keep the lane moving.

### Finalization with the existing MemoryService

The normal finalizer MUST use the existing deterministic MemoryService/Core implementation rather than a Cloudflare- or workflow-specific mutation reimplementation.

For a cold executor, the target is at most **one GitHub clone** in the normal operation path. The clone SHOULD preserve complete reachable commit history required by `FindAppliedOperation` while avoiding eager historical blob transfer where Git partial-clone support is safe, for example a blobless partial clone. A shallow clone MUST NOT be used if it could hide historical idempotency evidence.

A warm container MAY reuse its existing clone as an untrusted cache by fetching and restoring a clean exact canonical state before the operation. Cache reuse must never be a correctness dependency; container disk is ephemeral and may disappear between requests.

Finalization first checks the current canonical revision. If the request is already committed under the same idempotency identity, the operation returns `ALREADY_COMMITTED`. If the request is not already committed and current canonical state differs from the prepared expected revision, it becomes `NEEDS_REPREPARE` before expensive candidate construction.

For a runnable operation, the finalizer calls `ApplyMutation` in the isolated hosted clone. The local fast-forward performed by `ApplyMutation` yields exact candidate commit `C` inside that disposable clone; remote GitHub `main` remains at expected revision `H0`.

The finalizer MUST NOT redundantly run another Index v2 write or another equivalent repair pass after successful `ApplyMutation`. `ApplyMutation` already performs Index v2 generation and hard validation as part of candidate construction.

### Candidate artifact

After finalization, the exact candidate is exported to a private immutable/content-addressed candidate artifact suitable for a fresh auditor and, if required, exact later publication. The initial hosted profile may use private object storage such as R2, but the object format is an implementation detail subject to an early prototype.

The artifact/manifest must bind at least:

- operation identity and idempotency identity;
- expected/base revision `H0`;
- candidate commit `C` and candidate tree identity;
- request fingerprint;
- mutation metadata;
- exact Runethread runtime/build identity and pinned contract identity used for finalization;
- a cryptographic digest of the candidate package.

The artifact may contain private memory data. It therefore MUST be private, inaccessible to semantic clients by default, excluded from ordinary logs, and retained only as long as required for audit, publication, retry/recovery evidence, or an integrity incident.

The artifact format must be proven self-contained enough for the auditor to reconstruct/verify the candidate without another GitHub clone. A partial-clone promisor state must not yield an incomplete candidate package.

### Independent audit without another GitHub clone

A fresh auditor executes in a separate fresh container/environment with no canonical publication credential. It consumes the exact immutable candidate artifact rather than recloning/reconstructing the repository from GitHub.

The auditor uses the same deterministic Runethread implementation/runtime identity as the finalizer and verifies the exact candidate, including at minimum:

- artifact/package digest and manifest binding;
- candidate commit/tree identity and expected parent/base relationship;
- operation/idempotency/request-fingerprint mutation metadata;
- applicable trust/control-plane state;
- hard repository validation;
- strict `index --check` freshness/integrity;
- expected mutation diff/scope constraints;
- absence of unauthorized control-plane or unrelated user-data changes.

The auditor is observational: it MUST NOT run repair/index-write steps to make a candidate green and MUST NOT possess the publisher's canonical-write credential.

This independent audit remains intentionally limited: a deterministic bug shared by the same Runethread implementation can affect both finalizer and auditor. Independence protects against workspace contamination, candidate mix-ups, transport corruption, wrong-SHA publication, privilege mistakes, and environment/race errors; it is not a second semantic implementation.

### Publication and GitHub App authority

The Runethread GitHub App is installed only on repositories explicitly authorized by the user. Long-lived App private-key material lives only in the hosted Runethread secret boundary and is never copied into user memory repositories or exposed to semantic clients.

Per-operation installation credentials must be repository-scoped and permission-narrowed to the minimum publication authority GitHub supports. Installation tokens are short-lived and SHOULD be revoked when no longer needed where practical.

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

If the Git object API cannot preserve the exact candidate commit identity reliably, the fallback is a minimal privileged publication environment that imports the audited candidate artifact and pushes exact `C`; it still must not perform a new semantic mutation or reconstruct a different commit.

### Publication completion and post-publication verification

A successful normal operation becomes `COMMITTED` after:

1. the exact audited candidate wins the expected-revision publication compare-and-swap; and
2. a cheap independent read confirms canonical `main == C`.

Phase 2.6 MUST NOT run another full clone/Index rebuild/full validation synchronously after every successful exact publication merely to re-prove the same immutable candidate. That would recreate the duplicated ceremony this milestone exists to remove.

Read-only GitHub Actions or another independent verifier may still perform periodic repository-health audits, explicit recovery/reconciliation checks, and validation around control-plane migrations. A red independent health audit or unexpected canonical movement suspends later hosted writes until reconciliation.

### Out-of-band canonical movement

The repository owner remains able to change their own Git repository, particularly when their GitHub plan does not offer private branch/ruleset enforcement.

Before admitting/publishing later hosted mutations, Runethread compares observed canonical state to its last accepted/audited canonical revision. Unexpected movement that was not produced/adopted through the hosted audited path yields `RECONCILIATION_REQUIRED` or equivalent fail-closed lane state.

Recovery must independently inspect the exact out-of-band revision and either:

- adopt it only after applicable trust/validation/index/repository-health checks establish a new accepted canonical base; or
- restore/repair through an explicit authorized recovery procedure.

The hosted service must not silently label arbitrary owner movement as audited merely because the owner had permission to push it.

### Control-plane barriers

Contract/schema/trust/repository-format/bootstrap/migration/managed-delivery changes remain exclusive barriers as defined by ADR-013.

The per-repository coordinator represents a barrier through a state equivalent to `MAINTENANCE`. New data-plane publication is refused/parked while the barrier is active. Pre-barrier semantic operations that have not committed cannot publish under post-barrier semantics without re-preparation.

Installing or materially changing the Phase 2.6 hosted delivery mechanism itself is a control-plane barrier and must be rolled out through the full development/release/downstream process rather than through the new normal data-plane path it is creating.

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
- GitHub Actions runner startup latency leaves the normal interactive mutation critical path.
- The Phase 2.6 implementation now owns a small hosted operational state store per repository, while canonical semantic state and committed evidence remain Git-owned.
- Existing MemoryService mutation semantics can be reused rather than split/reimplemented solely to produce a hosted candidate.
- A cold normal mutation targets one GitHub clone; warm cache reuse may reduce this to fetch-only without becoming a correctness dependency.
- Independent audit consumes the immutable candidate artifact instead of performing a second GitHub clone.
- Publisher implementation can be clone-free if GitHub Git-object APIs preserve exact candidate identity; otherwise exact artifact push is the safe fallback.
- Per-write post-publication full validation is removed from the normal synchronous path; cheap exact-ref confirmation plus independent health/recovery audits provide distinct evidence without re-proving the same candidate.
- Project current-state prose is prevented from becoming an authoritative dual-write requirement in the memory transaction.
- The Cloudflare control plane can later absorb richer remote execution/orchestration capabilities without changing Core's deterministic memory semantics, while the future general Orchestrator remains a separate concern under ADR-005.

## Alternatives considered

### Keep GitHub Actions as the primary Phase 2.6 executor

Rejected as the final hosted profile. It preserves provider runner latency and workflow-specific state plumbing while the project already intends to offer remote MCP/hosted execution. GitHub Actions remains useful as an independent read-only audit/recovery/control-plane surface.

### Maintain separate Free and paid GitHub delivery implementations

Rejected. Branch protection availability should not fork Runethread's normal mutation semantics. One hosted publisher path plus optional GitHub protection is simpler and easier to verify.

### Put the GitHub App private key in each user's memory-repository Actions secrets

Rejected. Long-lived canonical-publisher authority belongs in the hosted service secret boundary, not in user-editable repository workflow configuration.

### Rewrite MemoryService for Workers to avoid Containers

Rejected. Core already owns deterministic Git/filesystem mutation semantics. Reimplementing them in JavaScript/TypeScript to avoid the Workers Paid Container requirement would create a second mutation engine and increase correctness risk.

### Use a persistent cloud Git mirror as canonical memory

Rejected. The user-owned GitHub repository remains canonical. Hosted caches/artifacts are execution aids and must not become a second canonical memory database.

### Reclone GitHub separately for finalizer, auditor, publisher, and post-publication validation

Rejected. The normal design targets one cold clone; immutable candidate transport lets later trust boundaries verify/publish the same candidate without repeated repository downloads.

### Merge finalizer and auditor to save one container

Rejected. The fresh reduced-privilege auditor is a distinct trust boundary. Saving that container would remove meaningful independent evidence rather than remove ceremony.

### Make project current-state prose part of every atomic memory commit

Rejected for Phase 2.6. That creates a semantic dual-write requirement and expands the critical mutation surface. Project orientation views should be treated as projections/materialized views with separate freshness semantics.

## Verification

An implementation complies with this ADR only if tests/integration evidence demonstrate at least:

1. hosted mutation execution works for a private GitHub repository without requiring private-repository branch protection;
2. the same normal hosted mutation path works when stronger GitHub branch/ruleset protection is enabled;
3. the semantic client does not receive canonical GitHub write credentials;
4. the per-repository coordinator serializes hosted publication while independent repositories can progress independently;
5. Workflow retry/resume cannot duplicate a canonical mutation and preserves stable operation/idempotency identity;
6. a cold normal finalization path performs no more than one GitHub clone, while any warm clone reuse is discarded/refreshed safely when stale or dirty;
7. the clone strategy preserves historical idempotency evidence required by `FindAppliedOperation`;
8. the hosted finalizer invokes the existing MemoryService/Core mutation implementation and does not perform a redundant second Index write;
9. remote GitHub `main` remains at `H0` after finalization and before audit/publication;
10. the candidate package is cryptographically bound to `H0`, exact `C`, tree identity, request fingerprint, runtime/contract identity, and operation identity;
11. the candidate package is self-contained enough that a fresh auditor verifies exact `C` without another GitHub clone;
12. the fresh auditor has no canonical publication credential, performs no repair, and passes hard validation plus strict Index v2 freshness on exact `C`;
13. audit failure leaves GitHub canonical state unchanged and suspends the hosted repository lane;
14. publication requires the remote canonical ref to still equal `H0` and changes it only to exact audited `C` without force, merge, or semantic reconstruction;
15. the preferred clone-free GitHub-object publication path is used only if an integration test proves GitHub receives the exact candidate commit/tree identity; otherwise the exact-artifact push fallback is used;
16. successful publication is followed only by cheap exact-ref confirmation in the normal synchronous path, not another full clone/validation cycle;
17. unexpected/out-of-band canonical movement enters a fail-closed reconciliation state before later hosted writes;
18. `NO_OP`, `ALREADY_COMMITTED`, stale reprepare, crash retry, cancellation, finalization failure, audit failure, publication race, and reconciliation outcomes are deterministic and tested;
19. a control-plane barrier prevents pre-barrier prepared operations from publishing under changed semantics;
20. private candidate artifacts are not exposed through ordinary logs/client responses and are deleted according to a tested retention/recovery policy;
21. project current-state orientation files are not silently dual-written by workflow/provider code as part of the atomic memory transaction;
22. local/offline MemoryService behavior remains usable without Cloudflare and Core does not import provider-specific hosted dependencies;
23. Phase 3 MCP can call the same hosted delivery boundary without changing canonical memory format or mutation semantics;
24. end-to-end latency/cost measurements distinguish cold-clone, warm-fetch, finalization, audit, publication, and provider startup components so further optimization is evidence-driven.
