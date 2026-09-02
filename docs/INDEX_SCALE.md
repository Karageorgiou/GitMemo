# Index v2 scale envelope

This document records the synthetic scale test used before GitMemo v0.5.0. It is evidence about one benchmark environment, not a universal latency guarantee.

## Benchmark design

The gated Go test `TestSyntheticScale` constructs deterministic in-memory `memory.Record` values and exercises the same machine-index renderer and query implementation used by GitMemo. It intentionally uses sequential visible UUID prefixes so UUID shard balance depends on `SHA-256(lowercase full UUID)`, not on favorable UUID input distribution.

The corpus uses:

- 100 projects;
- 1,000 topics;
- one shared tag;
- common corpus-wide words;
- one unique searchable key per memory;
- deterministic valid UUIDv4-shaped IDs.

The test asserts that:

- every generated file remains at most 2 MiB;
- exact UUID lookup succeeds;
- a unique natural-language term resolves the expected memory;
- a direct project lookup returns the complete matching set;
- corpus-wide terms above the posting threshold are represented as high-frequency/suppressed instead of becoming unbounded posting files.

The test is gated by `GITMEMO_SCALE_N` and is skipped during normal CI unless explicitly requested.

## Measured GitHub-hosted-runner results

The v0.5.0 pre-release scale run used Ubuntu 24.04 on a GitHub-hosted runner with Go 1.27.0.

| Memories | Generated files | Generated index bytes | Largest file | Build | Write | Exact ID | Unique term | Project lookup | Common term |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 10,000 | 8,739 | 14,242,481 | 58,383 B | 181.9 ms | 355.0 ms | 131 µs | 80 µs | 51 µs | 11.75 ms |
| 100,000 | 9,590 | 91,097,185 | 114,385 B | 1.532 s | 400.6 ms | 219 µs | 191 µs | 189 µs | 72 µs |
| 1,000,000 | 14,227 | 910,155,671 | 195,941 B | 15.696 s | 904.4 ms | 961 µs | 1.317 ms | 1.341 ms | 307 µs |

At one million records, Go reported approximately 3.34 GB of system memory (`runtime.MemStats.Sys`) after the test's full build/write/query workload. The generated index was about 910 MB for this intentionally term-heavy synthetic corpus.

## Interpretation

The benchmark supports the v0.5.0 design goal that targeted reads remain driven by query terms, candidate postings, or matching category size rather than by the total number of memories. It also demonstrates that SHA-256 UUID sharding keeps adversarial sequential UUID inputs distributed across the 4,096 possible ID shards.

The benchmark does **not** prove that every million-memory repository will have these exact timings or sizes. Real performance varies with metadata length, term vocabulary, disk, filesystem, Git provider, network latency, hardware, and the number of matches returned.

The current full deterministic rebuild is still O(N) in memory count and constructs the generated representation in memory before atomically replacing `index/`. At one million synthetic records this completed comfortably on the hosted runner, but its multi-gigabyte memory footprint is a clear future optimization target. Incremental or streaming generation may reduce rebuild cost later without changing canonical memory data.

Normal search does not require a full rebuild. Once a current index exists:

- exact UUID lookup hashes the UUID and reads one deterministic ID shard;
- a natural-language query reads only the term descriptor shard(s), necessary posting chunks, and the ID shards for selected results;
- direct taxonomy/status queries read the relevant descriptor and only as many 1,024-ID chunks as required to return the matches.

## Reproducing

Run one size explicitly:

```bash
GITMEMO_SCALE_N=100000 go test ./internal/indexer -run '^TestSyntheticScale$' -count=1 -v
```

The test permits values from 1 through 1,000,000.
