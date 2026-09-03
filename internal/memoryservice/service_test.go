package memoryservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/memory"
	"github.com/runethread/core/internal/repository"
	"github.com/runethread/core/internal/starter"
)

const testMemoryID = "11111111-1111-4111-8111-111111111111"

type fakeRepository struct {
	root  string
	state repository.State
	err   error
}

func (r fakeRepository) Root() string { return r.root }
func (r fakeRepository) State(context.Context) (repository.State, error) {
	return r.state, r.err
}

func TestGetFallsBackToCanonicalMemoryWhenIndexUnavailable(t *testing.T) {
	root := makeServiceFixture(t)
	if err := os.Remove(filepath.Join(root, "index", "catalog.json")); err != nil {
		t.Fatal(err)
	}
	svc := New(fakeRepository{root: root, state: repository.State{Revision: "abc", Branch: "main", Clean: true}})

	got, err := svc.Get(context.Background(), testMemoryID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Document.Memory.ID != testMemoryID {
		t.Fatalf("memory ID = %q", got.Document.Memory.ID)
	}
	if got.Document.Markdown == "" {
		t.Fatal("expected canonical Markdown")
	}
}

func TestPrepareMutationIsReadOnlyRevisionBoundAndReturnsLegalOperations(t *testing.T) {
	root := makeServiceFixture(t)
	repo := fakeRepository{root: root, state: repository.State{Revision: "expected-revision", Branch: "main", Clean: true}}
	svc := New(repo)

	before := snapshotFiles(t, root)
	got, err := svc.PrepareMutation(context.Background(), PrepareMutationRequest{Query: "alpha", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	after := snapshotFiles(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("PrepareMutation modified repository files")
	}
	if got.ExpectedRevision != "expected-revision" {
		t.Fatalf("expected revision = %q", got.ExpectedRevision)
	}
	if len(got.Candidates) != 1 || got.Candidates[0].Memory.ID != testMemoryID {
		t.Fatalf("unexpected candidates: %+v", got.Candidates)
	}
	wantOperations := []string{"correct", "create", "noop", "supersede", "update", "withdraw"}
	if !reflect.DeepEqual(got.LegalOperations, wantOperations) {
		t.Fatalf("legal operations = %v, want %v", got.LegalOperations, wantOperations)
	}
}

func TestPrepareMutationRejectsDirtyRepository(t *testing.T) {
	root := makeServiceFixture(t)
	svc := New(fakeRepository{
		root:  root,
		state: repository.State{Revision: "abc", Branch: "main", Clean: false, DirtyEntries: []string{" M README.md"}},
	})

	_, err := svc.PrepareMutation(context.Background(), PrepareMutationRequest{})
	var serviceErr *Error
	if !errors.As(err, &serviceErr) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if serviceErr.Code != CodeRepositoryDirty {
		t.Fatalf("error code = %q, want %q", serviceErr.Code, CodeRepositoryDirty)
	}
}

func TestStatusReportsNativeRepositoryHealth(t *testing.T) {
	root := makeServiceFixture(t)
	svc := New(fakeRepository{root: root, state: repository.State{Revision: "abc", Branch: "main", Clean: true}})

	got, err := svc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.TrustOK || got.TrustProblems != 0 {
		t.Fatalf("unexpected trust status: %+v", got)
	}
	if got.ValidationErrors != 0 || got.ValidationWarnings != 0 {
		t.Fatalf("unexpected validation status: %+v", got)
	}
	if !got.IndexCurrent {
		t.Fatalf("expected current index: %+v", got)
	}
	if got.Revision != "abc" || got.Branch != "main" || !got.Clean {
		t.Fatalf("unexpected repository state: %+v", got)
	}
}

func TestSearchRejectsInvalidLimit(t *testing.T) {
	root := makeServiceFixture(t)
	svc := New(fakeRepository{root: root, state: repository.State{Revision: "abc", Clean: true}})

	_, err := svc.Search(context.Background(), SearchRequest{Query: "alpha", Limit: 101})
	var serviceErr *Error
	if !errors.As(err, &serviceErr) || serviceErr.Code != CodeInvalidArgument {
		t.Fatalf("error = %v, want invalid_argument", err)
	}
}

func makeServiceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, "memories", "projects", "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	markdownRel := "memories/projects/alpha/choose-alpha-mode--11111111.md"
	sidecarRel := "memories/projects/alpha/choose-alpha-mode--11111111.json"
	m := memory.Memory{
		SchemaVersion: 1,
		ID:            testMemoryID,
		Title:         "Choose alpha mode",
		Type:          "decision",
		Lifecycle:     "active",
		Summary:       "Alpha mode is the chosen operating mode.",
		Projects:      []string{"alpha"},
		Topics:        []string{"architecture"},
		Tags:          []string{"alpha"},
		Aliases:       []string{"alpha mode"},
		Entities:      []memory.Entity{},
		Importance:    "normal",
		Temporal: memory.Temporal{
			CreatedAt: "2026-09-03T00:00:00Z",
			UpdatedAt: "2026-09-03T00:00:00Z",
		},
		Provenance: memory.Provenance{
			Basis:                 "user_stated",
			Confidence:            "high",
			ExplicitMemoryRequest: true,
			Sources: []memory.Source{{
				Kind:    "conversation",
				Locator: "test-fixture",
			}},
		},
		Relationships: []memory.Relationship{},
		ContentPath:   markdownRel,
		Sensitivity:   "routine",
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(sidecarRel)), data, 0o644); err != nil {
		t.Fatal(err)
	}
	markdown := "# Choose alpha mode\n\n**Memory ID:** `" + testMemoryID + "`  \n**Type:** `decision`\n\n## Context\n\nAlpha mode was evaluated.\n\n## Decision\n\nUse alpha mode.\n\n## Rationale\n\nIt is deterministic for this fixture.\n"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(markdownRel)), []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := indexer.Write(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func snapshotFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
