# ADR-018: Indeterminate publication preserves possible committed history

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-014, ADR-016, ADR-017

## Context

ADR-016 requires exact expected-old publication from canonical revision `H0` to the independently audited candidate `C`. ADR-017 then fences every issued publisher capability before resolving an ambiguous publication and ordinarily permits out-of-band canonical adoption only when the last accepted revision remains in the new head's ancestry.

A later adversarial review found a gap between those two rules.

Consider this execution:

1. the last accepted canonical revision is `A = H0`;
2. Runethread authorizes exact publication `H0 -> C`;
3. the remote accepts `C`, but the publication response is lost or otherwise becomes indeterminate;
4. before Runethread resolves the attempt, an authorized repository owner or another out-of-band writer force-moves the canonical ref to `N`, where `H0` is an ancestor of `N` but `C` is not;
5. the old publisher capability is correctly fenced;
6. Runethread observes `N`.

If reconciliation now considers only the previous accepted anchor `A = H0`, ADR-017's ordinary descendant-adoption rule would allow `N` to be adopted after validation because `H0` remains reachable. That can erase a publication which actually succeeded. Core's committed idempotency lookup is intentionally based on mutation metadata reachable from canonical `HEAD`; once `C` is excluded from canonical ancestry, a later retry can no longer discover that committed operation and may re-execute it.

The same problem exists even when the publisher returned a definite success before an out-of-band rewrite occurred. A later current-ref read describes the repository's present state; it does not negate the already-proven historical fact that `C` was published.

Exact expected-old ref update remains necessary, but it is a point-in-time compare-and-swap on the ref value. It is not by itself a durable record of what happened after a successful update when the branch is later rewritten. Publication outcome and reconciliation therefore need one additional durable history-preservation invariant.

## Decision

A Phase 2.6 publication candidate which is **proven published** or **possibly published** remains a protected canonical-history anchor until its publication outcome and ancestry are safely resolved. Reconciliation MUST NOT discard that anchor merely because the previously accepted base revision is still an ancestor of the current ref.

This decision applies regardless of the concrete exact-CAS transport. The v1 Git publisher fallback and any later GitHub API publication path accepted under ADR-016 must obey the same outcome/reconciliation rules.

### Publication outcome classes

The repository Durable Object records publication outcome conservatively as one of three semantic classes:

- **proven not published** — the trusted publication transport provides a definitive result proving that the target ref update did not occur;
- **proven published** — the trusted publication transport provides a definitive success for exact `H0 -> C`, or later canonical ancestry independently proves that exact `C` is reachable from the bound canonical ref;
- **indeterminate** — timeout, response loss, executor loss, connection failure, or any other condition where Runethread cannot prove whether the exact ref update occurred.

Only a result which proves that the remote ref update did not occur may enter **proven not published**. A timeout, lost response, process crash, or merely observing `ref != C` is not such proof.

The exact candidate identity `C`, base `H0`, operation/generation/publisher-attempt identity, and still-required candidate/publication evidence remain durably referenced while publication is proven published but unreconciled, or indeterminate. Reference-aware retention from ADR-016/ADR-017 therefore pins this evidence until the history question is resolved.

### Positive publication is a durable fact

Once exact `H0 -> C` is **proven published**, the operation's publication fact is not revoked by a later force-push, branch reset, or other out-of-band ref movement.

`C` becomes an accepted history anchor for that committed operation even if the present canonical ref has already moved elsewhere. If the current ref excludes `C`, the repository lane is in reconciliation because accepted committed/idempotency history has been removed; Runethread MUST NOT reinterpret the operation as uncommitted and MUST NOT re-run its semantic mutation as a new candidate.

A current descendant of `C` may be independently reconciled under ADR-017's trust/repository/index/history-integrity rules. The operation remains committed at exact `C` while the later descendant is treated as separate out-of-band canonical evolution.

### Indeterminate publication resolution

After ADR-017 capability fencing has quiesced the prior publisher attempt, the DO resolves an **indeterminate** candidate `C` against the directly observed bound canonical ref.

1. **`ref == C`** — `C` is committed. Promote `C` to the accepted history anchor and mark the operation committed.
2. **`C` is an ancestor of current `ref`** — `C` is committed. Promote `C` to the accepted history anchor, mark the operation committed, then separately validate/reconcile the later descendant before reopening the lane.
3. **`ref == H0`** — Runethread may create a new tracked publication attempt for the **same exact `C`** after all normal authorization/privacy/barrier checks. Re-publishing the same candidate is safe whether the earlier attempt never wrote or wrote `C` and was later reset to `H0`; no new semantic finalization or `C2` is created merely to resolve the ambiguity.
4. **current `ref` is neither `H0` nor a descendant containing `C`** — the lane remains `RECONCILIATION_REQUIRED`. Even if the current ref is a descendant of the older accepted anchor `A = H0`, ordinary ADR-017 descendant adoption is forbidden because Runethread cannot distinguish “publication never happened, then `N` advanced” from “`C` committed, then was rewritten away to `N`.”

A later definitive transport result proving no write occurred may clear the possibly-published anchor and return the repository to ordinary reconciliation from the previous accepted anchor. Without such proof, Phase 2.6 fails closed rather than guessing.

### Recovery after a proven publication is removed

If exact `C` is **proven published** and the current ref no longer contains `C`, recovery must preserve or restore that committed/idempotency evidence before the normal hosted lane reopens.

A simple reset back to `H0` may be recoverable by an exact authorized `H0 -> C` restoration of the already-committed candidate, subject to current repository binding, private-visibility, authorization, and deployment barriers. A sideways descendant which excludes `C` may require an ancestry-preserving merge/recovery or explicit operator action.

Phase 2.6 v1 does not introduce a destructive-rebaseline feature which declares previously committed `C` forgotten. Such a feature would change committed-idempotency semantics and requires a separate accepted design.

### Reconciliation precedence

The protected publication anchor takes precedence over the ordinary “last accepted revision remains an ancestor” adoption test.

Before adopting any unexpected current revision, reconciliation must first ask whether an unresolved proven/possible publication candidate exists. If so, it must resolve or preserve that candidate under this ADR before ADR-017's ordinary descendant-adoption rule can apply.

Webhook delivery, current-ref reads, recovery alarms, retries, deployment restarts, or status requests MUST NOT clear this protected anchor as a side effect.

## Consequences

- A publication which actually succeeded cannot be silently forgotten because an owner rewrote the branch before Runethread observed the final ref.
- Core's reachable-history idempotency model remains valid after hosted reconciliation.
- Rare ambiguous publication plus out-of-band movement may suspend a repository longer, because safety takes precedence over guessing whether `C` committed.
- The existing ADR-017 exact-same-candidate retry at `ref == H0` remains available and becomes the preferred ambiguity-resolution path when possible.
- Publication transports may still be simplified or replaced under ADR-016, but they must expose results conservatively enough to distinguish definite no-write from indeterminate completion.
- No second semantic mutation engine, destructive automatic rebase, or destructive history rebaseline is added.

## Alternatives considered

### Treat current `ref != C` as proof that publication failed

Rejected. `C` may have been accepted and then rewritten away before the resolving read.

### Apply ADR-017 descendant adoption using only the older accepted base `H0`

Rejected. A sibling descendant can preserve `H0` while excluding a previously committed `C`, erasing canonical idempotency evidence.

### Re-finalize the operation into a new `C2`

Rejected. The unresolved question is publication history, not mutation semantics. Re-finalization can duplicate a mutation which already committed and breaks exact-candidate audit evidence.

### Depend on provider reflogs, audit logs, or webhook delivery to reconstruct the missing publication

Rejected as a correctness dependency. Availability, retention, plan support, and delivery guarantees are not uniform enough for the one-architecture private-repository baseline. Such evidence may help operators but does not replace the fail-closed canonical rule.

### Automatically discard previously proven publication after an owner force-push

Rejected. That converts an out-of-band rewrite into implicit destructive rebaseline and invalidates committed-idempotency guarantees.

## Verification

Implementation satisfies this ADR only if tests demonstrate at minimum:

1. exact `H0 -> C` succeeds, its response is lost, and the ref is then force-moved to sibling descendant `N`: `N` is **not** ordinarily adopted and `C` remains protected;
2. indeterminate publication followed by current ref `C` resolves `COMMITTED`;
3. indeterminate publication followed by descendant `N` containing `C` resolves the operation committed at `C` before independent descendant reconciliation;
4. indeterminate publication followed by exact `H0` may retry only the same exact `C`, never a new semantic candidate `C2`;
5. a definitive expected-old/ref rejection proving no write occurred may clear the possible-publication anchor and use ordinary reconciliation;
6. a definitive publication success followed by force-reset to `H0` or sideways `N` never reclassifies the operation as uncommitted;
7. recovery of a proven published `C` which was removed restores/preserves `C` in canonical ancestry before the normal hosted lane reopens;
8. webhooks, alarms, retries, stale generations, and process restarts cannot clear a proven/possible publication anchor prematurely;
9. candidate/publication evidence needed for this decision remains retained throughout ambiguous/reconciliation state and corrupt/missing evidence fails closed;
10. once the lane reopens, every previously proven committed Runethread operation remains discoverable through canonical reachable history under Core's existing committed-idempotency lookup;
11. both the minimal Git publisher fallback and any accepted API-based exact-CAS publication path obey the same proven-not-published/proven-published/indeterminate classification; and
12. no destructive rebaseline is silently introduced as an implementation shortcut.
