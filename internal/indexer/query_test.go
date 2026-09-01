package indexer

import (
	"strings"
	"testing"
)

func TestLookupByIDReadsDeterministicShard(t *testing.T) {
	root := setup(t)
	id := "11111111-1111-4111-8111-111111111111"
	writeMemoryText(t, root, id, "carbonara", "decision", "", "Carbonara recipe decision", "Keep this recipe simple.")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}

	result, ok, err := LookupByID(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || result.ID != id || result.Title != "Carbonara recipe decision" {
		t.Fatalf("unexpected exact lookup: ok=%v result=%#v", ok, result)
	}
}

func TestFilterIndexesReturnSortedIDs(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "22222222-2222-4222-8222-222222222222", "b", "decision", "")
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "fact", "")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}

	ids, err := IDsForProject(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || !strings.HasPrefix(ids[0], "11111111-") || !strings.HasPrefix(ids[1], "22222222-") {
		t.Fatalf("unexpected project IDs: %v", ids)
	}
	factIDs, err := IDsForType(root, "fact")
	if err != nil {
		t.Fatal(err)
	}
	if len(factIDs) != 1 || !strings.HasPrefix(factIDs[0], "11111111-") {
		t.Fatalf("unexpected type IDs: %v", factIDs)
	}
}

func TestSearchRanksTitleAboveSummaryOnlyMatch(t *testing.T) {
	root := setup(t)
	writeMemoryText(t, root, "11111111-1111-4111-8111-111111111111", "primary", "decision", "", "Carbonara recipe decision", "Primary cooking decision.")
	writeMemoryText(t, root, "22222222-2222-4222-8222-222222222222", "secondary", "fact", "", "Pasta note", "Carbonara appears only in this summary.")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}

	results, err := Search(root, "carbonara", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two matches, got %#v", results)
	}
	if !strings.HasPrefix(results[0].ID, "11111111-") || results[0].Score <= results[1].Score {
		t.Fatalf("title match should rank first: %#v", results)
	}
}

func TestSearchRewardsMatchingMoreQueryTerms(t *testing.T) {
	root := setup(t)
	writeMemoryText(t, root, "11111111-1111-4111-8111-111111111111", "both", "fact", "", "Pepper museum architecture", "Robot project notes.")
	writeMemoryText(t, root, "22222222-2222-4222-8222-222222222222", "one", "fact", "", "Pepper notes", "Unrelated summary.")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}

	results, err := Search(root, "pepper museum", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || len(results[0].MatchedTerms) != 2 || !strings.HasPrefix(results[0].ID, "11111111-") {
		t.Fatalf("multi-term match should rank first: %#v", results)
	}
}

func TestSearchExactUUID(t *testing.T) {
	root := setup(t)
	id := "11111111-1111-4111-8111-111111111111"
	writeMemory(t, root, id, "a", "decision", "")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}
	results, err := Search(root, id, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != id {
		t.Fatalf("unexpected UUID search results: %#v", results)
	}
}

func TestSearchRefusesExplicitlyStaleIndex(t *testing.T) {
	root := setup(t)
	writeMemory(t, root, "11111111-1111-4111-8111-111111111111", "a", "decision", "")
	if err := Write(root); err != nil {
		t.Fatal(err)
	}
	if err := MarkStale(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Search(root, "memory", 10); err == nil || !strings.Contains(err.Error(), "marked stale") {
		t.Fatalf("expected stale-index refusal, got %v", err)
	}
}
