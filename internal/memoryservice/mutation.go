package memoryservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/memory"
	"github.com/runethread/core/internal/repository"
	"github.com/runethread/core/internal/validation"
)

const (
	CodeUnsupportedOperation = "unsupported_operation"
	CodeStaleRevision       = "stale_revision"
	CodeIdempotencyConflict = "idempotency_conflict"
	CodeValidationFailed    = "validation_failed"
	CodeTransactionFailed   = "transaction_failed"
)

var (
	idempotencyKeyRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	safeSlugRE       = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type ProposedDocument struct {
	Memory   memory.Memory `json:"memory"`
	Markdown string        `json:"markdown"`
}

type ApplyMutationRequest struct {
	ExpectedRevision string            `json:"expected_revision"`
	IdempotencyKey   string            `json:"idempotency_key"`
	Operation        string            `json:"operation"`
	MutationTime     string            `json:"mutation_time,omitempty"`
	TargetID         string            `json:"target_id,omitempty"`
	Proposed         *ProposedDocument `json:"proposed,omitempty"`
}

type ApplyMutationResult struct {
	Status           string   `json:"status"`
	Operation        string   `json:"operation"`
	IdempotencyKey   string   `json:"idempotency_key"`
	PreviousRevision string   `json:"previous_revision"`
	Revision         string   `json:"revision"`
	Commit           string   `json:"commit,omitempty"`
	PrimaryMemoryID  string   `json:"primary_memory_id,omitempty"`
	TargetMemoryID   string   `json:"target_memory_id,omitempty"`
	ChangedMemoryIDs []string `json:"changed_memory_ids,omitempty"`
}

type WithdrawRequest struct {
	ExpectedRevision string `json:"expected_revision"`
	IdempotencyKey   string `json:"idempotency_key"`
	TargetID         string `json:"target_id"`
	MutationTime     string `json:"mutation_time"`
}

type mutationPlan struct {
	PrimaryMemoryID  string
	TargetMemoryID   string
	ChangedMemoryIDs []string
}

func (s *Service) ApplyMutation(ctx context.Context, request ApplyMutationRequest) (ApplyMutationResult, error) {
	writer, ok := s.repo.(repository.Writer)
	if !ok {
		return ApplyMutationResult{}, errorf(CodeRepository, "apply", nil, "repository does not provide mutation capability")
	}

	request = normalizeApplyRequest(request)
	if err := validateApplyRequest(request); err != nil {
		return ApplyMutationResult{}, err
	}
	fingerprint, err := applyRequestFingerprint(request)
	if err != nil {
		return ApplyMutationResult{}, errorf(CodeInvalidArgument, "apply", err, "fingerprint mutation request: %v", err)
	}

	// ADR-003 requires exact committed retries to be recognized before the
	// ordinary stale-revision check because the successful prior commit itself
	// advances HEAD.
	applied, found, err := writer.FindAppliedOperation(ctx, request.IdempotencyKey)
	if err != nil {
		return ApplyMutationResult{}, errorf(CodeRepository, "apply", err, "search idempotency history: %v", err)
	}
	if found {
		if applied.RequestSHA256 != fingerprint {
			return ApplyMutationResult{}, errorf(CodeIdempotencyConflict, "apply", nil, "idempotency key %q is already committed with a different request fingerprint", request.IdempotencyKey)
		}
		return resultFromApplied("already_applied", request.ExpectedRevision, applied), nil
	}

	state, err := writer.State(ctx)
	if err != nil {
		return ApplyMutationResult{}, errorf(CodeRepository, "apply", err, "read repository state: %v", err)
	}
	if !state.Clean {
		return ApplyMutationResult{}, errorf(CodeRepositoryDirty, "apply", nil, "working tree has uncommitted changes")
	}
	if state.Branch == "" {
		return ApplyMutationResult{}, errorf(CodeRepository, "apply", nil, "canonical repository is in detached HEAD state")
	}
	if state.Revision != request.ExpectedRevision {
		return ApplyMutationResult{}, errorf(CodeStaleRevision, "apply", nil, "repository revision is %s, expected %s", state.Revision, request.ExpectedRevision)
	}
	issues := validation.Validate(s.root)
	if validation.HasErrors(issues) {
		return ApplyMutationResult{}, errorf(CodeRepositoryInvalid, "apply", nil, "repository has hard validation errors before mutation")
	}

	if request.Operation == "noop" {
		return ApplyMutationResult{
			Status:           "no_op",
			Operation:        request.Operation,
			IdempotencyKey:   request.IdempotencyKey,
			PreviousRevision: state.Revision,
			Revision:         state.Revision,
			TargetMemoryID:   request.TargetID,
		}, nil
	}

	txn, err := writer.BeginTransaction(ctx, request.ExpectedRevision)
	if err != nil {
		return ApplyMutationResult{}, errorf(CodeTransactionFailed, "apply", err, "create isolated mutation transaction: %v", err)
	}
	closed := false
	closeTxn := func() error {
		if closed {
			return nil
		}
		closed = true
		return txn.Close()
	}
	defer func() { _ = closeTxn() }()

	plan, err := applyOperation(txn.Root(), request)
	if err != nil {
		return ApplyMutationResult{}, err
	}
	if err := indexer.Write(txn.Root()); err != nil {
		return ApplyMutationResult{}, errorf(CodeTransactionFailed, "apply", err, "regenerate indexes in mutation transaction: %v", err)
	}
	issues = validation.Validate(txn.Root())
	if validation.HasErrors(issues) {
		return ApplyMutationResult{}, errorf(CodeValidationFailed, "apply", nil, "mutation failed hard validation: %s", validation.RenderText(issues))
	}

	commitMetadata := repository.AppliedOperation{
		IdempotencyKey:   request.IdempotencyKey,
		RequestSHA256:    fingerprint,
		Operation:        request.Operation,
		PrimaryMemoryID:  plan.PrimaryMemoryID,
		TargetMemoryID:   plan.TargetMemoryID,
		ChangedMemoryIDs: plan.ChangedMemoryIDs,
	}
	message, err := repository.FormatMutationCommitMessage(mutationCommitSubject(request.Operation, plan), commitMetadata)
	if err != nil {
		return ApplyMutationResult{}, errorf(CodeTransactionFailed, "apply", err, "format mutation commit: %v", err)
	}
	commit, err := txn.Commit(ctx, message)
	if err != nil {
		return ApplyMutationResult{}, errorf(CodeTransactionFailed, "apply", err, "commit verified mutation transaction: %v", err)
	}
	if err := closeTxn(); err != nil {
		return ApplyMutationResult{}, errorf(CodeTransactionFailed, "apply", err, "close mutation transaction before publish: %v", err)
	}

	if err := writer.Publish(ctx, state.Branch, request.ExpectedRevision, commit); err != nil {
		var stale *repository.StaleRevisionError
		if errors.As(err, &stale) {
			return ApplyMutationResult{}, errorf(CodeStaleRevision, "apply", err, "%v", err)
		}
		var dirty *repository.DirtyRepositoryError
		if errors.As(err, &dirty) {
			return ApplyMutationResult{}, errorf(CodeRepositoryDirty, "apply", err, "%v", err)
		}
		return ApplyMutationResult{}, errorf(CodeTransactionFailed, "apply", err, "publish verified mutation: %v", err)
	}

	return ApplyMutationResult{
		Status:           "applied",
		Operation:        request.Operation,
		IdempotencyKey:   request.IdempotencyKey,
		PreviousRevision: request.ExpectedRevision,
		Revision:         commit,
		Commit:           commit,
		PrimaryMemoryID:  plan.PrimaryMemoryID,
		TargetMemoryID:   plan.TargetMemoryID,
		ChangedMemoryIDs: append([]string(nil), plan.ChangedMemoryIDs...),
	}, nil
}

func (s *Service) Withdraw(ctx context.Context, request WithdrawRequest) (ApplyMutationResult, error) {
	return s.ApplyMutation(ctx, ApplyMutationRequest{
		ExpectedRevision: request.ExpectedRevision,
		IdempotencyKey:   request.IdempotencyKey,
		Operation:        "withdraw",
		MutationTime:     request.MutationTime,
		TargetID:         request.TargetID,
	})
}

func normalizeApplyRequest(request ApplyMutationRequest) ApplyMutationRequest {
	request.ExpectedRevision = strings.TrimSpace(request.ExpectedRevision)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.Operation = strings.ToLower(strings.TrimSpace(request.Operation))
	request.MutationTime = strings.TrimSpace(request.MutationTime)
	request.TargetID = strings.ToLower(strings.TrimSpace(request.TargetID))
	if request.Proposed != nil {
		copy := *request.Proposed
		copy.Memory.ID = strings.ToLower(strings.TrimSpace(copy.Memory.ID))
		request.Proposed = &copy
	}
	return request
}

func validateApplyRequest(request ApplyMutationRequest) error {
	if request.ExpectedRevision == "" {
		return errorf(CodeInvalidArgument, "apply", nil, "expected_revision must not be empty")
	}
	if !idempotencyKeyRE.MatchString(request.IdempotencyKey) {
		return errorf(CodeInvalidArgument, "apply", nil, "idempotency_key must be 1-128 safe token characters")
	}
	switch request.Operation {
	case "create", "update", "correct", "supersede", "resolve", "withdraw", "noop":
	default:
		return errorf(CodeUnsupportedOperation, "apply", nil, "unsupported mutation operation %q", request.Operation)
	}
	if request.Operation != "noop" {
		if _, err := time.Parse(time.RFC3339, request.MutationTime); err != nil {
			return errorf(CodeInvalidArgument, "apply", err, "mutation_time must be RFC3339")
		}
	}
	needsTarget := request.Operation == "update" || request.Operation == "correct" || request.Operation == "supersede" || request.Operation == "resolve" || request.Operation == "withdraw"
	if needsTarget && !memoryIDRE.MatchString(request.TargetID) {
		return errorf(CodeInvalidArgument, "apply", nil, "operation %s requires a valid target_id", request.Operation)
	}
	if !needsTarget && request.TargetID != "" {
		return errorf(CodeInvalidArgument, "apply", nil, "operation %s does not accept target_id", request.Operation)
	}
	needsProposal := request.Operation == "create" || request.Operation == "update" || request.Operation == "correct" || request.Operation == "supersede" || request.Operation == "resolve"
	if needsProposal && request.Proposed == nil {
		return errorf(CodeInvalidArgument, "apply", nil, "operation %s requires proposed memory and Markdown", request.Operation)
	}
	if !needsProposal && request.Proposed != nil {
		return errorf(CodeInvalidArgument, "apply", nil, "operation %s does not accept proposed memory", request.Operation)
	}
	return nil
}

type fingerprintPayload struct {
	ExpectedRevision string            `json:"expected_revision"`
	Operation        string            `json:"operation"`
	MutationTime     string            `json:"mutation_time,omitempty"`
	TargetID         string            `json:"target_id,omitempty"`
	Proposed         *ProposedDocument `json:"proposed,omitempty"`
}

func applyRequestFingerprint(request ApplyMutationRequest) (string, error) {
	data, err := json.Marshal(fingerprintPayload{
		ExpectedRevision: request.ExpectedRevision,
		Operation:        request.Operation,
		MutationTime:     request.MutationTime,
		TargetID:         request.TargetID,
		Proposed:         request.Proposed,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func resultFromApplied(status, previousRevision string, applied repository.AppliedOperation) ApplyMutationResult {
	return ApplyMutationResult{
		Status:           status,
		Operation:        applied.Operation,
		IdempotencyKey:   applied.IdempotencyKey,
		PreviousRevision: previousRevision,
		Revision:         applied.Commit,
		Commit:           applied.Commit,
		PrimaryMemoryID:  applied.PrimaryMemoryID,
		TargetMemoryID:   applied.TargetMemoryID,
		ChangedMemoryIDs: append([]string(nil), applied.ChangedMemoryIDs...),
	}
}

func applyOperation(root string, request ApplyMutationRequest) (mutationPlan, error) {
	switch request.Operation {
	case "create":
		return applyCreate(root, request)
	case "update":
		return applyUpdate(root, request, false)
	case "resolve":
		return applyUpdate(root, request, true)
	case "correct":
		return applyCorrection(root, request)
	case "supersede":
		return applySupersession(root, request)
	case "withdraw":
		return applyWithdrawal(root, request)
	default:
		return mutationPlan{}, errorf(CodeUnsupportedOperation, "apply", nil, "operation %s is not writable", request.Operation)
	}
}

func applyCreate(root string, request ApplyMutationRequest) (mutationPlan, error) {
	proposal := cloneProposed(*request.Proposed)
	if !memoryIDRE.MatchString(proposal.Memory.ID) {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "create proposal requires a valid memory UUID")
	}
	if _, found, err := canonicalDocumentAt(root, proposal.Memory.ID); err != nil {
		return mutationPlan{}, err
	} else if found {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "memory UUID %s already exists", proposal.Memory.ID)
	}
	proposal.Memory.Lifecycle = "active"
	proposal.Memory.Temporal.CreatedAt = request.MutationTime
	proposal.Memory.Temporal.UpdatedAt = request.MutationTime
	doc, err := writeProposedDocument(root, nil, proposal)
	if err != nil {
		return mutationPlan{}, err
	}
	return mutationPlan{PrimaryMemoryID: doc.Memory.ID, ChangedMemoryIDs: []string{doc.Memory.ID}}, nil
}

func applyUpdate(root string, request ApplyMutationRequest, resolving bool) (mutationPlan, error) {
	target, found, err := canonicalDocumentAt(root, request.TargetID)
	if err != nil {
		return mutationPlan{}, err
	}
	if !found {
		return mutationPlan{}, errorf(CodeNotFound, "apply", nil, "target memory %s was not found", request.TargetID)
	}
	if target.Memory.Lifecycle != "active" {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "target memory %s is not active", request.TargetID)
	}
	proposal := cloneProposed(*request.Proposed)
	if proposal.Memory.ID != target.Memory.ID {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "%s proposal UUID must equal target UUID", request.Operation)
	}
	proposal.Memory.Lifecycle = target.Memory.Lifecycle
	proposal.Memory.Temporal.CreatedAt = target.Memory.Temporal.CreatedAt
	proposal.Memory.Temporal.UpdatedAt = request.MutationTime
	if resolving {
		if target.Memory.Type != "open_loop" || proposal.Memory.Type != "open_loop" {
			return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "resolve requires an open_loop target and proposal")
		}
		if !isUnresolvedOpenLoop(target.Memory.OpenLoopStatus) {
			return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "open_loop target is already terminal")
		}
		if !isTerminalOpenLoop(proposal.Memory.OpenLoopStatus) {
			return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "resolve proposal must set open_loop_status to resolved or cancelled")
		}
	} else if !sameOptionalString(target.Memory.OpenLoopStatus, proposal.Memory.OpenLoopStatus) {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "update cannot change open_loop_status; use resolve for terminal transitions")
	}
	doc, err := writeProposedDocument(root, &target, proposal)
	if err != nil {
		return mutationPlan{}, err
	}
	return mutationPlan{PrimaryMemoryID: doc.Memory.ID, TargetMemoryID: target.Memory.ID, ChangedMemoryIDs: []string{doc.Memory.ID}}, nil
}

func applyCorrection(root string, request ApplyMutationRequest) (mutationPlan, error) {
	target, found, err := canonicalDocumentAt(root, request.TargetID)
	if err != nil {
		return mutationPlan{}, err
	}
	if !found {
		return mutationPlan{}, errorf(CodeNotFound, "apply", nil, "target memory %s was not found", request.TargetID)
	}
	if target.Memory.Lifecycle != "active" {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "correction target %s is not active", request.TargetID)
	}
	proposal := cloneProposed(*request.Proposed)
	if proposal.Memory.Type != "correction" {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "correct requires a correction memory")
	}
	if proposal.Memory.ID == target.Memory.ID || !memoryIDRE.MatchString(proposal.Memory.ID) {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "correction requires a new memory UUID")
	}
	if !hasRelationship(proposal.Memory, "corrects", target.Memory.ID) {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "correction memory must contain corrects -> %s", target.Memory.ID)
	}
	if _, exists, err := canonicalDocumentAt(root, proposal.Memory.ID); err != nil {
		return mutationPlan{}, err
	} else if exists {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "correction UUID %s already exists", proposal.Memory.ID)
	}
	proposal.Memory.Lifecycle = "active"
	proposal.Memory.Temporal.CreatedAt = request.MutationTime
	proposal.Memory.Temporal.UpdatedAt = request.MutationTime
	newDoc, err := writeProposedDocument(root, nil, proposal)
	if err != nil {
		return mutationPlan{}, err
	}
	changed := []string{newDoc.Memory.ID}
	if hasRelationship(proposal.Memory, "supersedes", target.Memory.ID) {
		if err := transitionLifecycle(root, target, "superseded", request.MutationTime); err != nil {
			return mutationPlan{}, err
		}
		changed = append(changed, target.Memory.ID)
	}
	sort.Strings(changed)
	return mutationPlan{PrimaryMemoryID: newDoc.Memory.ID, TargetMemoryID: target.Memory.ID, ChangedMemoryIDs: changed}, nil
}

func applySupersession(root string, request ApplyMutationRequest) (mutationPlan, error) {
	target, found, err := canonicalDocumentAt(root, request.TargetID)
	if err != nil {
		return mutationPlan{}, err
	}
	if !found {
		return mutationPlan{}, errorf(CodeNotFound, "apply", nil, "target memory %s was not found", request.TargetID)
	}
	if target.Memory.Lifecycle != "active" {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "supersession target %s is not active", request.TargetID)
	}
	proposal := cloneProposed(*request.Proposed)
	if proposal.Memory.ID == target.Memory.ID || !memoryIDRE.MatchString(proposal.Memory.ID) {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "supersession requires a new memory UUID")
	}
	if !hasRelationship(proposal.Memory, "supersedes", target.Memory.ID) {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "superseding memory must contain supersedes -> %s", target.Memory.ID)
	}
	if _, exists, err := canonicalDocumentAt(root, proposal.Memory.ID); err != nil {
		return mutationPlan{}, err
	} else if exists {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "superseding UUID %s already exists", proposal.Memory.ID)
	}
	proposal.Memory.Lifecycle = "active"
	proposal.Memory.Temporal.CreatedAt = request.MutationTime
	proposal.Memory.Temporal.UpdatedAt = request.MutationTime
	newDoc, err := writeProposedDocument(root, nil, proposal)
	if err != nil {
		return mutationPlan{}, err
	}
	if err := transitionLifecycle(root, target, "superseded", request.MutationTime); err != nil {
		return mutationPlan{}, err
	}
	changed := []string{newDoc.Memory.ID, target.Memory.ID}
	sort.Strings(changed)
	return mutationPlan{PrimaryMemoryID: newDoc.Memory.ID, TargetMemoryID: target.Memory.ID, ChangedMemoryIDs: changed}, nil
}

func applyWithdrawal(root string, request ApplyMutationRequest) (mutationPlan, error) {
	target, found, err := canonicalDocumentAt(root, request.TargetID)
	if err != nil {
		return mutationPlan{}, err
	}
	if !found {
		return mutationPlan{}, errorf(CodeNotFound, "apply", nil, "target memory %s was not found", request.TargetID)
	}
	if target.Memory.Lifecycle != "active" {
		return mutationPlan{}, errorf(CodeInvalidArgument, "apply", nil, "withdraw target %s is not active", request.TargetID)
	}
	if err := transitionLifecycle(root, target, "withdrawn", request.MutationTime); err != nil {
		return mutationPlan{}, err
	}
	return mutationPlan{PrimaryMemoryID: target.Memory.ID, TargetMemoryID: target.Memory.ID, ChangedMemoryIDs: []string{target.Memory.ID}}, nil
}

func cloneProposed(proposal ProposedDocument) ProposedDocument {
	copy := proposal
	copy.Memory.Projects = append([]string(nil), proposal.Memory.Projects...)
	copy.Memory.Topics = append([]string(nil), proposal.Memory.Topics...)
	copy.Memory.Tags = append([]string(nil), proposal.Memory.Tags...)
	copy.Memory.Aliases = append([]string(nil), proposal.Memory.Aliases...)
	copy.Memory.Entities = append([]memory.Entity(nil), proposal.Memory.Entities...)
	copy.Memory.Relationships = append([]memory.Relationship(nil), proposal.Memory.Relationships...)
	copy.Memory.Provenance.Sources = append([]memory.Source(nil), proposal.Memory.Provenance.Sources...)
	return copy
}

func canonicalDocumentAt(root, id string) (Document, bool, error) {
	records, err := memory.LoadAll(root)
	if err != nil {
		return Document{}, false, errorf(CodeRepositoryInvalid, "apply", err, "load canonical memories: %v", err)
	}
	var match *memory.Record
	for i := range records {
		if records[i].Memory.ID != id {
			continue
		}
		if match != nil {
			return Document{}, false, errorf(CodeRepositoryInvalid, "apply", nil, "memory UUID %s occurs more than once", id)
		}
		match = &records[i]
	}
	if match == nil {
		return Document{}, false, nil
	}
	rel, err := filepath.Rel(root, match.Path)
	if err != nil {
		return Document{}, false, errorf(CodeRepository, "apply", err, "resolve canonical sidecar path: %v", err)
	}
	doc, err := loadDocument(root, filepath.ToSlash(rel))
	if err != nil {
		return Document{}, false, err
	}
	return doc, true, nil
}

func writeProposedDocument(root string, existing *Document, proposal ProposedDocument) (Document, error) {
	markdownRel, sidecarRel, err := canonicalPaths(proposal.Memory)
	if err != nil {
		return Document{}, err
	}
	proposal.Memory.ContentPath = markdownRel
	data, err := json.MarshalIndent(proposal.Memory, "", "  ")
	if err != nil {
		return Document{}, errorf(CodeInvalidArgument, "apply", err, "encode proposed memory: %v", err)
	}
	data = append(data, '\n')
	if _, problems := memory.Decode(data); len(problems) > 0 {
		return Document{}, errorf(CodeInvalidArgument, "apply", nil, "proposed memory is invalid: %s", problems[0].Error())
	}

	if existing == nil {
		for _, rel := range []string{markdownRel, sidecarRel} {
			if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
				return Document{}, errorf(CodeInvalidArgument, "apply", nil, "canonical path %s already exists", rel)
			} else if !os.IsNotExist(err) {
				return Document{}, errorf(CodeTransactionFailed, "apply", err, "inspect canonical path %s: %v", rel, err)
			}
		}
	}
	for _, rel := range []string{markdownRel, sidecarRel} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(rel))), 0o755); err != nil {
			return Document{}, errorf(CodeTransactionFailed, "apply", err, "create parent for %s: %v", rel, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(markdownRel)), []byte(proposal.Markdown), 0o644); err != nil {
		return Document{}, errorf(CodeTransactionFailed, "apply", err, "write %s: %v", markdownRel, err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(sidecarRel)), data, 0o644); err != nil {
		return Document{}, errorf(CodeTransactionFailed, "apply", err, "write %s: %v", sidecarRel, err)
	}
	if existing != nil {
		for _, oldRel := range []string{existing.MarkdownPath, existing.SidecarPath} {
			if oldRel == markdownRel || oldRel == sidecarRel {
				continue
			}
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(oldRel))); err != nil && !os.IsNotExist(err) {
				return Document{}, errorf(CodeTransactionFailed, "apply", err, "remove obsolete canonical path %s: %v", oldRel, err)
			}
		}
	}
	return Document{SidecarPath: sidecarRel, MarkdownPath: markdownRel, Memory: proposal.Memory, Markdown: proposal.Markdown}, nil
}

func transitionLifecycle(root string, target Document, lifecycle, mutationTime string) error {
	target.Memory.Lifecycle = lifecycle
	target.Memory.Temporal.UpdatedAt = mutationTime
	data, err := json.MarshalIndent(target.Memory, "", "  ")
	if err != nil {
		return errorf(CodeTransactionFailed, "apply", err, "encode lifecycle transition: %v", err)
	}
	data = append(data, '\n')
	if _, problems := memory.Decode(data); len(problems) > 0 {
		return errorf(CodeInvalidArgument, "apply", nil, "lifecycle transition is invalid: %s", problems[0].Error())
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(target.SidecarPath)), data, 0o644); err != nil {
		return errorf(CodeTransactionFailed, "apply", err, "write lifecycle transition %s: %v", target.SidecarPath, err)
	}
	return nil
}

func canonicalPaths(m memory.Memory) (markdownRel, sidecarRel string, err error) {
	if !memoryIDRE.MatchString(m.ID) {
		return "", "", errorf(CodeInvalidArgument, "apply", nil, "proposed memory requires a valid UUID")
	}
	dir := "memories/general"
	if len(m.Projects) > 0 {
		project := strings.TrimSpace(m.Projects[0])
		if !safeSlugRE.MatchString(project) {
			return "", "", errorf(CodeInvalidArgument, "apply", nil, "first project %q is not a canonical slug", project)
		}
		dir = "memories/projects/" + project
	}
	slug := titleSlug(m.Title)
	base := slug + "--" + m.ID[:8]
	markdownRel = dir + "/" + base + ".md"
	sidecarRel = dir + "/" + base + ".json"
	return markdownRel, sidecarRel, nil
}

func titleSlug(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
		} else if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
		if b.Len() >= 80 {
			break
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "memory"
	}
	return slug
}

func hasRelationship(m memory.Memory, relationshipType, targetID string) bool {
	for _, relationship := range m.Relationships {
		if relationship.Type == relationshipType && relationship.TargetID == targetID {
			return true
		}
	}
	return false
}

func sameOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func isUnresolvedOpenLoop(status *string) bool {
	if status == nil {
		return false
	}
	return *status == "open" || *status == "blocked" || *status == "deferred"
}

func isTerminalOpenLoop(status *string) bool {
	if status == nil {
		return false
	}
	return *status == "resolved" || *status == "cancelled"
}

func mutationCommitSubject(operation string, plan mutationPlan) string {
	id := plan.PrimaryMemoryID
	if id == "" {
		id = plan.TargetMemoryID
	}
	if id == "" {
		return "memory: " + operation
	}
	return fmt.Sprintf("memory: %s %s", operation, id)
}
