# GitMemo Taxonomy and Canonical Vocabulary — V1

## Purpose

This document defines the canonical vocabulary used by GitMemo and gives AI/LLM operators rules for avoiding taxonomy drift.

This is an operational specification. Reuse existing canonical terms whenever they already represent the intended concept. Do not create spelling variants, stylistic variants, or near-synonyms merely because they seem natural in the current conversation.

The JSON schema defines which values are structurally legal. This document defines what those values mean and how to choose them.

---

# 1. General canonicalization rules

For machine identifiers such as project slugs, topic slugs, and tags:

- use lowercase ASCII;
- use kebab-case;
- do not use spaces;
- do not use underscores;
- do not use camelCase;
- do not encode dates or transient state unless the concept itself requires them;
- prefer stable domain concepts over task-specific wording;
- reuse an existing canonical term when its meaning is equivalent.

Before introducing a new project, topic, tag, or entity kind, an LLM SHOULD search existing memory metadata and indexes for an equivalent concept.

Do not create both `android-architecture`, `android_architecture`, `androidArchitecture`, and `android-architectural-design` when they represent the same retrieval concept.

Aliases belong in a memory item's `aliases` array when they are useful search phrases. They do not require parallel canonical topic/tag identifiers.

---

# 2. Memory types

Exactly eight V1 memory types exist.

## `fact`

A reasonably stable piece of information worth retaining. Use when the primary semantic content is “this is/was true.” Do not use for a choice (`decision`), a temporary snapshot (`state`), or an unresolved item (`open_loop`).

## `preference`

A durable preference, working style, interaction rule, or constraint about how the user wants work performed or answers produced. Do not broaden the scope beyond what was established.

## `decision`

A meaningful choice whose rationale, alternatives, consequences, or conditions have durable value. Use when “we chose X” is central.

## `state`

A historically meaningful project/system snapshot. Use sparingly. Routine progress belongs in current-state documents or the underlying project history unless the snapshot itself has durable historical value.

## `open_loop`

Something unresolved that should remain discoverable for later action or decision. Operational status lives in `open_loop_status`.

## `correction`

A durable record that a previous understanding was wrong or materially incomplete. Do not use for a normal state transition where the earlier state was correct at the time.

## `milestone`

A significant completed event or transition worth preserving historically. Do not create a milestone for every completed task.

## `reference`

Durable procedural or explanatory knowledge worth retrieving later that does not primarily fit another memory type. Do not turn GitMemo into a general encyclopedia or duplicate authoritative source repositories.

---

# 3. Lifecycle

## `active`

The memory remains valid evidence and has not been withdrawn or replaced as canonical understanding. `active` does not mean “currently happening.” An old milestone can remain active indefinitely.

## `superseded`

A newer memory replaces this memory as the canonical understanding for the relevant scope. The old memory remains useful historical evidence. A superseded memory MUST have at least one valid incoming `supersedes` edge from another memory. This is a repository-wide validator rule.

## `withdrawn`

The memory should no longer be relied upon as evidence, for example because it was created erroneously or should not have been admitted. Withdrawal is not a substitute for secret-removal or Git-history remediation.

---

# 4. Importance

## `normal`
Default. Useful durable memory with no special retrieval urgency.

## `high`
Should receive stronger retrieval priority because overlooking it could materially impair future work.

## `critical`
Reserve for rare memories where failure to retrieve the item could cause serious correctness, safety, security, or major project-integrity problems. Do not use `critical` merely because a memory is interesting.

---

# 5. Provenance basis

## `user_stated`
The user explicitly provided the information. This can be authoritative for the user's own decisions and preferences but does not automatically verify external factual claims.

## `project_verified`
The claim was checked against an authoritative project source, such as source code, configuration, tests, or project documentation.

## `external_verified`
The claim was checked against an appropriate external authoritative or reliable source.

## `derived`
The claim follows deterministically from identified evidence. A derivation must be reproducible from its sources.

## `inferred`
The claim is model judgment or pattern inference rather than directly established evidence. Do not silently upgrade inference to fact.

## `migrated`
The memory was imported from older stored context whose original provenance is incomplete or cannot be represented precisely. Migration provenance should still identify the migration source.

---

# 6. Confidence

Confidence describes certainty in the memory claim. It is independent of provenance basis.

- `high`: strongly established for the represented scope.
- `medium`: useful and likely correct, but meaningful uncertainty remains.
- `low`: speculative, weakly supported, incomplete, or retained because the uncertainty itself has future value.

Do not use confidence as a replacement for recording provenance.

---

# 7. Relationship types

Relationships are stored only as canonical outgoing graph edges.

- `related_to`: broad semantic relationship; use sparingly.
- `depends_on`: current memory materially depends on target memory.
- `supersedes`: current memory replaces target as canonical understanding.
- `corrects`: current memory establishes target was wrong or materially incomplete.
- `conflicts_with`: unresolved incompatible claims or guidance.

A correction that fully replaces old canonical truth will commonly contain both `corrects` and `supersedes` to the same target.

Reverse/incoming relationships are derived by tooling or indexes. Do not persist fields such as `superseded_by` or `corrected_by`.

---

# 8. Open-loop status

Only `open_loop` memories have `open_loop_status`.

- `open`: unresolved and actionable.
- `blocked`: unresolved because a concrete blocker prevents progress.
- `deferred`: intentionally postponed.
- `resolved`: satisfactorily closed; memory may remain active as history.
- `cancelled`: will not be pursued.

---

# 9. Sensitivity

- `routine`: ordinary project/personal working context suitable for this private repository.
- `private`: meaningfully personal, non-public, or project-confidential but appropriate to persist.
- `sensitive`: deserves additional caution but is still deliberately eligible for GitMemo.

There is deliberately no persisted `never-store` value. If information is forbidden to store, it MUST NOT be written at all.

Credentials, authentication secrets, recovery codes, private keys, and similar secret material are always excluded.

---

# 10. Projects

`projects` contains canonical project slugs, not arbitrary labels.

A project is a durable body of work with enough continuity that future memory retrieval may need to scope by it.

Examples: `recharge`, `gaio-pepper`, `gitmemo`.

Do not create separate project slugs for branches, individual tickets, phases, or one-off tasks.

A memory may belong to multiple projects. An empty project list is valid for general/profile memories.

Project slug registration will eventually be enforced by repository-wide tooling. Until then, operators MUST search for existing project usage before creating a new slug.

---

# 11. Topics

Topics are durable conceptual retrieval categories such as `android-architecture`, `release-engineering`, `memory-systems`, and `human-robot-interaction`.

Topics should be broader and more stable than tags. Do not use topics for every library, filename, function, or transient subtask.

---

# 12. Tags

Tags are lightweight retrieval/filter vocabulary such as `r8`, `gradle`, `refactor`, `schema`, and `validation`.

Before creating a new tag:

1. search existing tags;
2. reuse an exact semantic match;
3. prefer an established common technical term;
4. avoid plural/singular duplication when meaning is identical;
5. avoid spelling variants;
6. do not repeat every title word as a tag.

Search phrases and natural-language synonyms generally belong in `aliases`, not as duplicate tags.

---

# 13. Aliases

Aliases are natural-language search expressions and may contain spaces, capitalization, product terminology, abbreviations, or synonyms.

For an R8 memory, examples might include `architecture before R8`, `ProGuard sequencing`, `refactor before minification`, and `Android optimizer`.

Aliases are not canonical ontology terms. Their purpose is retrieval recall.

Do not add dozens of speculative aliases. Prefer phrases a future user or LLM could realistically use.

---

# 14. Entities

Entities identify named things involved in a memory and contain `kind` plus `name`.

Likely V1 entity kinds include:

- `person`
- `organization`
- `project`
- `repository`
- `technology`
- `library`
- `device`
- `document`
- `service`

Do not create a new entity kind when one of these already fits. Entity kinds remain extensible because real usage may reveal missing categories.

Do not use entities merely to repeat every noun in the memory.

---

# 15. Temporal semantics

`created_at` is when the GitMemo memory object was created.

`updated_at` is when the memory object was last materially modified.

`effective_from` is when the represented fact, state, decision, preference, or condition became applicable, when known.

`effective_until` is when it ceased to apply, when known.

Do not substitute memory creation time for an unknown real-world effective date. Use `null` when an effective boundary is genuinely unknown or not applicable.

Repository validation MUST reject `updated_at < created_at` and `effective_until < effective_from`.

---

# 16. Canonical vocabulary decision rule for LLMs

When choosing metadata vocabulary:

1. Look for an exact existing canonical term.
2. Look for an existing semantically equivalent term.
3. Prefer the existing term even if another spelling feels more natural.
4. If no suitable term exists, choose the simplest stable kebab-case concept.
5. Do not create multiple new terms to cover synonyms. Put synonyms in `aliases`.
6. If classification is materially uncertain, preserve the uncertainty rather than inventing a fake distinction.

Consistency across years is more valuable than stylistic perfection in one memory.
