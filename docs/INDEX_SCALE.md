# Index v2 scale envelope

This document records the synthetic scale test originally run before the v0.5.0 predecessor release and retained as evidence for Runethread Index v2. It is evidence about one benchmark environment, not a universal latency guarantee.

## Benchmark design

The gated Go test `TestSyntheticScale` constructs deterministic in-memory `memory.Record` values and exercises the same machine-index renderer/query implementation retained by Runethread. It intentionally uses sequential visible UUID prefixes so UUID shard balance depends on `SHA-256(lowercase full UUID)`, not favorable UUID input distribution.

The corpus uses 100 projects, 1,000 topics, one shared tag, common corpus-wide words, one unique searchable key per memory, and deterministic UUIDv4-shaped IDs.

The test asserts that every generated file stays at most 2 MiB, exact UUID lookup succeeds, a unique term resolves the expected memory, direct project lookup returns all matches, and corpus-wide terms above the posting threshold are suppressed rather than producing unbounded posting files.

The test is gated by `RUNETHREAD_SCALE_N` (the initial v0.5 benchmark used the predecessor environment variable name; the native runtime/test surface should use the Runethread name once the code-side gate is cut over).

## Measured predecessor-release runner results

The retained pre-release run used Ubuntu 24.04 on a GitHub-hosted runner with Go 1.27.0.

| Memories | Generated files | Generated index bytes | Largest file | Build | Write | Exact ID | Unique term | Project lookup | Common term |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 10,000 | 8,739 | 14,242,481 | 58,383 B | 181.9 ms | 355.0 ms | 131 µs | 80 µs | 51 µs | 11.75 ms |
| 100,000 | 9,590 | 91,097,185 | 114,385 B | 1.532 s | 400.6 ms | 219 µs | 191 µs | 189 µs | 72 µs |
| 1,000,000 | 14,227 | 910,155,671 | 195,941 B | 15.696 s | 904.4 ms | 961 µs | 1.317 ms | 1.341 ms | 307 µs |

At one million records, Go reported approximately 3.34 GB of system memory after the full build/write/query workload. The generated index was about 910 MB for this intentionally term-heavy synthetic corpus.

## Interpretation

The benchmark supports the design goal that targeted reads are driven by query terms, candidate postings, or matching category size rather than total memory count. SHA-256 UUID sharding also distributes adversarial sequential UUID inputs across 4,096 possible ID shards.

It does not guarantee these timings/sizes for every repository. Real performance varies with metadata length, term vocabulary, disk/filesystem, Git provider, network latency, hardware, and match count.

The current deterministic rebuild is O(N) and constructs generated representation in memory before atomically replacing `index/`. At one million synthetic records it completed on the hosted runner, but multi-gigabyte memory use is a future optimization target. Incremental or streaming generation may reduce rebuild cost without changing canonical data.

Normal search does not require a rebuild: exact UUID lookup reads one deterministic ID shard; ordinary language reads relevant term descriptor/posting shards plus selected ID shards; direct taxonomy/status queries read their descriptor and required chunks.

## Reproducing

After the code-side environment variable cutover, run:

```bash
RUNETHREAD_SCALE_N=100000 go test ./internal/indexer -run '^TestSyntheticScale$' -count=1 -v
```

The test permits values from 1 through 1,000,000.
