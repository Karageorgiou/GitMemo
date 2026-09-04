# Runethread agent instructions

This file applies to automated coding agents and AI assistants working in `runethread/core`.

Before making a substantive change, read [`docs/runethread/ENGINEERING_PROCESS.md`](docs/runethread/ENGINEERING_PROCESS.md) and [`docs/runethread/CURRENT_MILESTONE.md`](docs/runethread/CURRENT_MILESTONE.md). Also read the relevant accepted ADRs and architecture/roadmap documents for the area being changed. The engineering-process document governs how changes are made; the current-milestone document identifies the immediate verified work boundary.

## Non-negotiable working rules

1. **Verify state before planning from it.** Re-read `main`, the intended branch/base SHA, relevant release/version constants, repository rules, and the files that actually implement the behavior. Do not rely on a remembered SHA or prior-chat summary when a live repository read can verify it.
2. **Classify impact before editing.** Determine whether the change affects runtime only, public API/CLI, dependencies/toolchain, repository metadata, schema, contract semantics/bytes, indexes, trust, bootstrap, migrations, release automation, templates, or downstream memory repositories.
3. **Treat semantic contract changes as contract changes even when a vendored file was not initially edited.** Implementation and the immutable operational contract must never disagree.
4. **Use exact historical fixtures for historical compatibility.** Do not synthesize an old released repository with the current generator when released control-plane bytes or semantics differ.
5. **Make small deliberate writes.** No blind global replacement, unrelated refactor, speculative compatibility shim, or force-push. One logical writer at a time.
6. **Validation is read-only.** CI/test workflows MUST NOT modify source and push fixes back to the branch they validate. Make a deliberate commit first, then validate that exact committed SHA.
7. **Fail closed on discrepancies.** If verified state contradicts the plan, a tool writes unexpectedly, a workflow races, required evidence is incomplete, or an invariant cannot be proven, stop further writes and investigate before continuing.
8. **Use draft PRs as review boundaries.** Review the canonical GitHub PR patch, checks, comments/reviews, base/head SHAs, and mergeability before readiness. Merge only an exact verified head SHA.
9. **Verify after merge and release.** Re-read `main`, the merge commit, tag/release assets, generated template, and affected downstream repositories. Do not infer success from the merge/release action alone.
10. **Preserve canonical ownership.** Project engineering rules and source truth live in this repository. Personal semantic memory may reference project decisions but must not become the only source of project policy.

## Known agent/tool failure modes

- GitHub/code-search indexes can be incomplete or stale. Exhaustive audits require repository-tree/file enumeration or a verified local checkout when available.
- Tool responses can be truncated. Do not infer unseen content; issue narrower reads.
- Asynchronous Actions runs can finish after later branch edits. Never allow validation workflows to write commits, and never assume the most recently inspected run is the only active writer.
- A green test suite proves only the assertions it contains. Perform backward, forward, negative, and cross-surface checks described in the engineering process.
- Current generators are not historical fixtures. Historical release compatibility must be tested against exact released state.
- GitHub legacy branch-protection endpoints and repository rulesets are separate surfaces. Verify effective policy through the active ruleset when one exists.

When uncertainty remains, report it explicitly instead of converting it into an implementation assumption.
