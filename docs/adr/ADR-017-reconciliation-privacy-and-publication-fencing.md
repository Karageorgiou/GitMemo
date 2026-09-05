# ADR-017: Phase 2.6 reconciliation, repository privacy, and publication fencing

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-014 and ADR-016 for out-of-band history adoption, hosted repository privacy eligibility, publisher-capability lifetime, and managed validation-bootstrap rollout

## Context

A fresh pre-implementation adversarial review of the synchronized Phase 2.6 planning head found four remaining correctness/security gaps.

First, ADR-014 allows an unexpected out-of-band canonical revision to be adopted after trust/validation/index/repository-health checks. Core's committed idempotency evidence, however, is stored in reachable Git history: `FindAppliedOperation` searches `git log HEAD` for Runethread mutation metadata. If hosted reconciliation adopts a backward or sideways history rewrite that no longer contains a previously accepted committed mutation, a later exact retry can appear uncommitted and be executed again. Tree validity alone is therefore insufficient for re-adopting rewritten history.

Second, Phase 2.6 repeatedly describes the canonical memory repository as private, but repository visibility was not an explicit live hosted eligibility condition. GitHub repository visibility can change independently of repository identity and the bound canonical ref. A private-to-public transition exposes repository contents and must fail closed for future hosted writes. Because repository visibility and a Git ref are separate GitHub resources, Runethread cannot atomically compare-and-swap both visibility and the ref; owner/admin visibility changes remain an explicit external privacy authority boundary.

Third, ADR-016 correctly gives publication a short-lived repository-scoped write token only after durable `PUBLISHING`, but that token plus its publisher executor are themselves an externally effective write capability. The general Durable Object rule that a late result from an obsolete generation is ignored is not sufficient for this phase: an obsolete publisher could still mutate Git even if its result is ignored. Publication recovery therefore needs capability fencing, not only result fencing.

Fourth, the existing `.github/workflows/validate.yml` in generated memory repositories is not part of `ContractPaths()` or the trust-lock digest, but it is still a Runethread-managed bootstrap/support file: `runethread init` installs it, labels it managed, and `runethread upgrade` verifies ownership and reinstalls the desired managed workflow. Removing push-on-every-memory validation must therefore be a released managed-file migration, not an ad-hoc hosted-service deletion/edit that a later upgrade could undo.

These findings preserve the accepted Cloudflare Worker + repository Durable Object + deterministic finalizer + fresh auditor + minimal publisher architecture. They tighten recovery, privacy, and publication-lifetime semantics.

## Decision

### 1. Reconciliation must preserve accepted committed history

The repository Durable Object records a last accepted canonical revision `A` for the bound canonical ref.

An unexpected out-of-band revision `N` may be considered for ordinary adoption as a new accepted canonical base only when `A` remains reachable from `N` as an ancestor. This descendant-only rule preserves all previously accepted canonical commit/idempotency evidence.

Before a descendant is adopted, recovery must independently verify the exact `N` under the applicable trust/contract/repository/index rules and verify the newly introduced Git-history range does not corrupt Runethread mutation/idempotency metadata. At minimum, recovery must fail closed on duplicate/conflicting Runethread operation identities or malformed mutation metadata that would make Core committed-idempotency lookup ambiguous. Provider code must not reimplement the mutation-trailer grammar when an appropriate deterministic Core/repository helper can own that check.

A revision that is not a descendant of `A` is a destructive history rewrite relative to the hosted accepted history. Phase 2.6 v1 MUST NOT simply adopt such a revision merely because its current tree validates.

A backward/sideways non-descendant rewrite remains `RECONCILIATION_REQUIRED` until an explicit authorized recovery establishes an ancestry-preserving canonical state, for example by restoring the accepted line or creating an explicitly reviewed recovery commit/history that preserves the previously accepted commits/evidence. A future destructive-rebaseline feature would require a separate accepted design covering lost committed-idempotency evidence, operation identity reuse, audit history, and user-facing consequences.

A normal Git revert is not a destructive history rewrite: the reverted commit remains reachable and therefore preserves historical/idempotency evidence.

### 2. Private repository visibility is a live hosted-write eligibility condition

Phase 2.6 v1 normal hosted memory mutation supports repositories whose directly observed GitHub visibility is `private`.

Hosted onboarding/adoption records the private-repository eligibility condition alongside immutable repository identity, App installation, canonical-ref binding, supported contract identity, and last accepted revision.

The service directly rechecks repository metadata/visibility at least:

- at onboarding/adoption;
- before admitting new normal hosted mutation work after a visibility/authorization uncertainty;
- before starting finalization when the eligibility state may have changed; and
- immediately before durable publication authorization/token issuance.

A current visibility other than `private` fails closed for normal hosted writes and publication. The lane enters suspension/reconciliation/privacy-incident handling rather than continuing to publish new memories. Returning the repository to private does not erase any prior exposure; re-enabling hosted writes requires explicit revalidation/re-adoption rather than silent automatic resume.

GitHub's `public` webhook event and relevant repository metadata events may accelerate detection, but webhook payloads remain hints. Direct current repository metadata is authoritative for the observed state.

Runethread does **not** claim an impossible cross-resource atomic guarantee between GitHub repository visibility and Git ref mutation. An authorized repository owner/admin can change visibility concurrently after Runethread's last visibility check and before an already-authorized Git push completes. Such an administrative visibility change already controls exposure of the entire repository and is outside the Git-ref CAS transaction. The threat model and product documentation must state this owner-controlled privacy boundary explicitly rather than implying that the Contents-only publisher can lock repository visibility.

### 3. `PUBLISHING` is a capability-bearing in-doubt phase

The repository Durable Object remains the sole authority that may initiate/retry publication, but after it authorizes `PUBLISHING` it must also account for every externally issued publisher capability.

Before minting a write token or starting publisher I/O, the DO durably records an exact publisher-attempt identity bound to repository, operation, phase/execution generation, H0, C, canonical ref, evidence identity, publisher protocol/image, and a bounded attempt deadline.

Publisher startup/token issuance/execution is idempotently addressed by that publisher-attempt identity. Recovery MUST NOT create parallel authoritative publisher executors for the same attempt merely because a response was lost.

The publisher executor performs at most one authoritative Git push attempt for that publisher-attempt identity. It contains no autonomous retry loop that can outlive DO scheduling authority.

The DO MUST NOT release the repository lane, start a later canonical publication, abandon/reuse the publication generation, or classify a moved ref as safely terminal while an issued publisher executor/token from that generation may still perform a delayed write.

Publication capability becomes fenced/quiesced only when the system has sufficient evidence that the old capability can no longer act. Normal mechanisms include:

- the exact publisher attempt completed and the executor terminated under the expected protocol;
- the installation token was explicitly revoked after use and the executor is terminated; or
- after an ambiguous/lost publisher attempt, the executor is stopped/destroyed and any unconfirmed token capability has reached its recorded expiry before the lane is released.

If clean token revocation cannot be confirmed, expiry is the conservative safety fence. Current GitHub installation tokens have a bounded expiry, so this can impose a rare failure-path availability delay; that delay is preferable to allowing an obsolete publisher to reapply `C` after the canonical ref later returns to `H0`.

Only after prior publisher capability is fenced does the DO perform/accept the authoritative ref observation that resolves the attempt:

- ref == `C` -> `COMMITTED`;
- ref == `H0` -> the same logical publication may be retried by the DO with a new tracked publisher attempt/capability;
- any other value -> `RECONCILIATION_REQUIRED`.

A stale/obsolete publisher **result** still cannot advance a newer generation, but publication safety no longer relies on result rejection alone. The external write capability itself must be fenced first.

Opaque publisher-attempt/capability identity, deadline/expiry, and lifecycle status are operational metadata. The write token itself MUST NOT be stored in ordinary DO state, logs, status responses, or evidence objects.

### 4. Managed validation-workflow transition uses the released upgrader/downstream migration

The current memory-repository validation workflow is a managed Runethread support/bootstrap file even though it is outside the contract-lock digest.

Phase 2.6 therefore removes push-on-every-normal-memory full validation only through a deliberate released managed-file transition:

1. the Core release that introduces the new managed workflow behavior updates `runethread init`, upgrader ownership/desired-file logic, tests, and documentation together;
2. the exact prior managed workflow remains recognized as a supported migration source;
3. customized/unrecognized workflow state is not silently overwritten; it follows explicit operator/migration handling;
4. `runethread/memory-template` is migrated using the released upgrader and permanently validated;
5. the known private memory repository is migrated only after template/release verification;
6. hosted normal mutation is enabled only after the repository contains the supported managed validation-bootstrap state for that hosted release.

Because ADR-015 already requires a contract-v9 release/migration before normal hosted writes, Phase 2.6 should perform this managed workflow transition in that same migration/release sequence unless implementation preflight establishes a safer separately versioned barrier. The workflow-trigger change is not, by itself, a reason to pretend `.github/workflows/validate.yml` is a `ContractPaths()` file; it is a managed bootstrap/support-file migration with release/downstream consequences.

The replacement workflow may retain explicit PR, manual, recovery, migration, scheduled health, or other distinct validation triggers. The invariant is that every normal hosted memory publication must no longer automatically launch a redundant full validation run merely because the canonical ref moved.

Normal hosted mutation provider code MUST NOT hand-edit/delete the managed workflow as part of a memory operation.

### 5. Control-plane barriers include these new eligibility/fencing states

Hosted release barriers additionally cover:

- accepted-history/reconciliation policy;
- repository visibility eligibility state;
- publisher-attempt/capability-fencing protocol and deadline semantics;
- managed memory-repository validation-bootstrap version/support state.

An in-flight operation cannot cross incompatible versions of these policies.

## Consequences

- Ordinary out-of-band descendant commits can still be adopted after explicit health/history verification, preserving local/offline/user ownership.
- Destructive backward/sideways history rewrites cannot silently erase the very Git evidence Core uses to guarantee committed idempotency.
- Hosted memory writes stop when the repository is observed as non-private, while the design honestly acknowledges that repository owners/admins remain the authority capable of changing visibility outside the Git CAS.
- Publication response loss can block a repository lane until the old publisher capability is definitely dead/revoked/expired; this is intentional fail-closed behavior.
- DO generation checks remain necessary but are no longer incorrectly treated as sufficient fencing for an already-issued Git write capability.
- The default memory-repository Actions optimization becomes a reproducible released migration instead of configuration drift that `runethread upgrade` could later reverse.
- No Cloudflare Workflow, second mutation engine, second source clone, or redundant post-publication full validation is introduced.
- Because this ADR records material changes found by the current architecture-freeze attack, that attack does **not** satisfy the zero-edit gate. Implementation remains blocked until a later full adversarial review of the new exact planning head passes with zero required edits.

## Alternatives considered

### Adopt any valid out-of-band tree regardless of ancestry

Rejected. It can drop previously accepted mutation commits from reachable history and break ADR-003 committed-idempotency semantics.

### Copy all committed idempotency records into the Durable Object so rewritten Git history can be adopted freely

Rejected. It creates a second durable semantic/idempotency authority and makes hosted-state recovery part of canonical memory meaning.

### Reject every out-of-band canonical change permanently

Rejected. Runethread remains user-owned/local-first; valid descendant local/manual changes can be independently reconciled without losing accepted history.

### Give the GitHub App Administration permission so Runethread can lock repository visibility

Rejected. It materially broadens authority and still conflicts with the user-owned repository model. Visibility remains an owner/admin-controlled external privacy boundary that Runethread observes and fails closed around.

### Release the lane as soon as an ambiguous publisher result is ignored

Rejected. Ignoring a stale result cannot revoke a write token or stop a delayed publisher process from mutating Git later.

### Let the publisher executor retry internally until success

Rejected. It creates an external retry authority that can outlive the DO's current publication generation and defeats deterministic fencing/reconciliation.

### Delete the managed validation workflow from the hosted service

Rejected. The starter/upgrader owns that file and a later upgrade could recreate it. Managed repository support files transition through the release/upgrader/downstream migration process.

## Verification

Implementation is compliant only if tests/evidence demonstrate at least:

1. descendant out-of-band canonical movement preserves the last accepted revision in ancestry and can be adopted only after exact trust/repository/index/history checks;
2. a backward or sideways non-descendant rewrite is not adopted as an ordinary new base even when its tree validates;
3. a previously committed operation remains discoverable after every supported reconciliation/adoption path;
4. duplicate/conflicting/malformed Runethread mutation metadata introduced by an out-of-band descendant fails closed before adoption;
5. repository visibility is directly verified as private at hosted onboarding and publication-relevant revalidation points;
6. observed public/internal/non-private visibility blocks new normal hosted mutation/publication and requires explicit revalidation before resume;
7. the GitHub `public` webhook is only an acceleration hint and direct repository metadata owns the current observed visibility decision;
8. threat-model documentation does not claim atomic visibility+ref CAS or protection against an authorized owner/admin making the repository public concurrently with publication;
9. publisher-attempt identity is durably persisted before external token/executor I/O and binds exact repository/ref/H0/C/operation/generation/evidence/protocol identity;
10. response loss while creating/starting a publisher cannot create two parallel authoritative publisher executors for one publisher attempt;
11. the publisher performs at most one Git push per publisher-attempt identity and has no autonomous retry loop;
12. a lane cannot leave the publication in-doubt/fencing state while an old executor/token may still act;
13. clean completion proves executor termination and token revocation/disposal policy; ambiguous completion destroys/stops the executor and waits for unconfirmed token expiry before lane release;
14. an obsolete publisher cannot reapply `C` after a later out-of-band ref rewind to `H0`;
15. a new publication retry is issued only after every earlier publisher capability for the logical publication is fenced;
16. publisher tokens never appear in ordinary DO state/logs/status/evidence;
17. the released managed workflow transition updates starter/upgrader/tests/docs coherently and recognizes the exact previous managed workflow;
18. customized/unrecognized validation workflows are not silently overwritten;
19. template and real private-repository migration prove the new supported validation-bootstrap state before normal hosted writes are enabled;
20. normal hosted memory publication no longer triggers Runethread's redundant full push validation, while distinct explicitly retained validation/recovery/migration/health paths remain available;
21. provider code never modifies the managed validation workflow as part of an ordinary memory mutation;
22. hosted version/deployment barriers prevent mixing incompatible reconciliation, visibility-eligibility, publisher-fencing, or managed-bootstrap semantics.