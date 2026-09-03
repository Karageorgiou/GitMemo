package memoryservice

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/memory"
	"github.com/runethread/core/internal/validation"
)

const (
	secondMemoryID = "22222222-2222-4222-8222-222222222222"
	thirdMemoryID  = "33333333-3333-4333-8333-333333333333"
	fourthMemoryID = "44444444-4444-4444-8444-444444444444"
	fifthMemoryID  = "55555555-5555-4555-8555-555555555555"
)

func TestApplyCreateIsIdempotentBeforeStaleCheck(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	request := ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "create-alpha-2",
		Operation:        "create",
		MutationTime:     "2026-09-03T01:00:00Z",
		Proposed:         decisionProposal(secondMemoryID, "Choose beta mode", nil),
	}

	first, err := svc.ApplyMutation(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "applied" || first.Commit == "" || first.Revision != first.Commit {
		t.Fatalf("first result = %+v", first)
	}
	stateAfterFirst, err := svc.repo.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	second, err := svc.ApplyMutation(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != "already_applied" || second.Commit != first.Commit {
		t.Fatalf("retry result = %+v, first = %+v", second, first)
	}
	stateAfterRetry, err := svc.repo.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stateAfterRetry.Revision != stateAfterFirst.Revision {
		t.Fatalf("retry advanced HEAD from %s to %s", stateAfterFirst.Revision, stateAfterRetry.Revision)
	}
	records, err := memory.LoadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("memory count = %d, want 2", len(records))
	}
	assertRepositoryHealthy(t, root)
}

func TestApplyRejectsIdempotencyKeyReuseWithDifferentRequest(t *testing.T) {
	svc, _, revision := makeGitServiceFixture(t)
	request := ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "same-key",
		Operation:        "create",
		MutationTime:     "2026-09-03T01:00:00Z",
		Proposed:         decisionProposal(secondMemoryID, "Choose beta mode", nil),
	}
	if _, err := svc.ApplyMutation(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	conflict := request
	conflict.Proposed = decisionProposal(secondMemoryID, "Choose beta mode differently", nil)
	_, err := svc.ApplyMutation(context.Background(), conflict)
	requireServiceCode(t, err, CodeIdempotencyConflict)
}

func TestApplyRejectsStalePreparationWithoutCanonicalWrite(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	first := ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "first-create",
		Operation:        "create",
		MutationTime:     "2026-09-03T01:00:00Z",
		Proposed:         decisionProposal(secondMemoryID, "Choose beta mode", nil),
	}
	if _, err := svc.ApplyMutation(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	stale := ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "stale-create",
		Operation:        "create",
		MutationTime:     "2026-09-03T01:01:00Z",
		Proposed:         decisionProposal(thirdMemoryID, "Choose gamma mode", nil),
	}
	_, err := svc.ApplyMutation(context.Background(), stale)
	requireServiceCode(t, err, CodeStaleRevision)
	if _, err := svc.Get(context.Background(), thirdMemoryID); err == nil {
		t.Fatal("stale apply unexpectedly created the proposed memory")
	} else {
		requireServiceCode(t, err, CodeNotFound)
	}
	assertRepositoryHealthy(t, root)
}

func TestFailedValidationLeavesCanonicalRepositoryUnchanged(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	before := snapshotFiles(t, root)
	proposal := decisionProposal(secondMemoryID, "Choose beta mode", nil)
	proposal.Markdown = "# Wrong title\n\n**Memory ID:** `" + secondMemoryID + "`  \n**Type:** `decision`\n\n## Context\n\nBad fixture.\n\n## Decision\n\nBad.\n\n## Rationale\n\nBad.\n"
	request := ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "invalid-create",
		Operation:        "create",
		MutationTime:     "2026-09-03T01:00:00Z",
		Proposed:         proposal,
	}
	_, err := svc.ApplyMutation(context.Background(), request)
	requireServiceCode(t, err, CodeValidationFailed)
	after := snapshotFiles(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed validation changed canonical repository files")
	}
	state, err := svc.repo.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Revision != revision || !state.Clean {
		t.Fatalf("repository state after failed validation = %+v", state)
	}
}

func TestSupersedeTransitionsTargetAndPreservesHistory(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	proposal := decisionProposal(secondMemoryID, "Choose beta mode", []memory.Relationship{{Type: "supersedes", TargetID: testMemoryID}})
	request := ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "supersede-alpha",
		Operation:        "supersede",
		MutationTime:     "2026-09-03T01:00:00Z",
		TargetID:         testMemoryID,
		Proposed:         proposal,
	}
	result, err := svc.ApplyMutation(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "applied" || len(result.ChangedMemoryIDs) != 2 {
		t.Fatalf("supersede result = %+v", result)
	}
	oldDoc, err := svc.Get(context.Background(), testMemoryID)
	if err != nil {
		t.Fatal(err)
	}
	if oldDoc.Document.Memory.Lifecycle != "superseded" {
		t.Fatalf("target lifecycle = %q", oldDoc.Document.Memory.Lifecycle)
	}
	newDoc, err := svc.Get(context.Background(), secondMemoryID)
	if err != nil {
		t.Fatal(err)
	}
	if newDoc.Document.Memory.Lifecycle != "active" || !hasRelationship(newDoc.Document.Memory, "supersedes", testMemoryID) {
		t.Fatalf("superseding memory = %+v", newDoc.Document.Memory)
	}
	assertRepositoryHealthy(t, root)
}

func TestCorrectionRequiresCorrectsEdgeAndLeavesTargetActive(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	proposal := correctionProposal(thirdMemoryID, testMemoryID, false)
	request := ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "correct-alpha",
		Operation:        "correct",
		MutationTime:     "2026-09-03T01:00:00Z",
		TargetID:         testMemoryID,
		Proposed:         proposal,
	}
	if _, err := svc.ApplyMutation(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	oldDoc, err := svc.Get(context.Background(), testMemoryID)
	if err != nil {
		t.Fatal(err)
	}
	if oldDoc.Document.Memory.Lifecycle != "active" {
		t.Fatalf("correction without supersession changed target lifecycle to %q", oldDoc.Document.Memory.Lifecycle)
	}
	newDoc, err := svc.Get(context.Background(), thirdMemoryID)
	if err != nil {
		t.Fatal(err)
	}
	if newDoc.Document.Memory.Type != "correction" || !hasRelationship(newDoc.Document.Memory, "corrects", testMemoryID) {
		t.Fatalf("correction memory = %+v", newDoc.Document.Memory)
	}
	assertRepositoryHealthy(t, root)
}

func TestResolveOpenLoopAndWithdraw(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	openProposal := openLoopProposal(fourthMemoryID, "Track alpha rollout", "open")
	created, err := svc.ApplyMutation(context.Background(), ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "create-open-loop",
		Operation:        "create",
		MutationTime:     "2026-09-03T01:00:00Z",
		Proposed:         openProposal,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolvedProposal := openLoopProposal(fourthMemoryID, "Track alpha rollout", "resolved")
	resolved, err := svc.ApplyMutation(context.Background(), ApplyMutationRequest{
		ExpectedRevision: created.Revision,
		IdempotencyKey:   "resolve-open-loop",
		Operation:        "resolve",
		MutationTime:     "2026-09-03T02:00:00Z",
		TargetID:         fourthMemoryID,
		Proposed:         resolvedProposal,
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := svc.Get(context.Background(), fourthMemoryID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Document.Memory.Lifecycle != "active" || doc.Document.Memory.OpenLoopStatus == nil || *doc.Document.Memory.OpenLoopStatus != "resolved" {
		t.Fatalf("resolved open loop = %+v", doc.Document.Memory)
	}

	withdrawn, err := svc.Withdraw(context.Background(), WithdrawRequest{
		ExpectedRevision: resolved.Revision,
		IdempotencyKey:   "withdraw-original",
		TargetID:         testMemoryID,
		MutationTime:     "2026-09-03T03:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if withdrawn.Operation != "withdraw" || withdrawn.Status != "applied" {
		t.Fatalf("withdraw result = %+v", withdrawn)
	}
	original, err := svc.Get(context.Background(), testMemoryID)
	if err != nil {
		t.Fatal(err)
	}
	if original.Document.Memory.Lifecycle != "withdrawn" {
		t.Fatalf("withdrawn lifecycle = %q", original.Document.Memory.Lifecycle)
	}
	assertRepositoryHealthy(t, root)
}

func TestNoopProducesNoCommit(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	before := snapshotFiles(t, root)
	result, err := svc.ApplyMutation(context.Background(), ApplyMutationRequest{
		ExpectedRevision: revision,
		IdempotencyKey:   "noop-1",
		Operation:        "noop",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "no_op" || result.Revision != revision || result.Commit != "" {
		t.Fatalf("noop result = %+v", result)
	}
	after := snapshotFiles(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("noop changed repository files")
	}
	state, err := svc.repo.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Revision != revision {
		t.Fatalf("noop advanced revision to %s", state.Revision)
	}
}

func makeGitServiceFixture(t *testing.T) (*Service, string, string) {
	t.Helper()
	root := makeServiceFixture(t)
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not available")
	}
	runMemoryGit(t, git, root, "init", "-b", "main")
	runMemoryGit(t, git, root, "config", "user.name", "Runethread Test")
	runMemoryGit(t, git, root, "config", "user.email", "runethread-test@example.invalid")
	runMemoryGit(t, git, root, "add", "-A")
	runMemoryGit(t, git, root, "commit", "-m", "initial memory fixture")
	svc, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	state, err := svc.repo.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !state.Clean || state.Branch != "main" {
		t.Fatalf("unexpected initial state: %+v", state)
	}
	return svc, root, state.Revision
}

func decisionProposal(id, title string, relationships []memory.Relationship) *ProposedDocument {
	m := baseProposedMemory(id, title, "decision")
	m.Summary = title + " is the durable decision."
	m.Relationships = append([]memory.Relationship(nil), relationships...)
	markdown := "# " + title + "\n\n**Memory ID:** `" + id + "`  \n**Type:** `decision`\n\n## Context\n\nThe operating mode was evaluated.\n\n## Decision\n\nUse the selected mode.\n\n## Rationale\n\nThe choice is durable and testable.\n"
	return &ProposedDocument{Memory: m, Markdown: markdown}
}

func correctionProposal(id, targetID string, supersedes bool) *ProposedDocument {
	m := baseProposedMemory(id, "Correct alpha mode understanding", "correction")
	m.Summary = "Correct the earlier alpha mode understanding."
	m.Relationships = []memory.Relationship{{Type: "corrects", TargetID: targetID}}
	if supersedes {
		m.Relationships = append(m.Relationships, memory.Relationship{Type: "supersedes", TargetID: targetID})
	}
	markdown := "# Correct alpha mode understanding\n\n**Memory ID:** `" + id + "`  \n**Type:** `correction`\n\n## Previous understanding\n\nThe earlier understanding was incomplete.\n\n## Corrected understanding\n\nThe corrected understanding is now explicit.\n\n## Basis for correction\n\nThe source was rechecked.\n\n## Impact\n\nFuture reasoning should use the corrected understanding.\n"
	return &ProposedDocument{Memory: m, Markdown: markdown}
}

func openLoopProposal(id, title, status string) *ProposedDocument {
	m := baseProposedMemory(id, title, "open_loop")
	m.Summary = "Track the alpha rollout until the outcome is known."
	m.OpenLoopStatus = stringPointer(status)
	var markdown string
	if status == "resolved" || status == "cancelled" {
		markdown = "# " + title + "\n\n**Memory ID:** `" + id + "`  \n**Type:** `open_loop`\n\n## Original question or task\n\nTrack the alpha rollout.\n\n## Outcome\n\nThe rollout is complete.\n\n## Closure basis\n\nThe result was verified.\n"
	} else {
		markdown = "# " + title + "\n\n**Memory ID:** `" + id + "`  \n**Type:** `open_loop`\n\n## Open question or task\n\nTrack the alpha rollout.\n\n## Why it remains open\n\nThe rollout is still in progress.\n\n## Next useful action\n\nVerify the final result.\n"
	}
	return &ProposedDocument{Memory: m, Markdown: markdown}
}

func baseProposedMemory(id, title, memoryType string) memory.Memory {
	return memory.Memory{
		SchemaVersion: 1,
		ID:            id,
		Title:         title,
		Type:          memoryType,
		Lifecycle:     "active",
		Summary:       title,
		Projects:      []string{"alpha"},
		Topics:        []string{"architecture"},
		Tags:          []string{"alpha"},
		Aliases:       []string{title},
		Entities:      []memory.Entity{},
		Importance:    "normal",
		Provenance: memory.Provenance{
			Basis:                 "user_stated",
			Confidence:            "high",
			ExplicitMemoryRequest: true,
			Sources: []memory.Source{{
				Kind:    "conversation",
				Locator: "mutation-test",
			}},
		},
		Relationships: []memory.Relationship{},
		Sensitivity:   "routine",
	}
}

func assertRepositoryHealthy(t *testing.T, root string) {
	t.Helper()
	issues := validation.Validate(root)
	if validation.HasErrors(issues) {
		t.Fatalf("repository validation failed: %s", validation.RenderText(issues))
	}
	stale, err := indexer.Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale index paths: %v", stale)
	}
}

func requireServiceCode(t *testing.T, err error, code string) {
	t.Helper()
	var serviceErr *Error
	if !errors.As(err, &serviceErr) {
		t.Fatalf("error = %v, want *Error code %s", err, code)
	}
	if serviceErr.Code != code {
		t.Fatalf("error code = %q, want %q: %v", serviceErr.Code, code, err)
	}
}

func stringPointer(value string) *string { return &value }

func runMemoryGit(t *testing.T, git, root string, args ...string) {
	t.Helper()
	cmd := exec.Command(git, append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestCanonicalPathsUseGeneralForUnscopedMemory(t *testing.T) {
	m := baseProposedMemory(fifthMemoryID, "Unscoped preference", "preference")
	m.Projects = []string{}
	markdown, sidecar, err := canonicalPaths(m)
	if err != nil {
		t.Fatal(err)
	}
	wantMarkdown := filepath.ToSlash("memories/general/unscoped-preference--55555555.md")
	wantSidecar := filepath.ToSlash("memories/general/unscoped-preference--55555555.json")
	if markdown != wantMarkdown || sidecar != wantSidecar {
		t.Fatalf("paths = %q, %q; want %q, %q", markdown, sidecar, wantMarkdown, wantSidecar)
	}
}

func TestConcurrentAppliesFromSameRevisionSerializeAndRejectStaleWriter(t *testing.T) {
	svc, root, revision := makeGitServiceFixture(t)
	requests := []ApplyMutationRequest{
		{
			ExpectedRevision: revision,
			IdempotencyKey:   "concurrent-a",
			Operation:        "create",
			MutationTime:     "2026-09-03T04:00:00Z",
			Proposed:         decisionProposal(secondMemoryID, "Concurrent decision A", nil),
		},
		{
			ExpectedRevision: revision,
			IdempotencyKey:   "concurrent-b",
			Operation:        "create",
			MutationTime:     "2026-09-03T04:00:01Z",
			Proposed:         decisionProposal(thirdMemoryID, "Concurrent decision B", nil),
		},
	}
	type outcome struct {
		result ApplyMutationResult
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, len(requests))
	for _, request := range requests {
		request := request
		go func() {
			<-start
			result, err := svc.ApplyMutation(context.Background(), request)
			outcomes <- outcome{result: result, err: err}
		}()
	}
	close(start)

	applied := 0
	stale := 0
	for range requests {
		outcome := <-outcomes
		if outcome.err == nil {
			if outcome.result.Status != "applied" {
				t.Fatalf("successful concurrent result = %+v", outcome.result)
			}
			applied++
			continue
		}
		var serviceErr *Error
		if !errors.As(outcome.err, &serviceErr) || serviceErr.Code != CodeStaleRevision {
			t.Fatalf("concurrent loser error = %v, want stale_revision", outcome.err)
		}
		stale++
	}
	if applied != 1 || stale != 1 {
		t.Fatalf("concurrent outcomes: applied=%d stale=%d", applied, stale)
	}
	records, err := memory.LoadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("memory count after concurrent apply = %d, want 2", len(records))
	}
	assertRepositoryHealthy(t, root)
}
