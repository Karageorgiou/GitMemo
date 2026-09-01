# MEMORY_PROTOCOL.md

## Purpose and intended reader

This document is the mandatory operating protocol for any AI assistant, language model, agent, or automation that retrieves from or modifies this repository.

Treat this document as **instructions**, not merely as descriptive documentation.

GitMemo is a persistent external memory system. Its purpose is to preserve durable, auditable, searchable context across conversations while avoiding the noise and ambiguity of storing complete chat histories.

The repository is not automatically authoritative for every kind of information. Follow the authority and verification rules below.

The normative terms **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are intentional.

Before materially modifying atomic memories, an operator MUST also follow:

- `schema/memory-item.schema.json`
- `docs/MEMORY_CONTENT_FORMAT.md`
- `docs/TAXONOMY.md`

---

# 1. Core invariants

An AI assistant using this repository MUST preserve the following invariants.

1. Atomic memories exist as a Markdown content file paired with a machine-readable JSON sidecar.
2. The Markdown file contains the useful human-readable meaning, context, reasoning, and history.
3. The JSON sidecar contains identity, classification, provenance, retrieval metadata, lifecycle information, and relationships.
4. Generated indexes are discovery mechanisms. They are not independent sources of truth.
5. Current-state and summary documents are fast orientation views. They are not replacements for atomic memories.
6. Historical information MUST NOT be silently rewritten merely to make it agree with current understanding.
7. Corrections and supersession MUST preserve meaningful history.
8. For current source-code facts, the actual project repository is authoritative.
9. An inference MUST NOT be represented as a verified fact.
10. Secrets and authentication credentials MUST NOT be stored.
11. The assistant MUST NOT claim that a repository read, validation, update, commit, or verification occurred unless it actually occurred.

---

# 2. Determine whether memory retrieval is necessary

Do not retrieve memory merely because this repository exists.

Before retrieval, determine whether historical, personal, preference, project, decision, or prior-work context would materially improve the answer.

If the current request is self-contained and does not depend on historical context, the assistant SHOULD answer without reading unrelated memories.

Requests that normally require retrieval include questions such as:

- “Where did we leave off?”
- “What did we decide?”
- “Why did we choose this architecture?”
- “What are my standing preferences?”
- “What remains unresolved?”
- “What changed since the previous approach?”
- “What did I explicitly ask you to remember?”

Requests about current implementation details may require both memory retrieval and verification against the actual project repository.

---

# 3. Determine the operating mode

Before acting, classify the task internally into one or more of these modes.

**No-memory mode:** historical context is unnecessary.

**Retrieval mode:** stored context is needed to answer correctly.

**Current-state verification mode:** a claim concerns current source code, configuration, external facts, or another authoritative live source.

**Memory-write mode:** durable information may need to be created, updated, corrected, resolved, or superseded.

When operating in memory-write mode, read the memory schema, content-format rules, and taxonomy before creating or materially modifying an atomic memory.

---

# 4. Authority rules

Do not apply a simplistic “newest file wins” rule.

Authority depends on the kind of information.

A current explicit instruction from the user takes precedence over an older stored preference for the present interaction. The older preference remains historical context unless explicitly replaced.

For current source code, build configuration, tests, dependencies, implementation, or other code facts, the actual project repository takes precedence over the memory repository.

For the content of an atomic memory, the Markdown file is authoritative for the natural-language meaning and reasoning.

For an atomic memory's identity, lifecycle, classification, provenance, search metadata, and relationships, the JSON sidecar is authoritative.

Current-state and overview documents are curated or derived views intended for fast orientation. When they conflict with properly validated atomic memories or an authoritative project source, investigate the discrepancy rather than guessing.

Generated indexes are authoritative only as indexes generated from source metadata. They MUST NOT be treated as independent evidence for a fact.

Git history is an audit trail. A previous Git revision MUST NOT be treated as current truth merely because it existed historically.

---

# 5. Retrieval procedure

Begin retrieval from the narrowest useful entry point.

For a project-specific question, first read the project's `overview.md` and/or `current-state.md` when available.

For a preference question, begin with the relevant preference index or profile summary.

For unresolved work, begin with the open-loop index or relevant project current-state document.

Do not begin by reading every memory in the repository.

After orientation, search the generated metadata index using the user's terminology plus relevant aliases, topics, tags, projects, entities, and memory types.

Retrieve the smallest set of atomic memories that can answer the question.

The initial retrieval set SHOULD normally contain only a few highly relevant memories. Expand retrieval only when the first set leaves a material gap, unresolved dependency, ambiguity, or conflict.

Follow relationship edges only when they are relevant to the question.

If a retrieved memory is `superseded`, identify the active memory that supersedes it before using it to answer a current-state question.

If a retrieved memory is `withdrawn`, do not use it as evidence except when explaining repository history or the reason it was withdrawn.

When a memory has relevant dependencies, corrections, supersession, or conflicts, retrieve the connected memories necessary to interpret it correctly.

Do not continue expanding retrieval once the answer is adequately supported.

---

# 6. Conflict resolution during retrieval

When two memories appear to conflict, do not guess and do not average them together.

Inspect:

1. lifecycle state;
2. relationship edges;
3. effective dates;
4. provenance;
5. confidence;
6. relevant correction or supersession memories;
7. the authoritative live source when the claim concerns current external or project state.

A newer timestamp alone does not prove that a memory supersedes another memory.

A higher confidence value alone does not prove that a memory is newer or canonical.

If the repository does not contain enough information to resolve the conflict, explicitly state the uncertainty.

---

# 7. Preserve epistemic status

Every retrieved claim has an epistemic basis.

Preserve that basis when reasoning and answering.

`user_stated` means the information was explicitly provided by the user.

`project_verified` means the claim was verified against an authoritative project source.

`external_verified` means it was verified against an appropriate external source.

`derived` means it was deterministically concluded from identified evidence.

`inferred` means it is a model inference rather than an explicitly established fact.

`migrated` means the information was imported from legacy context whose original provenance may be incomplete.

Do not silently upgrade `inferred`, `migrated`, or uncertain information into verified fact.

Confidence and provenance basis are separate concepts.

---

# 8. Verify current implementation facts

The memory repository is not a mirror of project source trees.

When an answer depends materially on current source code, dependency versions, configuration, tests, branches, implementation behavior, or other facts controlled by an actual project repository, inspect the authoritative project repository when access is available.

A memory may accurately explain why code was written while being stale about what the current code now contains.

If verification is required but repository access is unavailable, state that limitation.

Do not pretend that a historical memory constitutes current code verification.

---

# 9. Memory admission test

Do not create a memory simply because something was discussed.

Before writing, ask:

> Would knowing this later materially improve a future conversation?

Persistent memory is normally appropriate for durable project state, decisions, rationale, standing preferences, significant milestones, useful workflows, technical discoveries with lasting value, open loops, corrections, important historical context, and explicit requests to remember something.

Transient debugging output, repetitive remarks, temporary failures that no longer matter, conversational filler, and intermediate thoughts without durable value SHOULD NOT become memories.

An explicit user request to remember information is presumptively eligible for storage unless storage would violate the security rules or the user explicitly chooses another destination.

---

# 10. Search before creating

Before creating a new atomic memory, search for an existing memory representing the same underlying fact, preference, decision, state, open loop, correction, milestone, or reference.

Use title terms, aliases, project, topic, entities, tags, and semantically related terminology.

Do not create a new memory merely because the new wording is different.

If an existing memory represents the same underlying concept, determine whether it should be updated, superseded, corrected, or left unchanged.

Creating near-duplicate memories is a repository-integrity failure.

---

# 11. Decide between update and new memory

Update an existing atomic memory when the underlying historical meaning has not changed and the modification only improves representation or metadata.

Examples include:

- improving wording;
- adding a source;
- adding useful aliases;
- correcting formatting;
- clarifying an explanation without changing its substantive historical meaning;
- improving retrieval metadata.

Create a new atomic memory when a meaningful historical event has occurred.

Examples include:

- a new decision;
- a meaningful reversal;
- a significant new project state;
- a newly opened unresolved task;
- correction of a previous belief;
- a major milestone;
- a changed preference where preserving the earlier state matters.

Do not rewrite an old memory to make it appear that the new state was always true.

---

# 12. Creating an atomic memory

1. Generate a new UUIDv4.
2. The UUID is permanent and MUST NOT encode project, type, date, or other mutable meaning.
3. Choose a concise semantic filename slug followed by the first eight hexadecimal characters of the UUID.
4. Create both the Markdown content file and JSON sidecar.
5. The pair MUST use the same basename.
6. The JSON `content_path` MUST identify the Markdown file.
7. The Markdown MUST contain the full UUID near its beginning.
8. Populate retrieval aliases using realistic terminology that a future user or assistant might use.
9. Do not stuff arbitrary synonyms, search aliases, tags, or machine keywords into prose solely for retrieval.
10. Record provenance and uncertainty accurately.
11. Assign sensitivity conservatively.
12. Create only relationships justified by actual knowledge.
13. Do not invent relationship targets, memory IDs, source locations, dates, commits, or citations.

---

# 13. Relationship rules

Relationships are stored as canonical graph edges from the current memory to target memory IDs.

Supported V1 relationship types are:

- `related_to`
- `depends_on`
- `supersedes`
- `corrects`
- `conflicts_with`

Do not store redundant inverse fields such as both `supersedes` and `superseded_by`.

Inverse relationships SHOULD be derived by indexes or tooling.

When memory B supersedes memory A, memory B stores a `supersedes` relationship targeting A and A's lifecycle becomes `superseded`.

When B corrects an incorrect or materially incomplete claim in A, B stores a `corrects` relationship targeting A.

If the corrected understanding fully replaces A as canonical truth, B SHOULD also supersede A.

A normal state transition is not automatically a correction.

---

# 14. Corrections and supersession

Never silently destroy meaningful historical context.

If A was previously believed true and B later establishes that A was wrong, create or identify the appropriate correction memory and link B to A.

If A was true at the time but B represents a later state, use supersession rather than describing A as erroneous.

Preserve A unless security, privacy, corruption, or explicit deletion requirements justify removal.

Retrieval for current-state questions SHOULD naturally favor the active replacement.

Historical questions MAY intentionally retrieve superseded memories.

---

# 15. Current-state documents

Current-state documents exist to bootstrap a future conversation quickly.

They SHOULD remain concise.

They SHOULD contain an explicit `Last reviewed` date.

When relevant to source-code state, they SHOULD identify when the authoritative project repository was last verified and MAY record the verified branch or commit.

Important current-state claims SHOULD reference the atomic memory IDs that justify them when such memories exist.

Current-state documents MUST NOT become the only location where important durable information exists.

When a new memory materially changes present project state, update the appropriate current-state document.

---

# 16. Memory granularity

Memories SHOULD be atomic enough to retrieve independently but large enough to remain semantically meaningful.

“Atomic” does not mean “one sentence.”

Do not split one coherent decision into many tiny memories merely because it contains several details.

Do not combine unrelated facts, decisions, states, and history into giant catch-all memories.

Prefer one memory for one cohesive concept whose content is useful when retrieved independently.

---

# 17. Inferred memories

Treat inferred memories conservatively.

Do not create an inferred memory merely because a model noticed a pattern.

An inference is eligible for persistent storage only when it has clear future value and its inferred status is materially useful.

The provenance MUST remain `inferred`.

Its evidence SHOULD be identified.

Its confidence MUST reflect actual uncertainty.

Do not infer sensitive personal attributes for persistent storage.

When confirmation from the user would materially improve reliability, prefer confirmation over converting an inference into a durable canonical fact.

---

# 18. Security and privacy

This repository is private but is not a secrets vault.

Never store:

- passwords;
- authentication tokens;
- API secrets;
- private keys;
- recovery codes;
- session credentials;
- banking credentials;
- other authentication secrets.

Do not store a forbidden secret and merely mark it `sensitive`.

Information classified as “never store” must never be committed.

Remember that deleting a file from the current Git tree does not necessarily remove it from Git history.

If a secret is accidentally committed, treat it as a security incident requiring credential rotation where applicable and Git-history remediation.

Be conservative with sensitive personal information.

---

# 19. Generated indexes

Machine-readable indexes SHOULD be generated from the JSON sidecars.

Generated indexes MUST be reconstructable.

Deleting generated indexes MUST NOT destroy unique knowledge.

Human-readable indexes are navigation aids.

Do not manually introduce facts into an index that do not exist in authoritative memory content or metadata.

After a write that affects indexed data, regenerate affected indexes when tooling is available.

---

# 20. Context-window discipline

This repository may eventually contain thousands of memories.

Do not attempt to load the entire repository into an LLM context window.

Use summaries and indexes to narrow retrieval before reading atomic memories.

Prefer metadata filtering before prose retrieval.

Prefer a small evidence set whose relevance is understood over a large context dump whose relevance is uncertain.

When additional context is needed, expand incrementally.

Do not retrieve historical chains merely because they exist. Retrieve history when the user's question or a conflict requires it.

---

# 21. Stale-write and multi-agent protection

Assume another conversation, user action, or agent may have modified the repository since the current session last read it.

Before modifying an existing memory, current-state document, or generated index, re-read the latest relevant version when repository tooling permits.

Do not overwrite a newer change using stale content.

Keep memory changes small and reviewable.

Avoid combining unrelated memory updates into one large modification.

---

# 22. Validation before claiming completion

A memory write is not complete merely because text was generated.

Before claiming that a repository update succeeded, verify as many applicable invariants as tooling permits.

At minimum:

1. The Markdown and JSON pair both exist.
2. The JSON parses.
3. The JSON satisfies the current schema.
4. The UUID is unique.
5. The Markdown filename UUID suffix matches the sidecar UUID.
6. The `content_path` resolves to the paired Markdown file.
7. Relationship target IDs exist.
8. Duplicate logical relationships are not present.
9. Lifecycle and supersession are consistent.
10. Supersession contains no cycles.
11. Conditional fields such as `open_loop_status` obey their type rules.
12. Temporal ordering is valid.
13. Generated indexes are rebuilt when applicable.
14. Relevant current-state documents are synchronized when affected.
15. Secrets have not been introduced.

Repository-wide invariants are specified in `docs/REPOSITORY_VALIDATION.md`.

If validation cannot be performed, state what was not validated.

Never claim success based only on intended output.

---

# 23. Failure behavior

If repository information is incomplete, say so.

If sources conflict and cannot be resolved, say so.

If a source cannot be accessed, do not pretend it was checked.

If an expected memory does not exist, do not invent it.

If a relationship target cannot be found, do not manufacture a replacement ID.

If a current-state summary appears stale, treat it as stale until verified.

If the correct memory action is uncertain, prefer preserving existing history and making the smallest safe change.

Do not “clean up” historical information merely because it looks inconsistent with the present.

---

# 24. Retrieval stop condition

Stop retrieving when:

- the smallest reasonably sufficient set of evidence has been obtained to answer the user's actual question;
- relevant conflicts have been resolved or explicitly identified as unresolved; and
- any required authoritative current-state verification has been performed.

**More context is not automatically better context.**

---

# 25. Write stop condition

A memory operation is complete only when:

- the intended durable information is represented without unnecessary duplication;
- its provenance and epistemic status are preserved;
- relevant relationships and lifecycle changes are correct;
- affected current-state views are synchronized where necessary; and
- available validation passes.

Do not continue generating additional memories merely to make the repository appear more comprehensive.

**Repository quality takes priority over repository size.**
