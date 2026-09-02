package indexer

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/runethread/core/internal/memory"
)

func TestSyntheticScale(t *testing.T) {
	raw := os.Getenv("RUNETHREAD_SCALE_N")
	if raw == "" {
		t.Skip("set RUNETHREAD_SCALE_N to run the synthetic scale test")
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || n > 1_000_000 {
		t.Fatalf("RUNETHREAD_SCALE_N must be an integer from 1 to 1000000, got %q", raw)
	}

	projects := make([]string, 100)
	for i := range projects {
		projects[i] = fmt.Sprintf("project-%03d", i)
	}
	topics := make([]string, 1000)
	for i := range topics {
		topics[i] = fmt.Sprintf("topic-%04d", i)
	}

	records := make([]memory.Record, n)
	for i := 0; i < n; i++ {
		id := syntheticScaleUUID(i)
		key := fmt.Sprintf("key%08d", i)
		records[i] = memory.Record{Memory: memory.Memory{
			ID:          id,
			Title:       "Synthetic " + key + " common",
			Type:        "fact",
			Lifecycle:   "active",
			Summary:     "Common benchmark corpus memory.",
			Projects:    []string{projects[i%len(projects)]},
			Topics:      []string{topics[i%len(topics)]},
			Tags:        []string{"benchmark"},
			Aliases:     []string{key},
			ContentPath: "memories/bench/entry--" + id[:8] + ".md",
		}}
	}

	buildStart := time.Now()
	files, err := renderMachineIndexesWithDigest(records, strings.Repeat("0", 64))
	if err != nil {
		t.Fatal(err)
	}
	buildDuration := time.Since(buildStart)

	var totalBytes int64
	largestBytes := 0
	largestPath := ""
	for path, data := range files {
		totalBytes += int64(len(data))
		if len(data) > largestBytes {
			largestBytes = len(data)
			largestPath = path
		}
	}
	const maxExpectedGeneratedFile = 2 * 1024 * 1024
	if largestBytes > maxExpectedGeneratedFile {
		t.Fatalf("largest generated file %s is %d bytes; expected every generated file <= %d bytes", largestPath, largestBytes, maxExpectedGeneratedFile)
	}

	root := t.TempDir()
	writeStart := time.Now()
	if err := replaceIndexDirectory(root, Result{Files: files}); err != nil {
		t.Fatal(err)
	}
	writeDuration := time.Since(writeStart)

	lastID := syntheticScaleUUID(n - 1)
	lookupStart := time.Now()
	got, ok, err := LookupByID(root, lastID)
	lookupDuration := time.Since(lookupStart)
	if err != nil || !ok || got.ID != lastID {
		t.Fatalf("exact lookup failed: ok=%v result=%#v err=%v", ok, got, err)
	}

	key := fmt.Sprintf("key%08d", n-1)
	searchStart := time.Now()
	results, err := Search(root, key, 10)
	searchDuration := time.Since(searchStart)
	if err != nil || len(results) == 0 || results[0].ID != lastID {
		t.Fatalf("unique term search failed: results=%#v err=%v", results, err)
	}

	projectStart := time.Now()
	projectIDs, err := IDsForProject(root, projects[0])
	projectDuration := time.Since(projectStart)
	if err != nil {
		t.Fatal(err)
	}
	expectedProjectCount := (n + len(projects) - 1) / len(projects)
	if len(projectIDs) != expectedProjectCount {
		t.Fatalf("project lookup returned %d IDs, expected %d", len(projectIDs), expectedProjectCount)
	}

	commonStart := time.Now()
	commonResults, commonErr := Search(root, "common", 10)
	commonDuration := time.Since(commonStart)
	if n > maxStoredTermPostings {
		if commonErr == nil || !strings.Contains(commonErr.Error(), "high-frequency") {
			t.Fatalf("expected high-frequency refinement error at n=%d, got results=%d err=%v", n, len(commonResults), commonErr)
		}
	} else if commonErr != nil || len(commonResults) == 0 {
		t.Fatalf("common term should remain searchable below suppression threshold: results=%d err=%v", len(commonResults), commonErr)
	}

	records = nil
	files = nil
	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	t.Logf("SCALE n=%d files=%d total_index_bytes=%d largest_file_bytes=%d largest_file=%s build=%s write=%s exact_lookup=%s unique_search=%s project_lookup=%s common_query=%s heap_after_gc=%d sys=%d",
		n, lenGeneratedFiles(root), totalBytes, largestBytes, largestPath, buildDuration, writeDuration, lookupDuration, searchDuration, projectDuration, commonDuration, mem.HeapAlloc, mem.Sys)
}

func syntheticScaleUUID(i int) string {
	return fmt.Sprintf("%08x-0000-4000-8000-%012x", i, i)
}

func lenGeneratedFiles(root string) int {
	paths, err := ExistingPaths(root)
	if err != nil {
		return -1
	}
	return len(paths)
}
