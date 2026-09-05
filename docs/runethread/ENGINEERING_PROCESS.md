# Runethread engineering change protocol

Status: **Active project policy**

Runethread preserves durable user memory. A defect in repository compatibility, trust, migration, or mutation logic can compound across releases and user repositories. Development therefore optimizes for **recoverability, explicit evidence, small reviewable changes, and early detection of wrong assumptions** rather than speed of implementation.

This document governs substantive changes to `runethread/core`. Accepted ADRs govern architecture; this document governs how changes to that architecture are planned, implemented, verified, reviewed, released, and corrected.

---

## 1. Core development principles

1. **Live repository state beats remembered context.** Previous-chat summaries and planning notes are orientation material, not verification.
2. **One canonical owner per fact.** Project source, architecture, ADRs, compatibility policy, and engineering procedure live in the project repository. Personal memory may point to them but must not replace them.
3. **Deterministic invariants require deterministic tests.** Semantic judgment may identify intent, but storage, versioning, trust, validation, migration, concurrency, and release invariants must be enforced by code/tests where practical.
4. **A green build is necessary, not sufficient.** Tests prove only what was asserted. Every meaningful change also requires impact, backward, forward, negative, and cross-surface review.
5. **Published history is immutable evidence.** Do not reinterpret released contract semantics to avoid a version or migration.
6. **Historical compatibility uses historical state.** Never manufacture an old released state with a new generator when its bytes or semantics may differ.
7. **Validation is observational.** CI may test and report; it must not repair source or push commits to the branch it validates.
8. **Unexpected state stops writes.** A contradiction, race, stale base, unexplained diff, or incomplete evidence is a reason to investigate before proceeding.
9. **Failures must be recoverable.** Prefer explicit migrations, snapshots, optimistic concurrency, exact-source recognition, and rollback over permissive repair.
10. **Evidence is attached to exact revisions.** Claims about tests, diffs, releases, or downstream migrations must identify the exact commit/release they verified.

---

## 2. Change classes

Classify every substantive change before implementation. A change may belong to multiple classes.

| Class | Examples | Minimum additional review |
| --- | --- | --- |
| Documentation-only | explanatory docs, non-normative examples | verify no normative/control-plane impact |
| Runtime-only | performance, adapters, internal implementation | forward compatibility and API behavior |
| Public API / CLI | JSON fields, commands, exit codes | compatibility, callers, docs, integration tests |
| Dependency / toolchain | Go version, module, SDK | support floor, transitive deps, licenses, advisories, dependency graph |
| Contract | operational semantics or any `ContractPaths()` file | contract release/version, migration, historical fixture, downstream repin |
| Schema / repository format | sidecar shape, canonical layout | explicit version bump, migration, fixtures, validator/index impact |
| Trust / lock | lock envelope, trust authority, digest rules | threat model, bootstrap, migration, tamper tests |
| Index format | committed generated layout/semantics | version bump when format changes, rebuild/compatibility tests |
| Bootstrap | onboarding machine protocol/setup behavior | protocol compatibility, old/new repository discovery |
| Migration | supported source/target transitions | exact source fixture, rollback, canonical-data preservation |
| Release / packaging | release workflow, artifacts, signing | immutable publication and artifact verification |
| Downstream repository | template or private memory migration | exact before/after invariants and post-merge validation |

**Semantic impact controls classification.** A runtime-code change that changes behavior promised by the vendored operational contract is a contract change even if no contract file was edited initially.

---

## 3. Preflight gate — before editing

For a substantive change, capture and verify the following before the first source write:

- current `main` commit SHA and, when useful, tree SHA;
- branch point and intended branch name;
- current published Runethread release and relevant version dimensions;
- active repository ruleset / required checks;
- current CI status on `main`;
- relevant implementation files and tests;
- relevant accepted ADRs and normative contract files;
- supported historical source fixtures/releases affected by the change;
- current downstream template/private-repository state when the change may affect them.

If any verified fact contradicts the plan, stop and revise the plan before editing.

Do not create a branch from a floating assumption such as “latest main” remembered from an earlier turn. Use a freshly verified exact SHA.

---

## 4. Impact matrix — before implementation

Record an explicit impact decision for every relevant surface:

| Surface | Questions |
| --- | --- |
| Canonical memory data | Can Markdown/JSON bytes, UUIDs, provenance, lifecycle, relationships, or meaning change? |
| Project/user data | Can `projects/` or unrelated user files change? |
| `.runethread` metadata | Does config/lock meaning or representation change? |
| Operational contract | Do normative semantics or any `ContractPaths()` bytes change? |
| Schema | Does a valid/invalid sidecar set change? |
| Repository format | Does canonical layout/identity change? |
| Index | Does committed generated layout or interpretation change? |
| Trust | Does authority, digest verification, bootstrap trust, or tamper behavior change? |
| Bootstrap | Does setup/discovery/version-resolution behavior change? |
| MemoryService / CLI / API | Do request/result/error semantics change? |
| Migration | Which exact historical states must reach the new state? |
| Versioning | Which dimensions must advance, and why? |
| Release tooling | Are tags/assets/install paths affected? |
| Dependencies / Go | Does the build floor, supply chain, license set, or supported platforms change? |
| Template | Must `runethread/memory-template` change? |
| Private memory | Must an existing user repository change? What invariants must remain byte-identical? |
| Security/privacy | Are privileges, secrets exposure, prompt-injection boundaries, or public/private data boundaries affected? |
| Documentation | Which normative and non-normative docs encode the old assumption? |

A blank cell is not evidence of “no impact.” State why the surface is unaffected.

---

## 5. Contract gate

The operational contract is security- and compatibility-sensitive.

Before claiming a change is runtime-only:

1. enumerate the files returned by `ContractPaths()`;
2. read the relevant normative wording, not merely file names/digests;
3. compare the proposed implementation semantics against the currently published contract semantics;
4. inspect trust/bootstrap/versioning behavior that interprets those files;
5. search non-vendored policy/docs for the same assumption.

If implementation behavior changes a normative contract promise, **the contract changes even if the original patch did not edit a contract file**.

A genuine contract change must normally include:

- a new contract version;
- a new immutable contract release anchor;
- updated vendored normative files/digests;
- a deterministic migration from each supported prior state;
- an exact historical fixture for the prior released state;
- tamper/unsupported-source refusal tests;
- canonical-data preservation checks where representation is unchanged;
- template/downstream migration plan.

Never retroactively reinterpret an immutable published contract to avoid these requirements.

---

## 6. Historical / backward compatibility gate

For every supported source state affected by a change:

1. identify the exact released source state;
2. prefer a frozen fixture captured from the actual release output or verified release artifact;
3. verify the fixture's expected metadata/digests before migration;
4. run the current migration against that fixture;
5. verify the exact intended target state;
6. verify canonical data preservation and rollback behavior.

### Historical fixture rule

**Do not synthesize a historical repository by running the current generator and editing version numbers when released contract bytes or semantics differ.**

A current generator can only stand in for a historical fixture when byte/semantic identity has been independently proven and that equivalence itself is protected by tests.

For user-memory migrations, capture before the write:

- exact canonical UUID set/count;
- canonical file/blob or digest inventory;
- relationship closure;
- provenance-bearing sidecars;
- project files;
- trust/config state;
- index state where relevant.

---

## 7. Forward compatibility gate

Do not test only today's constants. Simulate the next plausible release relationship.

Examples:

- newer runtime with unchanged contract;
- newer contract with unchanged repository format/schema;
- older supported repository opened by newer runtime;
- unsupported newer repository opened by older/current runtime;
- newly initialized empty repository;
- already-current repository passed through `upgrade` again.

Tests should intentionally make version dimensions differ when the design says they are independent. Equal constants can hide coupling bugs.

For a runtime/contract split, for example, test something equivalent to:

```text
runtime release       vNext
contract release      vCurrent
repository pin        vCurrent
```

and separately prove that incorrectly pinning the repository to `vNext` is rejected when the runtime still embeds the older contract.

---

## 8. Negative and failure-mode gate

For relevant changes, test the ways the operation must fail safely, not only the success path.

Examples include:

- stale Git revision;
- concurrent writers;
- exact idempotent retry after lost response;
- idempotency-key conflict;
- dirty repository;
- detached HEAD;
- malformed or unknown JSON fields;
- invalid lifecycle/relationship transition;
- duplicate UUID/path collision;
- hard post-write validation failure;
- rollback failure boundaries;
- tampered trust/control-plane files;
- mixed/unknown historical state;
- unsupported newer schema/contract/repository format;
- missing/stale generated indexes;
- interrupted release publication;
- incomplete artifact set.

Failure tests must assert the repository state that remains afterward, not merely the returned error.

---

## 9. Implementation discipline

- Work from a dedicated branch created from an exact verified base SHA.
- Keep commits small and coherent enough to explain/revert.
- Avoid blind search/replace and unrelated refactoring.
- Re-read an existing file immediately before replacing it through an API when stale content is possible.
- Prefer deterministic transformations and explicit assertions over speculative patches.
- Do not force-push ordinary development branches.
- Do not modify `main` directly.
- Do not run multiple source writers concurrently on the same branch.
- Do not use GitHub Actions as a source-editing agent or branch fixer.
- After each significant write, verify the resulting branch head and changed-file surface.

If the underlying design premise changes materially during implementation, prefer abandoning/closing the exploratory PR and restarting from clean `main` over accumulating compatibility cruft or misleading history.

---

## 10. Agent/tool limitations that must be compensated for

AI tooling has failure modes that become more dangerous as the repository grows.

### Stale remembered state

A prior conversation may contain a once-correct SHA, version, branch, or file fact. Always re-read live state when correctness depends on it.

### Truncated tool output

A truncated tree/diff/file response is incomplete evidence. Narrow the query, page the result, or read specific files. Never infer unseen entries.

### Incomplete code search

GitHub code search can be unavailable, stale, or return incomplete results. An exhaustive audit must use a verified local checkout when available or enumerate the repository tree and inspect relevant files through repository APIs.

### Asynchronous workflow races

A workflow triggered by an earlier commit can finish after later commits. Validation workflows therefore remain read-only. Never let CI patch and push source. When multiple runs exist, tie every conclusion to the exact commit SHA.

### Synthetic history

Current code cannot be assumed to reproduce an old release. Use immutable tags/releases/frozen fixtures for historical compatibility.

### Tool-surface ambiguity

GitHub branch protection and repository rulesets can report different partial views. Verify the effective active mechanism before claiming a policy is or is not enforced.

### External dependency freshness

SDK/toolchain support changes over time. Re-check authoritative current sources at the dependency decision point rather than freezing old research into the plan.

---

## 11. Verification gate on the committed branch

Validation is performed **after** deliberate source commits and against exact committed SHAs.

Core's baseline gates should include:

```text
go mod verify
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/runethread
fresh runethread init
runethread index --check
runethread validate
```

Change-specific tests are added on top of this baseline.

CI must use read-only repository permissions unless a workflow's explicit purpose is release publication. Validation CI must never commit or push fixes.

A test result is associated with the exact commit SHA it tested. A later commit invalidates the earlier green result until the new SHA is tested.

---

## 12. Dependency and Go/toolchain gate

Do not add or upgrade a dependency merely because it is convenient.

Before a dependency/toolchain change:

1. verify the dependency is necessary at the architectural boundary;
2. prefer an official/maintained implementation when protocol correctness would otherwise become our burden;
3. verify its current stable version and supported Go floor from authoritative sources;
4. review direct and material transitive dependencies;
5. review licenses and known security advisories;
6. choose the lowest reasonable supported Go version, independently from the preferred CI/toolchain version;
7. run `go mod tidy` and `go mod verify`;
8. verify cross-platform build/release behavior;
9. verify GitHub dependency graph/Dependabot discovery after merge;
10. add a pinned vulnerability-scanning tool only when its version/update policy is deliberately owned.

Do not add a redundant GitHub dependency-submission workflow when the platform already submits Go dependency data correctly. Verify the observed graph first.

---

## 13. Draft PR review gate

Substantive work enters GitHub as a **draft PR first**.

Before readiness:

- verify base SHA and head SHA;
- inspect the canonical GitHub PR patch/file list, not only local intent;
- verify expected change class and impact matrix;
- confirm no temporary workflow/script/test harness leaked into the final diff;
- inspect API/CLI/docs naming for future ambiguity;
- check PR comments, reviews, and review threads;
- require the repository's protected `validate` check on the exact head;
- verify `main` has not moved unexpectedly or update/re-evaluate if it has;
- re-run the backward/forward/negative review mentally against the actual final diff.

If a major premise was invalidated, close/supersede the PR. Do not merge merely because its code is internally consistent.

---

## 14. Merge gate

Before merge:

1. PR is ready, mergeable, and review feedback is resolved;
2. required status checks pass on the exact head;
3. base still matches the reviewed state or the PR has been updated/revalidated;
4. final changed-file set is expected;
5. no release/downstream write has begun prematurely.

Prefer squash merge for a branch containing iterative implementation commits. Use expected-head protection so a moved head cannot be merged accidentally.

---

## 15. Post-merge gate

After merge, independently verify:

- PR is actually merged/closed;
- `main` points to the intended merge/squash commit;
- merge parent/base relationship is expected;
- commit verification/signature state where applicable;
- permanent CI passes on merged `main`;
- critical files contain the intended final semantics.

Do not begin release or downstream migration until these checks pass.

---

## 16. Release gate

A release is a separate correctness boundary.

Before publication:

- source version and release request agree;
- full tests/vet/build pass on release commit;
- smoke-init/validate/index checks pass;
- all intended platform binaries build;
- checksum set is complete;
- draft release target matches exact release commit.

After publication, independently verify:

- tag;
- target commit;
- immutable/non-draft status;
- expected asset names/count;
- checksums/artifacts when practical.

Downstream template/private migrations start only after the immutable release is independently verified.

---

## 17. Template and private-repository migration gate

For changes requiring repository migration:

1. migrate `runethread/memory-template` first;
2. require a minimal expected diff and normal bootstrap validation;
3. merge/verify template;
4. only then migrate a real private memory repository;
5. capture exact private canonical invariants before any write;
6. run the released upgrader, not development assumptions;
7. prove canonical/user-owned bytes are unchanged unless the migration explicitly requires representation changes;
8. require permanent post-merge validation on the private repository.

For canonical-data-preserving metadata migrations, use Git tree/blob identity where possible as an independent byte-preservation proof.

---

## 18. Correction / incident protocol

Mistakes are expected to be **detectable and recoverable**, never hidden.

When a mistake or unexplained state is discovered:

1. stop further writes;
2. capture current branch/main SHAs and active workflow runs;
3. identify exactly which action wrote what and when;
4. distinguish product defect, test defect, harness defect, stale assumption, and tool limitation;
5. preserve evidence; do not force-push it away merely to make history look clean;
6. correct the smallest affected layer;
7. re-run broader regression checks that should have caught the issue;
8. improve tests/process so the same class of mistake is detected earlier;
9. close/supersede invalid PRs explicitly;
10. report remaining uncertainty honestly.

If a wrong change reaches `main` or a published release, prefer an explicit corrective commit/release/migration over rewriting public history.

---

## 19. Stop conditions

Do not proceed to the next phase when any of these remain unresolved:

- live repository state contradicts the plan;
- the impact class is uncertain;
- implementation behavior and published contract wording disagree;
- an affected historical source state is not represented by trustworthy evidence;
- canonical-data preservation cannot be demonstrated for a supposedly metadata-only migration;
- tests are green only because independently meaningful version constants currently happen to be equal;
- CI results belong to an older head;
- the PR diff contains unexplained files;
- dependency/toolchain requirements are based on stale research;
- a required downstream migration/release rollback path is undefined.

A deliberate stop is a successful safety outcome, not a failure of progress.

---

## 20. Current Phase 2.6 application

Phase 2.5 compatibility hardening is complete: contract v8 / Runethread v0.8.0 is released, the public template and known private memory repository are migrated, runtime-release / contract-release separation is part of the compatibility model, and ADR-012 through ADR-022 are accepted.

Phase 2.6 Memory Write Delivery Pipeline is the current milestone. Phase 3 MCP implementation is blocked until Phase 2.6 satisfies issue #20.

For Phase 2.6 work:

- start from freshly verified `main` and ADR-012/ADR-013 invariants as amended/qualified by ADR-014 through ADR-022;
- treat the contract-v9 migration implementing ADR-015 as the first implementation prerequisite after architecture freeze; normal hosted mutation admission MUST reject contract-v8 repositories rather than silently omitting v8-required project current-state synchronization;
- transition the Runethread-managed memory-repository validation workflow through the released starter/upgrader/template/private-repository migration, preferably in the same v9 release sequence; remove redundant normal-push triggering, pin every retained external `uses:` Action to a verified full-length commit SHA, preserve exact prior managed workflow recognition, and never silently overwrite customized/unrecognized workflow state;
- align generated/current support prose with ADR-015 during the same v9 migration: project current-state/overview prose is an orientation/materialized view rather than a canonical source, project-view user bytes remain preserved, and automatic README replacement is limited to exact recognized prior managed README state rather than a broad heading/lock heuristic;
- keep Core's existing development pipeline intact; no reduced-safety fast mode;
- keep hosted provider code outside `runethread/core` (target `runethread/hosted`) and preserve local/offline Core operation;
- use one repository-runtime Durable Object per immutable GitHub repository identity as sole **live** hosted lane/operation-state authority;
- bind each hosted repository explicitly to immutable repository identity + App installation + canonical branch ref + last accepted revision and require directly observed private repository visibility for normal hosted writes; never silently follow branch/default changes, deletion, transfer, authorization change, or observed non-private visibility;
- keep bounded queue, active operation, phase/execution generation, retries/backoff/deadlines, evidence refs, canonical-ref binding, privacy eligibility, publisher-attempt state, protected proven/possible publication anchors, safety-journal checkpoint/binding epoch, and lane state in transactional DO SQLite;
- use the existing private evidence-storage boundary for a minimized append-only rollback-independent safety journal rather than adding a second queue/state machine. Journal binding/enrollment, client-visible acceptance, cancellation wins, accepted canonical anchors, publication intents/outcomes, required immutable receipt references, and independently verified terminal-success receipt references/result identities with create-if-absent/digest-bound records and no plaintext memory bodies/tokens;
- treat Durable Object PITR/recreation/detected rollback as an exclusive recovery barrier, not an ordinary retry. A restored/recreated DO cannot resume admission/publication from rolled-back generation state until it reconciles its journal checkpoint/epoch, immutable evidence, current GitHub binding/privacy, accepted operations/cancellations, protected publication anchors, terminal-success evidence, and possibly-issued publication capabilities;
- do not add Cloudflare Workflows in v1; use idempotent DO `drive()` plus at-least-once alarms with explicit rescheduling for prolonged retryable failures;
- treat async interleaving as real: atomically claim phase + execution generation before external work, persist claim, perform Container/R2/GitHub I/O without long `blockConcurrencyWhile()`, then compare active operation/phase/generation before accepting the result; obsolete-generation outputs cannot advance state;
- ensure only one authoritative external action exists for an active phase/generation; recovery may retry the same generation/action identity only under idempotent semantics;
- return `ACCEPTED` only after durable request/operation state, recoverable alarm scheduling, and rollback-independent acceptance evidence are established; exact resubmission/status/cancel/recovery repairs missing alarms for ordinary stored work, while destructive rollback enters the ADR-019 recovery barrier;
- make rollback-sensitive cancellation and accepted-anchor adoption write-ahead recoverable: a local winning transition is not exposed/released as durably final until the corresponding immutable safety record exists;
- store private request/candidate/finalization/audit/terminal-verification bodies in private content-addressed/no-overwrite short-retention storage, not ordinary DO/log/status plaintext;
- do not expose generic authoritative evidence/journal-write authority to Container roles: finalizer may submit only its exact candidate/finalization artifact class for current generation, auditor/verifier may submit only its exact candidate-audit or terminal-success-verification artifact class, and every write is repository/attempt/phase/generation/key/digest/create-if-absent bound through a private evidence boundary;
- keep referenced request/candidate/finalization/audit/terminal-verification/publication evidence alive for queued/active/retrying/audited/publishing/reconciling operations, including ADR-018 protected candidate anchors; private content may expire after its safe terminal window, while minimized ADR-019 safety-journal history required for an active binding epoch remains until explicit safe binding deletion/rebaseline policy permits removal;
- separate hosted attempt identity from Core idempotency identity; hosted identity binds repository/canonical-ref/request digest, while Core owns committed retry/conflict semantics;
- isolate long-lived GitHub App key in private internal gateway; public API has no publication binding and ordinary runtime App permissions exclude Administration/Workflows;
- serialize whole hosted finalization/audit/publication operation per repo while preserving ADR-003 committed-idempotency-before-stale ordering; stale work may need cold source preflight but stops before candidate/Index/package/audit once proven uncommitted;
- run real Runethread Core/Git finalizer in attached Container; cold target at most one source clone/fetch, with reachable idempotency history retained and repository-controlled Git execution surfaces disabled;
- every fresh finalization resets/reconstructs to direct observed canonical ref/revision before Core; never reuse unpromoted local candidate history as canonical evidence;
- make finalization idempotent by persisting complete candidate evidence first and immutable attempt/generation-bound receipt last; a valid receipt selects the finalizer's claimed result, but a success-like terminal claim does not become authoritative merely because the finalizer receipt is valid;
- let `ApplyMutation` preserve its own committed-retry-before-stale, Index write, validation, commit, and local-only fast-forward semantics once;
- for candidate-producing success, require ADR-020 independent request-to-candidate conformance before publication;
- construct hosted candidate commits under ADR-022's deterministic Core-owned envelope rather than ambient Git state: sanitize author/committer/date/config inputs, use release-defined identity and request-bound deterministic commit time, forbid unknown commit headers, and ensure two executions from the same immutable inputs derive the exact same candidate object ID;
- for `NO_OP`, require a fresh reduced-privilege Core verification of the exact immutable sealed request against exact H0 before client-visible terminal success/lane release; the finalizer cannot turn a candidate-producing request into authoritative no-op merely by claiming that result;
- for `ALREADY_COMMITTED`, require a fresh reduced-privilege canonical Core/repository idempotency lookup for the exact sealed request and accepted canonical history, preserving committed-idempotency-before-stale ordering and verifying the exact request fingerprint/mutation metadata before client-visible terminal success/lane release;
- persist immutable role-separated terminal-success evidence which the finalizer cannot create/replace, bind it to exact repository/ref/request/attempt/generation/release/canonical result state, and establish its minimal rollback-independent receipt reference/result identity before a `NO_OP`/`ALREADY_COMMITTED` result is exposed as durable terminal success;
- unsuccessful/stale finalizer results remain generation-bound/fail-closed but do not require an equivalent independent semantic replay solely to defend against denial of service by a compromised finalizer;
- separate request-local mutation failure from canonical repository/trust/compatibility/ref-binding failure; unhealthy canonical base fails closed at lane level;
- bind candidate evidence to repository/canonical-ref/attempt/generation/idempotency/H0/C/tree/request/runtime/delivery/contract identities and digests;
- audit exact C in a fresh reduced-privilege Container/DO context and independently prove request-to-candidate conformance under ADR-020: derive the expected semantic memory changes from exact immutable sealed request + exact H0 using the same pinned Core mutation semantics (or a Core-owned equivalent verifier), compare them to candidate C, then also require exact parent/binding/scope, hard validation, strict Index v2 freshness, no unauthorized unrelated/control-plane changes, and ADR-022 raw commit-envelope equality including exact expected candidate object ID; a finalizer-supplied expected diff, manifest, or commit metadata is never conformance authority;
- permit only disposable local scratch writes needed by the Core-owned conformance verifier; the auditor never repairs candidate C, never replaces its evidence, and has no canonical Git publication authority;
- bind the immutable generation-bound audit receipt to exact request digest/fingerprint, H0, C/tree, independently derived semantic + commit-envelope conformance result, and pinned release/runtime identities; finalizer must be unable to manufacture that authoritative audit receipt;
- persist deterministic audit disagreement as suspension/reconciliation before releasing active lane;
- only repository DO may atomically transition `AUDITED -> PUBLISHING`, after rechecking current generation, cancellation, lane state, exact evidence, authorization, directly observed private visibility, bound ref == H0, and barriers;
- make cancellation vs publication a local atomic race: whichever cancellation-claim/PUBLISHING transition wins defines the boundary, with ADR-019 external safety evidence required before a cancellation is durably released;
- before **any** externally effective publication/token/executor/API I/O, persist an immutable rollback-independent publication-intent record bound to repository/ref/H0/C/operation/generation/evidence/protocol/attempt and a conservative fencing horizon; a rolled-back DO which later discovers this record treats C as possibly published until proven otherwise;
- keep the long-lived App private key in the gateway Worker and, only after durable `PUBLISHING` plus publication-intent safety evidence, mint a short-lived one-repository minimum Contents-write installation token to a minimal trusted publisher executor/Container when the fallback path is used;
- publisher executor imports only the verified object closure needed for exact audited C relative to H0, performs no source clone/semantic mutation/repair/audit, ignores/rejects unrelated unreachable package objects, performs at most one exact bound-ref `H0 -> C` Git-protocol push for its publisher-attempt identity, never constructs `C2`, and has no autonomous retry loop;
- require a real expected-old ref update. GitHub REST `Update a reference` with `force=false` is not exact expected-old CAS; current GraphQL `updateRefs` documents `beforeOid`/`afterOid`, so the delegated API prototype should test exact candidate-object identity and GitHub App permission behavior before unnecessary publisher machinery is committed. Until that proof exists, the exact Git-protocol publisher is the accepted safe fallback;
- treat `PUBLISHING` as capability-bearing/in-doubt: the lane cannot be released, a later publication cannot start, and the publication generation cannot be abandoned/reused while an issued executor/token may still act; clean completion must satisfy executor termination/token-disposal policy, while ambiguous loss requires executor stop/destroy and conservative wait for any unconfirmed token expiry before resolution/retry;
- after fencing, classify the publication result conservatively as proven-not-published, proven-published, or indeterminate. Lost response, timeout, process loss, or current `ref != C` alone is never proof that C was not published;
- keep every proven or possibly published exact C as a protected history anchor. A definitive success is a durable committed fact even if a later owner rewrite removes C from current ancestry;
- resolve indeterminate publication under ADR-018: ref `C` or any current descendant containing `C` proves committed at exact C; exact `H0` may retry only the same C; a current revision that excludes C and is not H0 remains reconciliation-required even if it descends from the older H0;
- ordinary ADR-017 descendant adoption runs only after any ADR-018 protected publication anchor is resolved/preserved. A rewrite excluding proven C must restore/preserve C in canonical ancestry before normal hosted lane reopen; Phase 2.6 does not silently destructive-rebaseline committed history;
- after success confirm/reconcile current bound ref cheaply; no redundant full validation cycle;
- signed push webhooks and repository-visibility events are hints only and always trigger authoritative direct GitHub reads; they cannot clear a protected publication anchor or rollback-independent safety record;
- distinguish proven uncommitted stale work from unexpected bound-ref movement during active operation;
- ordinary reconciliation may adopt an out-of-band new canonical revision only if no protected publication anchor blocks it, the last accepted revision remains its ancestor, and the exact new revision passes trust/repository/index plus mutation/idempotency-history integrity checks; backward/sideways non-descendant rewrites or sibling descendants excluding a proven/possible C remain reconciliation-required until ancestry-preserving recovery, because Core committed-idempotency evidence lives in reachable Git history;
- treat observed non-private visibility as suspension/privacy-incident state and require explicit revalidation before resume; document that repository owners/admins remain able to change visibility outside the Git-ref CAS and Runethread does not claim atomic visibility+ref locking without Administration authority;
- use one Free/paid hosted architecture; paid ruleset protection is optional defense-in-depth;
- version hosted release/protocol and treat incompatible Worker/DO/Container/evidence/publisher/reconciliation/privacy/managed-bootstrap/canonical-ref/safety-journal/recovery/audit-conformance/terminal-success-verification/candidate-envelope changes as barriers; no assumption of atomic provider rollout;
- enforce explicit resource/private-data/log/retention limits and threat-model hosted plaintext processing plus minimized safety-journal retention/deletion;
- after contract-v9 migration, keep project orientation/current-state prose outside atomic memory dual-write transaction;
- measure acquisition/bytes/idempotency-stale/finalization/deterministic-candidate/package/request-conformance+commit-envelope-audit/terminal-success-verification/publication/publisher-or-API-path/fencing/journal-recovery/alarm/interleaving/provider startup latency and cost separately.

### Phase 2.6 architecture-freeze gate

Before implementation begins, the exact current ADR/planning head must complete a fresh adversarial architecture review covering correctness, contract compatibility, state ownership, component necessity, async interleaving, concurrency, crash/retry/ambiguous-response behavior, destructive Durable Object rollback/recreation, privilege/evidence-authority boundaries, evidence and safety-journal retention, independent request-to-candidate conformance, deterministic/full candidate commit-object envelope and object closure, independent terminal-success verification, publisher-capability lifetime, exact remote publication, proven/possible publication-history preservation, accepted-history reconciliation, repository visibility/privacy, canonical-ref lifecycle, managed-bootstrap/support rollout, workflow supply-chain immutability, deployment/version skew, resource limits, and avoidable latency/duplication.

The review passes only if it produces **zero required architecture or planning edits**. Any material correction, simplification, missing invariant, or changed implementation boundary must be recorded first and resets the gate; the full review then repeats against the new exact head. Green CI or a review of an older head does not satisfy the gate.

The attack review completed on 2026-09-05 against pre-amendment head `68549677e0fbb76b0018ce3aaa574c1d1ba4e1bb` found material changes and produced ADR-016. It therefore failed the zero-edit gate.

The next full review, explicitly started against synchronized head `0a1ea0b871105d6497754fbbee93a387cb2494b4`, found material corrections and produced ADR-017. It therefore also failed the zero-edit gate.

The following full review, explicitly started against synchronized head `a9e6db2f72c8d450753c5e70e4eea5eea2d78565`, found the indeterminate-publication/history-erasure race and produced ADR-018. It therefore also failed the zero-edit gate.

The next full review, explicitly started against synchronized head `0f7f95c8220d16121144de5d1c1a4f42978550bd`, found material destructive-control-plane-recovery and managed-support/security corrections and produced ADR-019. It therefore also failed the zero-edit gate.

The following full review, explicitly started against synchronized head `4dbdef5c08142856ba1795544795cea254193398`, found that candidate validity/binding/scope did not independently prove the candidate's in-scope semantic bytes were derived from the exact sealed request. ADR-020 therefore requires a Core-owned request-to-candidate conformance proof in the fresh audit. That review also failed the zero-edit gate.

The next full review, explicitly started against synchronized head `34421411f4501f762c9c104e45d2cc92a9c3c5cb`, confirmed ADR-020 closes the candidate-content gap but found that successful terminal `NO_OP`/`ALREADY_COMMITTED` claims still bypassed fresh independent verification. ADR-021 requires those no-candidate success claims to be independently proven before durable client-visible success or lane release. That review also failed the zero-edit gate.

The current full review, explicitly started against synchronized head `9151c9d2e1a383e79449af2963fc2c547bb49429`, confirmed ADR-021 closes that terminal-success bypass but found the exact candidate Git commit still had ambient/unconstrained metadata outside ADR-020's semantic tree/message proof. Current Core invokes `git commit` with config-provided identity, while Git author/committer environment variables can override those values; a compromised finalizer could therefore keep an authorized semantic tree/message while persisting attacker-chosen identity/private bytes in exact canonical `C`. ADR-022 makes the entire candidate commit envelope and reachable object closure deterministic/Core-owned and independently audited. **This review also fails the zero-edit gate.** No additional attack review is started in this prompt after these edits. Implementation remains blocked until a later explicitly requested full review of the new exact synchronized head passes with zero edits.

Prototype questions may remain only when an accepted invariant-preserving fallback already exists and architecture does not depend on guessing the outcome. The current GraphQL expected-old ref path is such a delegated prototype because the exact Git-protocol publisher remains a safe fallback until exact candidate-object identity and App-permission behavior are proven by integration tests.

Installing or materially changing hosted Phase 2.6 itself is a control-plane barrier and uses full Core/hosted release/downstream process.

For Phase 3, MCP remains transport over MemoryService and the established delivery lifecycle. Re-check current MCP SDK/protocol/auth requirements only when Phase 3 becomes current; do not add MCP dependencies during Phase 2.6 merely because planned next.

Any evidence that changes repository semantics still passes normal contract/migration gates rather than being smuggled in as hosted-adapter detail.