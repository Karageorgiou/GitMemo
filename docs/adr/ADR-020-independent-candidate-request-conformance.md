# ADR-020: Independent candidate-to-request conformance audit

Status: **Accepted**
Date: 2026-09-05
Tracking issue: #20
Amends: ADR-012, ADR-014, and ADR-016 where they define the independent prepublication audit

## Context

The next Phase 2.6 architecture-freeze attack review started from exact synchronized planning head `4dbdef5c08142856ba1795544795cea254193398`.

The accepted architecture already separates candidate construction from publication. The finalizer runs the pinned deterministic Runethread Core against exact base `H0`, persists exact candidate `C`, and cannot create authoritative audit evidence or publish canonical Git. A fresh reduced-privilege auditor then verifies exact `C`, including its parent/base binding, request fingerprint and mutation metadata, hard repository validity, strict Index v2 freshness, expected path scope, and absence of unrelated/control-plane changes.

That boundary is necessary but not sufficient to prove that the **semantic bytes in `C` are the result of the sealed mutation request**.

A faulty or compromised finalizer can know the legitimate sealed request and its normalized Core request fingerprint. It can therefore construct a different schema-valid/index-fresh candidate, copy the legitimate request fingerprint and operation trailers into the commit, keep the diff inside paths that are superficially plausible for that operation, and present the resulting candidate to the auditor. Parent, binding, digest, index, repository validation, and coarse scope checks can all pass while the candidate's Markdown/JSON meaning differs from the user's sealed request.

For example, an `update` request may propose content `A` for target memory `T`, while a malicious finalizer writes different valid content `B` to the same target, regenerates a perfectly fresh index, and places the request-for-`A` fingerprint in the mutation commit. Merely checking that `T` was the expected target does not prove that `B` came from request `A`.

The fresh auditor is intended to be an independent correctness boundary over finalizer output. It must therefore independently establish **request-to-candidate conformance**, not merely candidate self-consistency.

This does not require a second semantic implementation. The independent check must reuse the same pinned Core mutation semantics against the exact sealed request and exact `H0`.

## Decision

### 1. Publication eligibility requires independent request-to-candidate conformance

A candidate-producing Phase 2.6 operation is not `AUDITED` merely because exact `C` is repository-valid, index-fresh, correctly parented to `H0`, and within a permitted path set.

Before `C` is eligible for publication, the fresh auditor MUST independently prove that the candidate's semantic mutation is exactly the mutation that the pinned Core derives from:

- the exact sealed request accepted for the hosted operation;
- the exact normalized Core request fingerprint for that request;
- the exact canonical base `H0`;
- the pinned operational contract/runtime generation.

Finalizer-supplied descriptions of the expected diff, changed memory IDs, result class, or mutation metadata are evidence to compare, not authority for what the request should produce.

### 2. The conformance proof is Core-owned, not provider-hand-coded

The hosted provider MUST NOT implement a parallel hand-written interpretation of `create`, `update`, `correct`, `supersede`, `resolve`, `withdraw`, relationship/lifecycle rules, or request normalization merely to audit the finalizer.

The auditor uses a Core-owned verification path in its fresh disposable environment. The implementation may take either of these equivalent forms:

1. independently replay the mutation from exact `H0` and the sealed request through the pinned Core in a disposable local repository/worktree, then compare the deterministic semantic result to `C`; or
2. expose a dedicated Core candidate-conformance verifier that reuses the same mutation primitives to derive the exact expected canonical-memory changes from `H0` + request and compares them to `C` without reimplementing semantics in hosted code.

The verifier may optimize away construction of a second complete derived Index tree when it can prove the same invariant more cheaply. It MUST still use Core-owned mutation semantics for the semantic delta, and the actual candidate `C` still passes the existing independent strict Index v2 freshness/integrity check.

This is a second **execution** of the same deterministic semantics for verification, not a second semantic engine.

### 3. The complete candidate state must be explainable by the request

For a candidate-producing operation, the auditor proves at minimum:

- `C` has exactly one expected parent and that parent is exact `H0`;
- the candidate's canonical memory Markdown/JSON changes are byte-for-byte the semantic changes Core derives from exact `H0` + sealed request under the pinned contract/runtime;
- every relationship/lifecycle change to additional memory documents is exactly one Core derives from that request;
- candidate Index v2 is strictly fresh for the candidate canonical metadata;
- every repository path not legitimately changed by the Core mutation plus deterministic index regeneration is unchanged from `H0`;
- the mutation commit message/trailers, Core request fingerprint, operation identity, primary/target/changed-memory identities, and other Core-owned mutation metadata agree with the independently derived result;
- no finalizer-provided manifest or changed-path list can weaken those comparisons.

If independent Core verification says the request is `NO_OP`, `ALREADY_COMMITTED`, stale, invalid, or otherwise does not produce a candidate from `H0`, an asserted candidate `C` fails audit rather than being published.

Likewise, if the independently derived semantic result differs from `C` by even one authoritative memory byte, the audit fails even when both trees independently pass repository validation.

### 4. Exact replayed commit SHA equality is not required unless Core makes commit construction deterministic

The publication object remains exact audited candidate `C`; the auditor never replaces it with a replay-created `C2`.

Current Core creates the mutation commit in Git after deterministic semantic/index construction, and ordinary Git author/committer timestamps can make two otherwise identical locally created commit objects have different SHAs. Therefore an implementation which uses replay MUST NOT declare conformance merely by requiring the replay's commit SHA to equal `C` unless the applicable Core release deliberately makes all commit-object inputs deterministic.

Instead, the auditor binds and compares the fields that define the authorized mutation independently of incidental replay-time commit identity:

- exact parent `H0`;
- exact expected semantic tree/delta;
- exact Core-owned mutation message/trailers and request fingerprint;
- the candidate commit/object identity `C` actually being audited and later published;
- any commit metadata fields which the applicable Core release declares invariant.

Any nondeterministic commit-object fields allowed by the implementation must be explicitly bounded/tested and MUST NOT provide a channel for changing the candidate tree, parent, operation metadata, or request binding.

### 5. Auditor local writes are disposable verification work, not repair

Older ADR wording says the auditor performs no repair or index write. That remains true for the **candidate being audited** and for canonical Git: the auditor never mutates `C`, never repairs its index, and never writes the bound remote canonical ref.

A Core-owned conformance replay MAY write to a separate disposable local verification worktree in order to derive the expected result. Those local bytes are nonauthoritative scratch state and are destroyed after the audit. The auditor still has no publication credential, no finalizer evidence authority, and no ability to replace candidate evidence.

### 6. Audit evidence binds the independent conformance result

The immutable audit receipt/evidence accepted by the repository Durable Object additionally binds at least:

- repository/canonical-ref identity;
- hosted operation/attempt + audit generation;
- exact sealed-request reference/digest and normalized Core request fingerprint;
- exact `H0`;
- exact candidate `C` and candidate tree identity;
- pinned Core/runtime/contract/delivery identities;
- the independently derived expected semantic result/tree or equivalent Core-conformance digest;
- the final conformance pass/fail result.

The repository Durable Object accepts `AUDITED -> PUBLISHING` only from an audit receipt which proves this request-conformance gate for the same exact request, base, candidate, operation, and compatible release generation already held by the lane.

### 7. This gate does not weaken existing audit checks

Request conformance is additive. The auditor still independently checks all previously required trust/repository/index/binding/scope invariants, and deterministic audit disagreement still suspends/reconciles the lane.

A candidate which is the exact Core-derived semantic result but has stale/forged indexes, invalid trust/control-plane state, unauthorized unrelated changes, wrong parent, wrong repository/ref binding, or mismatched evidence still fails audit.

## Consequences

- The finalizer can no longer make arbitrary valid in-scope semantic bytes publishable merely by copying a legitimate request fingerprint into the commit.
- The fresh auditor becomes an actual request-to-output integrity boundary rather than only a repository-validity gate.
- Provider code still does not learn or reimplement memory semantics; Core remains the one deterministic implementation.
- Candidate publication remains exact-`C`; a replay-created verification commit is never substituted for the audited candidate.
- The auditor may incur additional local Core work. Implementations may optimize with a Core-owned conformance verifier, but may not optimize away the semantic comparison itself.
- The existing finalizer/auditor privilege split, evidence-role separation, exact publication, publication fencing, protected-history, and rollback-recovery architecture remain unchanged.
- Because this ADR records a material correction found by the architecture-freeze attack against `4dbdef5c...`, that review does **not** satisfy the zero-edit gate. Implementation remains blocked until a later full review of the new synchronized planning head itself produces zero required architecture/planning edits.

## Alternatives considered

### Trust the request fingerprint in the candidate commit

Rejected. A finalizer which knows the request can copy its legitimate fingerprint while writing different semantic bytes.

### Treat path scope + hard validation + fresh Index as semantic conformance

Rejected. They prove that the candidate is structurally valid and confined, not that its in-scope content is what the sealed request asked Core to produce.

### Have hosted TypeScript/Worker code compute the expected mutation

Rejected. That would create the second mutation implementation the architecture is explicitly designed to avoid.

### Require the auditor's replayed commit SHA to equal C

Rejected as a universal rule. Current Git commit creation may include nondeterministic author/committer timestamps even when semantic tree and mutation message are identical. The required invariant is exact request-derived semantic state plus exact audited publication object, not accidental equality with a separately replayed commit object.

### Let the auditor repair C when replay shows a difference

Rejected. Audit remains observational/fail-closed. A mismatched candidate is an integrity failure and cannot be rewritten into a new candidate under the same audit.

## Verification

Implementation satisfies this ADR only if tests/evidence demonstrate at minimum:

1. a legitimate candidate produced by the pinned Core from exact `H0` + sealed request passes independent request-conformance audit;
2. a finalizer changes proposed Markdown content while preserving the legitimate request fingerprint, target path, valid schema, and fresh Index: audit fails;
3. a finalizer changes an in-scope JSON semantic field while preserving valid repository/index state: audit fails;
4. a finalizer adds/removes/changes a relationship/lifecycle side effect not derived by Core from the request: audit fails;
5. a finalizer supplies a forged but plausible changed-path/result manifest: the manifest cannot make a mismatching candidate pass;
6. correct canonical-memory bytes with a stale/forged Index still fail the existing strict Index check;
7. a candidate with wrong parent, extra parent, unrelated path change, control-plane change, or wrong repository/ref binding fails independently of semantic conformance;
8. if fresh Core verification of `H0` + request produces `NO_OP`, `ALREADY_COMMITTED`, stale, invalid, or another noncandidate result while the finalizer asserts candidate `C`, audit fails;
9. the auditor's conformance path reads the exact immutable sealed request/evidence selected by the DO and not finalizer-substituted request bytes;
10. provider code contains no second hand-written mutation-semantic switch whose result is treated as conformance authority;
11. when replay is used and Git commit timestamps differ, equal authorized semantic tree/message/parent may pass without substituting replay commit for exact candidate `C`;
12. any allowed nondeterministic commit metadata is explicitly bounded and cannot alter tree/parent/request/mutation bindings;
13. audit receipt binds exact request digest/fingerprint, `H0`, `C`, candidate tree, independently derived conformance result, and release/runtime identities;
14. the DO rejects a conformance receipt for the wrong request/base/candidate/generation even when all referenced objects individually exist;
15. auditor scratch writes remain local/disposable and cannot modify candidate evidence or canonical Git;
16. finalizer still cannot create authoritative audit evidence and auditor still cannot publish; and
17. disabling coarse path-scope checks does not make conformance sufficient by itself: all existing independent trust/repository/index/scope checks remain required.