# ADR-002: Deterministic Core and semantic AI boundary

Status: **Accepted**
Date: 2026-09-02

## Context

Runethread needs AI judgment for semantic tasks such as deciding whether new information creates, corrects, supersedes, resolves, or duplicates existing memory. The same system also needs hard guarantees about repository integrity, trust locks, schema validity, relationships, concurrency, and rollback.

Putting both responsibilities inside model prompts would make correctness dependent on nondeterministic behavior and provider-specific capabilities.

## Decision

Runethread separates semantic judgment from repository invariants.

- AI models may interpret meaning, propose semantic operations, summarize, classify, and choose among legal operation types.
- Runethread Core deterministically enforces schema, trust, lifecycle constraints, relationship validity, repository layout, revision checks, index regeneration, rollback, and validation.
- AI models must not directly manipulate canonical storage formats when an equivalent deterministic Core operation exists.
- Runethread Core must not require an OpenAI, Anthropic, Google, or other model API to maintain repository integrity.
- Provider integrations remain adapters outside Core's invariant boundary.

## Consequences

- Core remains testable, reproducible, offline-capable, and provider-independent.
- Semantic behavior can improve without changing repository legality.
- A model can make a bad semantic proposal, but Core can still reject an illegal or stale mutation.
- Tool/API design must expose enough structured current state and legal operations for a model to make informed decisions.

## Alternatives considered

### Let the LLM edit Markdown/JSON directly
Rejected because correctness would depend on prompt adherence and every client would need to reimplement repository rules.

### Put a model inside Core for deduplication and lifecycle decisions
Rejected because repository validity would then depend on availability, pricing, and behavior of a specific model/provider.

### Make all semantic decisions deterministic
Rejected because meaning, contradiction, correction, and relevance cannot be fully captured by stable mechanical rules without unacceptable brittleness.

## Verification

1. Core unit/integration tests run without model credentials.
2. Invalid mutations are rejected regardless of caller/model.
3. No Core integrity path requires a remote AI API.
4. MCP/CLI/native adapters call the same deterministic application services.
5. Canonical writes are produced by Core operations, not arbitrary AI-authored file edits.
