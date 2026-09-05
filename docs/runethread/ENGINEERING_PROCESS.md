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

Phase 2.5 compatibility hardening is complete: contract v8 / Runethread v0.8.0 is released, the public template and known private memory repository are migrated, runtime-release / contract-release separation is part of the compatibility model, and ADR-012 through ADR-016 are accepted.

Phase 2.6 Memory Write Delivery Pipeline is the current milestone. Phase 3 MCP implementation is blocked until Phase 2.6 satisfies issue #20.

For Phase 2.6 work:

- start from freshly verified `main` and ADR-012/ADR-013 invariants as amended/qualified by ADR-014, ADR-015, and ADR-016;
- treat the contract-v9 migration implementing ADR-015 as the first implementation prerequisite after architecture freeze; normal hosted mutation admission MUST reject contract-v8 repositories rather than silently omitting v8-required project current-state synchronization;
- keep Core's existing development pipeline intact; no reduced-safety fast mode;
- keep hosted provider code outside `runethread/core` (target `runethread/hosted`) and preserve local/offline Core operation;
- use one repository-runtime Durable Object per immutable GitHub repository identity as sole hosted lane/operation-state authority;
- bind each hosted repository explicitly to immutable repository identity + App installation + canonical branch ref + last accepted revision; normally discover the ref from default branch at adoption, but never silently follow branch/default changes, deletion, transfer, or authorization change;
- keep bounded queue, active operation, phase/execution generation, retries/backoff/deadlines, evidence refs, canonical-ref binding, and lane state in transactional DO SQLite;
- do not add Cloudflare Workflows in v1; use idempotent DO `drive()` plus at-least-once alarms with explicit rescheduling for prolonged retryable failures;
- treat async interleaving as real: atomically claim phase + execution generation before external work, persist claim, perform Container/R2/GitHub I/O without long `blockConcurrencyWhile()`, then compare active operation/phase/generation before accepting the result; obsolete-generation outputs cannot advance state;
- ensure only one authoritative external action exists for an active phase/generation; recovery may retry the same generation/action identity only under idempotent semantics;
- return `ACCEPTED` only after durable request/operation state and recoverable alarm scheduling are established; exact resubmission/status/cancel/recovery repairs missing alarms for stored work;
- store private request/candidate/finalization/audit bodies in private content-addressed/no-overwrite short-retention storage, not ordinary DO/log/status plaintext;
- do not expose generic authoritative evidence-write authority to Container roles: finalizer may submit only its exact candidate/finalization artifact class for current generation, auditor may submit only exact audit artifact class, and every write is repository/attempt/phase/generation/key/digest/create-if-absent bound through a private evidence boundary;
- keep referenced request/candidate/finalization/audit evidence alive for queued/active/retrying/audited/publishing/reconciling operations; provider TTL cannot delete live evidence and only unreferenced/orphan or safely terminal evidence after its bounded recovery/incident window is GC-eligible;
- separate hosted attempt identity from Core idempotency identity; hosted identity binds repository/canonical-ref/request digest, while Core owns committed retry/conflict semantics;
- isolate long-lived GitHub App key in private internal gateway; public API has no publication binding and ordinary runtime App permissions exclude Administration/Workflows;
- serialize whole hosted finalization/audit/publication operation per repo while preserving ADR-003 committed-idempotency-before-stale ordering; stale work may need cold source preflight but stops before candidate/Index/package/audit once proven uncommitted;
- run real Runethread Core/Git finalizer in attached Container; cold target at most one source clone/fetch, with reachable idempotency history retained and repository-controlled Git execution surfaces disabled;
- every fresh finalization resets/reconstructs to direct observed canonical ref/revision before Core; never reuse unpromoted local candidate history as canonical evidence;
- make finalization idempotent by persisting complete candidate evidence first and immutable attempt/generation-bound receipt last; a valid receipt wins, otherwise restart from canonical state;
- let `ApplyMutation` preserve its own committed-retry-before-stale, Index write, validation, commit, and local-only fast-forward semantics once;
- treat `NO_OP` as Core-validated terminal with no candidate/audit/publication;
- separate request-local mutation failure from canonical repository/trust/compatibility/ref-binding failure; unhealthy canonical base fails closed at lane level;
- bind candidate evidence to repository/canonical-ref/attempt/generation/idempotency/H0/C/tree/request/runtime/delivery/contract identities and digests;
- audit exact C in fresh reduced-privilege Container/DO context, no repair/publication authority, with immutable generation-bound audit receipt; finalizer must be unable to manufacture that authoritative audit receipt;
- persist deterministic audit disagreement as suspension/reconciliation before releasing active lane;
- only repository DO may atomically transition `AUDITED -> PUBLISHING`, after rechecking current generation, cancellation, lane state, exact evidence, authorization, bound ref == H0, and barriers;
- make cancellation vs publication a local atomic race: whichever terminal/PUBLISHING transition wins defines the boundary;
- keep the long-lived App private key in the gateway Worker and, only after durable `PUBLISHING`, mint a short-lived one-repository minimum Contents-write installation token to a minimal trusted publisher executor/Container;
- publisher executor imports exact audited Git objects, performs no source clone/semantic mutation/repair/audit, and performs only exact bound-ref `H0 -> C` Git-protocol publication; it never constructs `C2` and discards token/state after the attempt;
- do not treat GitHub REST `Update a reference` with `force=false` as exact expected-old CAS; a future API path may replace the publisher executor only after authoritative documentation and integration tests prove exact candidate identity and true atomic expected-old-`H0` semantics;
- after ambiguous publication, bound ref == C means committed, == H0 permits retry of same exact authorized publication, any other value means reconciliation;
- after success confirm only bound ref == C; no redundant full validation cycle;
- signed push webhooks are hints only and always trigger direct read of bound canonical ref;
- distinguish proven uncommitted stale work from unexpected bound-ref movement during active operation;
- use one Free/paid hosted architecture; paid ruleset protection is optional defense-in-depth;
- version hosted release/protocol and treat incompatible Worker/DO/Container/evidence/publisher/canonical-ref changes as barriers; no assumption of atomic provider rollout;
- enforce explicit resource/private-data/log/retention limits and threat-model hosted plaintext processing;
- after contract-v9 migration, keep project orientation/current-state prose outside atomic memory dual-write transaction;
- remove push-on-every-normal-memory full Actions validation only through managed control-plane rollout;
- measure acquisition/bytes/idempotency-stale/finalization/package/audit/publication/publisher-executor/alarm/interleaving/provider startup latency and cost separately.

### Phase 2.6 architecture-freeze gate

Before implementation begins, the exact current ADR/planning head must complete a fresh adversarial architecture review covering correctness, contract compatibility, state ownership, component necessity, async interleaving, concurrency, crash/retry/ambiguous-response behavior, privilege/evidence-authority boundaries, evidence retention, exact remote publication, canonical-ref lifecycle, deployment/version skew, privacy/resource limits, and avoidable latency/duplication.

The review passes only if it produces **zero required architecture or planning edits**. Any material correction, simplification, missing invariant, contract prerequisite, or changed implementation boundary must be recorded first and resets the gate; the full review then repeats against the new exact head. Green CI or a review of an older head does not satisfy the gate.

The attack review completed on 2026-09-05 against pre-amendment head `68549677e0fbb76b0018ce3aaa574c1d1ba4e1bb` found material changes and produced ADR-016. It therefore **failed** the zero-edit gate. No new attack review is started automatically after those edits; implementation remains blocked until a later explicitly requested full review of the new exact head passes with zero edits.

Prototype questions may remain only when an accepted invariant-preserving fallback already exists and architecture does not depend on guessing the outcome. Phase 2.6 now has a concrete exact Git-protocol publication fallback rather than relying on the unproven REST ref-update path.

Installing or materially changing hosted Phase 2.6 itself is a control-plane barrier and uses full Core/hosted release/downstream process.

For Phase 3, MCP remains transport over MemoryService and the established delivery lifecycle. Re-check current MCP SDK/protocol/auth requirements only when Phase 3 becomes current; do not add MCP dependencies during Phase 2.6 merely because planned next.

Any evidence that changes repository semantics still passes normal contract/migration gates rather than being smuggled in as hosted-adapter detail.