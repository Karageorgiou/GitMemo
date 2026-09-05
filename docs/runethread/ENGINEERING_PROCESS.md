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

Phase 2.5 compatibility hardening is complete: contract v8 / Runethread v0.8.0 is released, the public template and known private memory repository are migrated, runtime-release / contract-release separation is part of the compatibility model, and ADR-012/ADR-013/ADR-014 are accepted.

Phase 2.6 Memory Write Delivery Pipeline is the current milestone. Phase 3 MCP implementation is blocked until Phase 2.6 satisfies the exit criteria tracked in issue #20.

For Phase 2.6 work:

- start from a freshly verified current `main` and the accepted ADR-012/ADR-013 invariants as amended by ADR-014;
- keep the existing Core development pipeline intact; do not create a reduced-safety "fast mode" for Core engineering changes;
- keep provider-specific hosted implementation outside `runethread/core` (target `runethread/hosted`), pin exact verified Core/runtime/container identities there, and preserve local/offline Core operation without Cloudflare;
- treat Cloudflare as the primary hosted execution/control-plane provider for Phase 2.6;
- use one repository-runtime Durable Object per immutable GitHub repository identity as the sole hosted lane/operation-state authority, with transactional SQLite for bounded queue state, one active operation, retry/backoff deadlines, evidence references, suspension/maintenance/reconciliation, and last accepted canonical revision;
- use Cloudflare's Container/Durable-Object relationship so the repository runtime manages its attached finalizer Container instead of adding a second per-repository coordination object solely for lifecycle;
- do not add Cloudflare Workflows in Phase 2.6 v1; drive the one active operation through an idempotent Durable Object state machine plus at-least-once alarms, with explicit retry/backoff state and rescheduling so correctness does not depend only on the platform's finite automatic alarm retries;
- store full sealed request/candidate/audit bodies only in private content-addressed/no-overwrite object storage with short explicit retention; keep only opaque references/digests and bounded metadata in Durable Object state, logs, and client-visible status;
- keep hosted operation-attempt identity separate from Core idempotency identity; hosted identity binds immutable repository identity plus exact sealed-request digest while Core remains authoritative for semantic committed retry and idempotency conflict;
- keep the public API Worker free of the GitHub App private key and publication capability; hold the long-lived key only in a private internal GitHub gateway/publisher service, and request no Administration or Workflows permission for ordinary memory delivery;
- authenticate callers and verify explicit authorization to the installed GitHub repository; never authorize mutation merely from a caller-supplied repository identifier;
- serialize the whole hosted finalization/audit/publication operation per repository in v1 while preserving ADR-003's committed-idempotency-before-stale ordering; without a complete canonical idempotency index, stale work may require cold Container/source acquisition, but it must stop before candidate construction, Index write, packaging, and audit;
- run the real Runethread Go/Core + Git finalizer in a Container and invoke existing MemoryService rather than reproducing canonical pathing, lifecycle, provenance, indexing, idempotency, concurrency, or Git transaction semantics in provider code;
- on every fresh finalization invocation, restore/reconstruct the working clone to the directly observed remote canonical revision before calling Core; never retry against a surviving unpromoted local candidate because its Git history can make `FindAppliedOperation` mistake candidate evidence for canonical committed evidence;
- make finalization itself idempotent through complete candidate evidence written first and an immutable attempt-bound finalization receipt written last with create-if-absent semantics; a retry uses a valid receipt or restarts from remote canonical state when no authoritative receipt exists;
- preserve reachable commit history required for `FindAppliedOperation`; do not use shallow history that can hide committed idempotency evidence, and harden Git execution against repository-controlled hooks/submodules/filters/config/credential execution surfaces;
- let `ApplyMutation` perform its existing committed-retry preflight, stale check, Index v2 write, hard validation, commit creation, and local-only fast-forward once; do not add a redundant equivalent generation/repair pass;
- treat `NO_OP` as a Core-validated terminal outcome and do not invent Git evidence or skip Core validation merely because the requested operation is labeled `noop`;
- distinguish request-local mutation failure from accepted canonical repository/trust/compatibility failure; an unhealthy canonical base fails closed at the lane rather than generating endless independent request failures;
- bind exact candidate `C` to private immutable evidence containing operation/idempotency, `H0`, tree, request fingerprint, runtime/image/delivery identity, contract identity, and cryptographic digests; evidence/receipt mismatch is an integrity failure and orphan package data without a valid receipt is nonauthoritative;
- audit exact `C` in a separate fresh reduced-privilege Container/DO context with hard validation, strict Index v2 freshness, candidate/request/runtime binding, and expected-diff checks; the auditor performs no repair and has no publication credential;
- make audit completion idempotent through immutable audit evidence/receipt, and persist deterministic audit disagreement as lane suspension or conservative reconciliation before releasing the active operation;
- require exact audit evidence to return to the repository Durable Object; only that lane authority may atomically transition `AUDITED -> PUBLISHING` after rechecking cancellation, lane state, evidence identity, authorization, and direct canonical ref state;
- publish only exact DO-authorized audited `C` through atomic expected-old-revision non-force compare-and-swap; use clone-free Git-object publication only after integration evidence proves exact candidate identity, otherwise use an exact-candidate push fallback from a minimal privileged environment;
- once `PUBLISHING` is durably entered, cancellation is no longer a correctness mechanism; after crash/response loss resolve exact Git state (`C`, `H0`, or unexpected) and retry only the same exact authorized publication when appropriate;
- after exact publication, synchronously confirm only `main == C`; do not add another full clone/validation cycle merely to re-prove the same immutable candidate;
- use signed GitHub push webhooks only as hints for fast observation; every relevant delivery triggers a direct canonical-ref read and stale/out-of-order webhook payloads never directly change accepted state;
- distinguish proven uncommitted stale work (`NEEDS_REPREPARE`) from unexpected canonical movement during the one active hosted operation (`RECONCILIATION_REQUIRED`);
- use one normal hosted architecture for GitHub Free and paid private repositories, treating paid branch/ruleset protection as optional defense-in-depth rather than a correctness prerequisite;
- version the hosted delivery protocol/release and treat breaking Worker/Container/control-path changes as control-plane barriers; provider rollout is not assumed atomic, so incompatible changes require draining/maintenance or versioned blue/green isolation;
- preserve stable idempotent crash/lost-response recovery, cancellation-before-publication, audit-failure suspension, explicit reconciliation, and exclusive control-plane barriers;
- enforce explicit request/rate/repository/artifact/runtime/retry/operation-history/log/retention/privacy limits so quota/provider failure leaves canonical Git unchanged;
- keep Phase 2.6 v1 singleton-only; semantic dependency quantification, neighboring-operation batching/coalescing, and automatic semantic re-preparation require later accepted design work;
- do not turn project orientation/current-state prose into a required atomic-memory dual write; treat future refresh of those views as a separate projection/materialized-view concern;
- remove push-on-every-normal-memory full GitHub Actions validation from the hosted data-plane path only through the proper managed workflow/control-plane rollout, and retain Actions where independent repository-health, migration, recovery, or control-plane validation proves a distinct invariant;
- measure cold/warm source acquisition, bytes transferred, idempotency/stale preflight, finalization, candidate packaging, audit, publication, alarm/retry overhead, and total latency/cost separately so later optimization is evidence-driven.

### Phase 2.6 architecture-freeze gate

Before implementation begins, the exact current ADR/planning head must complete a fresh adversarial architecture review covering correctness, state ownership, concurrency, crash/retry behavior, privilege boundaries, deployment/version skew, privacy/resource limits, and avoidable latency/duplication.

The review passes only if it produces **zero required architecture or planning edits**. Any material correction, simplification, missing invariant, or changed implementation boundary must be recorded first and resets the gate; the full review is then repeated against the new exact head. Green CI or a previous review against an older head does not satisfy this gate.

Prototype questions already explicitly delegated by accepted ADRs may remain open only when a safe invariant-preserving fallback exists and the architecture does not depend on guessing the result.

Installing or materially changing the hosted Phase 2.6 mechanism is itself a control-plane barrier. Roll it out through the full Core/hosted release/downstream process rather than using the new data-plane write path to install itself.

For the later Phase 3 MCP work, retain the existing rule that MCP is transport over MemoryService and the established delivery lifecycle. Phase 3 may expose both local/offline MemoryService and remote hosted memory delivery; re-check current authoritative MCP SDK/protocol/authentication requirements only when Phase 3 becomes current, and do not add MCP dependencies during Phase 2.6 merely because they are planned next.

Any new evidence that changes repository semantics must still pass the normal contract and migration gates above rather than being smuggled into Phase 2.6 as a hosted-adapter detail.