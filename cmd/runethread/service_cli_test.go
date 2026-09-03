package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runethread/core/internal/memory"
	"github.com/runethread/core/internal/memoryservice"
	"github.com/runethread/core/internal/starter"
)

func TestMemoryServiceCLIEndToEnd(t *testing.T) {
	root, initialRevision := makeCLIServiceRepo(t)

	preparePath := writeCLIRequestFile(t, root, "prepare.json", memoryservice.PrepareMutationRequest{})
	code, stdout, stderr := runCLIWithCapturedServiceIO(t, []string{"prepare", "--root", root, "--request", preparePath})
	if code != 0 {
		t.Fatalf("prepare exit=%d stderr=%s", code, stderr)
	}
	var prepared memoryservice.PreparedMutation
	decodeCLIOutput(t, stdout, &prepared)
	if prepared.ExpectedRevision != initialRevision {
		t.Fatalf("prepare revision=%s want=%s", prepared.ExpectedRevision, initialRevision)
	}
	if !containsString(prepared.LegalOperations, "create") || !containsString(prepared.LegalOperations, "noop") {
		t.Fatalf("prepare legal operations=%v", prepared.LegalOperations)
	}

	code, stdout, stderr = runCLIWithCapturedServiceIO(t, []string{"status", "--root", root})
	if code != 0 {
		t.Fatalf("status exit=%d stderr=%s", code, stderr)
	}
	var status memoryservice.StatusResponse
	decodeCLIOutput(t, stdout, &status)
	if status.Revision != initialRevision || !status.Clean || !status.TrustOK || !status.IndexCurrent || status.ValidationErrors != 0 {
		t.Fatalf("unexpected status=%+v", status)
	}

	id := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	createRequest := memoryservice.ApplyMutationRequest{
		ExpectedRevision: initialRevision,
		IdempotencyKey:   "cli-create-alpha",
		Operation:        "create",
		MutationTime:     "2026-09-03T12:00:00Z",
		Proposed:         cliDecisionProposal(id, "Choose deterministic memory writes"),
	}
	createPath := writeCLIRequestFile(t, root, "apply-create.json", createRequest)
	code, stdout, stderr = runCLIWithCapturedServiceIO(t, []string{"apply", "--root", root, "--request", createPath})
	if code != 0 {
		t.Fatalf("apply exit=%d stderr=%s", code, stderr)
	}
	var applied memoryservice.ApplyMutationResult
	decodeCLIOutput(t, stdout, &applied)
	if applied.Status != "applied" || applied.Commit == "" || applied.Revision != applied.Commit || applied.PrimaryMemoryID != id {
		t.Fatalf("unexpected apply result=%+v", applied)
	}

	code, stdout, stderr = runCLIWithCapturedServiceIO(t, []string{"apply", "--root", root, "--request", createPath})
	if code != 0 {
		t.Fatalf("idempotent retry exit=%d stderr=%s", code, stderr)
	}
	var retry memoryservice.ApplyMutationResult
	decodeCLIOutput(t, stdout, &retry)
	if retry.Status != "already_applied" || retry.Commit != applied.Commit || retry.Revision != applied.Commit {
		t.Fatalf("unexpected retry result=%+v first=%+v", retry, applied)
	}

	code, stdout, stderr = runCLIWithCapturedServiceIO(t, []string{"get", "--root", root, id})
	if code != 0 {
		t.Fatalf("get exit=%d stderr=%s", code, stderr)
	}
	var got memoryservice.GetResponse
	decodeCLIOutput(t, stdout, &got)
	if got.Document.Memory.ID != id || got.Document.Memory.Lifecycle != "active" || got.Document.Markdown == "" {
		t.Fatalf("unexpected get response=%+v", got.Document.Memory)
	}

	staleID := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	staleRequest := memoryservice.ApplyMutationRequest{
		ExpectedRevision: initialRevision,
		IdempotencyKey:   "cli-stale-create",
		Operation:        "create",
		MutationTime:     "2026-09-03T12:01:00Z",
		Proposed:         cliDecisionProposal(staleID, "This stale write must fail"),
	}
	stalePath := writeCLIRequestFile(t, root, "apply-stale.json", staleRequest)
	code, stdout, stderr = runCLIWithCapturedServiceIO(t, []string{"apply", "--root", root, "--request", stalePath})
	if code != 1 || strings.TrimSpace(stdout) != "" {
		t.Fatalf("stale apply exit=%d stdout=%q stderr=%s", code, stdout, stderr)
	}
	var staleError serviceCLIError
	decodeCLIOutput(t, stderr, &staleError)
	if staleError.Error.Code != memoryservice.CodeStaleRevision {
		t.Fatalf("stale error=%+v", staleError)
	}

	withdrawRequest := memoryservice.WithdrawRequest{
		ExpectedRevision: applied.Revision,
		IdempotencyKey:   "cli-withdraw-alpha",
		TargetID:         id,
		MutationTime:     "2026-09-03T12:02:00Z",
	}
	withdrawPath := writeCLIRequestFile(t, root, "withdraw.json", withdrawRequest)
	code, stdout, stderr = runCLIWithCapturedServiceIO(t, []string{"withdraw", "--root", root, "--request", withdrawPath})
	if code != 0 {
		t.Fatalf("withdraw exit=%d stderr=%s", code, stderr)
	}
	var withdrawn memoryservice.ApplyMutationResult
	decodeCLIOutput(t, stdout, &withdrawn)
	if withdrawn.Status != "applied" || withdrawn.Operation != "withdraw" || withdrawn.Revision == applied.Revision {
		t.Fatalf("unexpected withdraw result=%+v", withdrawn)
	}

	code, stdout, stderr = runCLIWithCapturedServiceIO(t, []string{"get", "--root", root, id})
	if code != 0 {
		t.Fatalf("get after withdraw exit=%d stderr=%s", code, stderr)
	}
	decodeCLIOutput(t, stdout, &got)
	if got.Document.Memory.Lifecycle != "withdrawn" {
		t.Fatalf("lifecycle after withdraw=%q", got.Document.Memory.Lifecycle)
	}
}

func TestMemoryServiceCLIRejectsUnknownJSONField(t *testing.T) {
	root, _ := makeCLIServiceRepo(t)
	requestPath := filepath.Join(t.TempDir(), "bad-request.json")
	if err := os.WriteFile(requestPath, []byte(`{"unknown":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLIWithCapturedServiceIO(t, []string{"prepare", "--root", root, "--request", requestPath})
	if code != 2 || strings.TrimSpace(stdout) != "" {
		t.Fatalf("bad request exit=%d stdout=%q stderr=%s", code, stdout, stderr)
	}
	var payload serviceCLIError
	decodeCLIOutput(t, stderr, &payload)
	if payload.Error.Code != memoryservice.CodeInvalidArgument || payload.Error.Operation != "prepare" {
		t.Fatalf("bad request error=%+v", payload)
	}
}

func makeCLIServiceRepo(t *testing.T) (string, string) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not available")
	}
	root := filepath.Join(t.TempDir(), "memory")
	if err := starter.Init(root); err != nil {
		t.Fatal(err)
	}
	runCLIGit(t, git, root, "init", "-b", "main")
	runCLIGit(t, git, root, "config", "user.name", "Runethread CLI Test")
	runCLIGit(t, git, root, "config", "user.email", "runethread-cli-test@example.invalid")
	runCLIGit(t, git, root, "add", "-A")
	runCLIGit(t, git, root, "commit", "-m", "initial memory repository")
	return root, strings.TrimSpace(outputCLIGit(t, git, root, "rev-parse", "HEAD"))
}

func cliDecisionProposal(id, title string) *memoryservice.ProposedDocument {
	m := memory.Memory{
		SchemaVersion: 1,
		ID:            id,
		Title:         title,
		Type:          "decision",
		Lifecycle:     "active",
		Summary:       "Use deterministic, validated Git-backed memory writes.",
		Projects:      []string{},
		Topics:        []string{"memory"},
		Tags:          []string{"deterministic"},
		Aliases:       []string{},
		Entities:      []memory.Entity{},
		Importance:    "normal",
		Provenance: memory.Provenance{
			Basis:                 "user_stated",
			Confidence:            "high",
			ExplicitMemoryRequest: true,
			Sources: []memory.Source{{
				Kind:    "conversation",
				Locator: "cli-integration-test",
			}},
		},
		Relationships: []memory.Relationship{},
		Sensitivity:   "routine",
	}
	markdown := "# " + title + "\n\n**Memory ID:** `" + id + "`  \n**Type:** `decision`\n\n## Context\n\nMemory writes need deterministic storage boundaries.\n\n## Decision\n\nUse the verified MemoryService mutation path.\n\n## Rationale\n\nIt validates before publishing canonical Git state.\n"
	return &memoryservice.ProposedDocument{Memory: m, Markdown: markdown}
}

func writeCLIRequestFile(t *testing.T, _ string, name string, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runCLIWithCapturedServiceIO(t *testing.T, args []string) (int, string, string) {
	t.Helper()
	oldIn, oldOut, oldErr := serviceCLIStdin, serviceCLIStdout, serviceCLIStderr
	var stdout, stderr bytes.Buffer
	serviceCLIStdin = bytes.NewReader(nil)
	serviceCLIStdout = &stdout
	serviceCLIStderr = &stderr
	t.Cleanup(func() {
		serviceCLIStdin, serviceCLIStdout, serviceCLIStderr = oldIn, oldOut, oldErr
	})
	code := run(args)
	serviceCLIStdin, serviceCLIStdout, serviceCLIStderr = oldIn, oldOut, oldErr
	return code, stdout.String(), stderr.String()
}

func decodeCLIOutput(t *testing.T, data string, target any) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode CLI JSON %q: %v", data, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("CLI JSON contains trailing value: %q", data)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func runCLIGit(t *testing.T, git, root string, args ...string) {
	t.Helper()
	cmd := exec.Command(git, append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func outputCLIGit(t *testing.T, git, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command(git, append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}
