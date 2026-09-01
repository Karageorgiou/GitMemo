# Atomic Memory Markdown Format — V1

## Purpose

This document defines how an AI assistant or other LLM should write the human-readable Markdown half of an atomic memory.

The paired JSON sidecar is authoritative for identity, lifecycle, provenance, retrieval metadata, machine relationships, timestamps, sensitivity, and other structured management fields.

The Markdown file is authoritative for the memory's natural-language meaning, context, reasoning, and useful historical explanation.

Do not duplicate the full JSON sidecar in Markdown.

---

# 1. General authoring rules for LLMs

## 1.1 Preserve meaning, not conversation wording

Write a durable memory, not a transcript.

Summarize the information in language that will still make sense when retrieved months or years later.

Do not preserve conversational filler, jokes, repeated explanations, temporary debugging noise, or irrelevant chronology.

## 1.2 Do not invent missing information

A template heading is not a request to fabricate content.

If a section is optional and the source material does not support useful content for it, omit the section.

If a required semantic field is unknown, say that it is unknown only when the uncertainty itself matters. Otherwise, do not create the memory until the required meaning can be represented accurately.

## 1.3 Keep epistemic status intact

Do not rewrite inference as fact.

Do not strengthen tentative wording merely to make the memory sound cleaner.

If the memory is based on an inference, uncertainty or limitation that is important to interpretation should be visible in the prose.

The JSON sidecar still carries the canonical provenance basis and confidence.

## 1.4 Preserve historical perspective

Describe what was known, decided, corrected, or true at the relevant time.

Do not rewrite an old decision as though later knowledge was available when the decision was originally made.

When later information changes the canonical understanding, preserve the earlier memory and use correction or supersession rather than retroactively rewriting history.

## 1.5 Prefer cohesive atomicity

One Markdown memory should represent one independently retrievable concept.

It may contain several paragraphs and supporting details when they all explain the same concept.

Do not create one memory per sentence.

Do not create giant catch-all memories spanning unrelated concepts.

## 1.6 Write for future retrieval

Assume the future reader has not seen the original conversation.

Provide enough context for the memory to stand alone.

Name the relevant project, system, decision, artifact, person, or technology naturally when doing so improves comprehension.

Do not stuff synonyms, search aliases, tags, or machine keywords into prose solely for retrieval. Those belong in the JSON sidecar.

## 1.7 Avoid unnecessary duplication with JSON

Every Markdown memory MUST contain the canonical title as the H1 heading, the full memory UUID, and the memory type.

The full UUID is repeated intentionally so that the Markdown remains identifiable if separated from its sidecar.

Other structured metadata SHOULD remain in the JSON unless it is naturally useful to understanding the memory.

## 1.8 Remove template scaffolding before finalizing

The files in `templates/` are authoring scaffolds, not valid completed memories.

Before writing a finalized memory, remove all placeholder text and instructional HTML comments.

A finalized memory MUST NOT contain unresolved template markers such as `<Canonical title>`, `<full UUID>`, placeholder instructions enclosed in angle brackets, or instructional template comments.

Do not preserve an empty optional section merely because it existed in the template.

---

# 2. Common header

Every atomic memory begins with:

```markdown
# <Canonical title>

**Memory ID:** `<full UUID>`  
**Type:** `<memory type>`
```

Do not add YAML frontmatter in V1.

Do not copy tags, aliases, lifecycle, timestamps, sensitivity, or relationship arrays into the Markdown header.

---

# 3. Type-specific semantics

## FACT
Required: `## Fact`, `## Context`.
Optional: `## Evidence`, `## Limitations`, `## Why this matters`.

## PREFERENCE
Required: `## Preference`, `## Application`.
Optional: `## Rationale`, `## Exceptions`, `## Examples`.
Do not broaden a preference beyond what the user actually established.

## DECISION
Required: `## Context`, `## Decision`, `## Rationale`.
Optional: `## Alternatives considered`, `## Consequences`, `## Constraints`, `## Revisit conditions`.
Do not invent rejected alternatives merely to fill the template.

## STATE
Required: `## State`, `## Significance`.
Optional: `## Relevant context`, `## Constraints`, `## Next transition`.
Use sparingly. If durable significance cannot be explained, reconsider whether an atomic STATE memory belongs at all.

## OPEN LOOP

`open_loop` is the memory type. It records an unresolved-work item and its history. It does **not** mean the item must remain unresolved forever.

The JSON `open_loop_status` determines which Markdown form is valid.

Unresolved statuses are `open`, `blocked`, and `deferred`.

For unresolved statuses, required sections are:

- `## Open question or task`
- `## Why it remains open`
- `## Next useful action`

Optional unresolved sections are `## Context`, `## Blockers`, `## Constraints`, and `## Resolution criteria`.

Terminal statuses are `resolved` and `cancelled`.

For terminal statuses, required sections are:

- `## Original question or task`
- `## Outcome`
- `## Closure basis`

Optional terminal sections are `## Prior unresolved context` and `## Follow-up`.

When an open loop becomes terminal, rewrite future-directed unresolved headings into the terminal form. Do not retain headings such as `## Why it remains open` merely to preserve history. Preserve the useful historical reason under `## Prior unresolved context` instead.

A terminal open-loop memory normally remains `lifecycle: active` because the memory itself remains a valid historical record. `lifecycle` describes memory validity; `open_loop_status` describes task status.

Resolving or cancelling an open loop does not automatically require a new atomic memory. Create a separate milestone, decision, correction, or other memory only when the outcome has independent durable value.

Do not automatically set `effective_until` merely because an open loop closes. Use temporal boundaries only when they correctly describe the substantive applicability of the memory.

## CORRECTION
Required: `## Previous understanding`, `## Corrected understanding`, `## Basis for correction`, `## Impact`.
Optional: `## Remaining uncertainty`.
Do not use CORRECTION for a normal state transition where the previous state was valid at the time.

## MILESTONE
Required: `## Milestone`, `## Result`, `## Why it matters`.
Optional: `## What changed`, `## Follow-up`.
Do not store every completed task as a milestone.

## REFERENCE
Required: `## Reference`, `## When to use this`.
Optional: `## Procedure`, `## Details`, `## Caveats`, `## Examples`.
Do not turn GitMemo into a duplicate encyclopedia or source-code mirror.

---

# 4. Length guidance

There is no hard word limit.

Use the minimum amount of prose required to preserve durable meaning and reasoning.

Do not pad a memory to satisfy a perceived template length, and do not aggressively compress rationale that is important to future interpretation.

---

# 5. Final LLM self-check before writing

Before finalizing the Markdown, verify:

1. Could a future LLM understand this without the original conversation?
2. Is this one cohesive memory?
3. Did I preserve the historical perspective?
4. Did I accidentally turn inference into fact?
5. Did I invent content to fill an optional section?
6. Is any paragraph merely transient noise?
7. Am I duplicating metadata that belongs only in JSON?
8. Does the title accurately describe the memory?
9. Does the UUID exactly match the paired JSON sidecar?
10. Does the type exactly match the paired JSON sidecar?
11. Did I remove every placeholder and instructional template comment?
12. If this is an `open_loop`, does its Markdown form match `open_loop_status`?

If any answer reveals a problem, fix it before writing the memory.
