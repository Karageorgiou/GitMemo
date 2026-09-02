package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChunkedIDListRoundTrip(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{}
	ids := make([]string, idListChunkSize+1)
	for i := range ids {
		ids[i] = fmt.Sprintf("%08x-0000-4000-8000-%012x", i, i)
	}
	if err := renderChunkedIDList(files, "index/by-project", "large-project", ids); err != nil {
		t.Fatal(err)
	}
	var descriptor idList
	if err := json.Unmarshal(files["index/by-project/large-project.json"], &descriptor); err != nil {
		t.Fatal(err)
	}
	if descriptor.Count != len(ids) || descriptor.ChunkSize != idListChunkSize || descriptor.ChunkCount != 2 || descriptor.IDs != nil {
		t.Fatalf("unexpected descriptor: %#v", descriptor)
	}
	writeGeneratedTestFiles(t, root, files)
	got, err := loadChunkedIDList(root, "by-project", "large-project", descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(ids) || got[0] != ids[0] || got[len(got)-1] != ids[len(ids)-1] {
		t.Fatalf("chunked ID round-trip mismatch: got %d IDs", len(got))
	}
}

func TestTermPostingsChunkAndRoundTrip(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{}
	postings := make([]termPosting, termPostingChunkSize+1)
	for i := range postings {
		postings[i] = termPosting{ID: fmt.Sprintf("%08x-0000-4000-8000-%012x", i, i), Score: 8}
	}
	value, err := renderTermIndexValue(files, "carbonara", postings)
	if err != nil {
		t.Fatal(err)
	}
	if value.DocumentFrequency != len(postings) || value.ChunkSize != termPostingChunkSize || value.ChunkCount != 2 || value.Postings != nil || value.Suppressed {
		t.Fatalf("unexpected term descriptor: %#v", value)
	}
	writeGeneratedTestFiles(t, root, files)
	got, err := loadTermPostings(root, "carbonara", value)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(postings) || got[0] != postings[0] || got[len(got)-1] != postings[len(postings)-1] {
		t.Fatalf("chunked posting round-trip mismatch: got %d postings", len(got))
	}
}

func TestHighFrequencyTermIsSuppressedNotTruncated(t *testing.T) {
	postings := make([]termPosting, maxStoredTermPostings+1)
	for i := range postings {
		postings[i] = termPosting{ID: fmt.Sprintf("%08x-0000-4000-8000-%012x", i, i), Score: 2}
	}
	files := map[string][]byte{}
	value, err := renderTermIndexValue(files, "common", postings)
	if err != nil {
		t.Fatal(err)
	}
	if !value.Suppressed || value.DocumentFrequency != len(postings) || value.Postings != nil || value.ChunkCount != 0 {
		t.Fatalf("high-frequency term was not represented as suppressed: %#v", value)
	}
	for path := range files {
		if strings.Contains(path, "term-postings") {
			t.Fatalf("suppressed term unexpectedly emitted posting chunk %s", path)
		}
	}
}

func TestSearchRequestsRefinementWhenOnlyMatchingTermIsSuppressed(t *testing.T) {
	root := t.TempDir()
	catalogData, err := json.Marshal(catalog{IndexVersion: IndexVersion})
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "index", "catalog.json"), string(catalogData)+"\n")
	bucket := termBucket("common")
	termData, err := json.Marshal(map[string]termIndexValue{
		"common": {DocumentFrequency: maxStoredTermPostings + 1, Suppressed: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "index", "terms", bucket+".json"), string(termData)+"\n")
	_, err = Search(root, "common", 10)
	if err == nil || !strings.Contains(err.Error(), "refine the query") {
		t.Fatalf("expected refinement error for suppressed-only query, got %v", err)
	}
}

func writeGeneratedTestFiles(t *testing.T, root string, files map[string][]byte) {
	t.Helper()
	for rel, data := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
