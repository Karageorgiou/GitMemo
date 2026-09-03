package memoryservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/memory"
	"github.com/runethread/core/internal/repository"
	"github.com/runethread/core/internal/trust"
	"github.com/runethread/core/internal/validation"
)

const (
	CodeInvalidArgument   = "invalid_argument"
	CodeRepository        = "repository_error"
	CodeRepositoryDirty   = "repository_dirty"
	CodeRepositoryInvalid = "repository_invalid"
	CodeNotFound          = "not_found"
	CodeIndexUnavailable  = "index_unavailable"
)

var memoryIDRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type Error struct {
	Code      string `json:"code"`
	Operation string `json:"operation"`
	Message   string `json:"message"`
	Cause     error  `json:"-"`
}

func (e *Error) Error() string {
	if e.Operation == "" {
		return e.Message
	}
	return e.Operation + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func errorf(code, operation string, cause error, format string, args ...any) error {
	return &Error{Code: code, Operation: operation, Message: fmt.Sprintf(format, args...), Cause: cause}
}

type Service struct {
	root    string
	repo    repository.Reader
	writeMu sync.Mutex
}

func Open(root string) (*Service, error) {
	repo, err := repository.OpenGit(root)
	if err != nil {
		return nil, errorf(CodeRepository, "open", err, "open Git repository: %v", err)
	}
	return New(repo), nil
}

func New(repo repository.Reader) *Service {
	return &Service{root: repo.Root(), repo: repo}
}

type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type SearchResponse struct {
	Results []indexer.SearchResult `json:"results"`
}

func (s *Service) Search(_ context.Context, request SearchRequest) (SearchResponse, error) {
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return SearchResponse{}, errorf(CodeInvalidArgument, "search", nil, "query must not be empty")
	}
	limit := request.Limit
	if limit == 0 {
		limit = 10
	}
	if limit < 1 || limit > 100 {
		return SearchResponse{}, errorf(CodeInvalidArgument, "search", nil, "limit must be between 1 and 100")
	}
	results, err := indexer.Search(s.root, query, limit)
	if err != nil {
		return SearchResponse{}, errorf(CodeIndexUnavailable, "search", err, "%v", err)
	}
	return SearchResponse{Results: results}, nil
}

type Document struct {
	SidecarPath  string        `json:"sidecar_path"`
	MarkdownPath string        `json:"markdown_path"`
	Memory       memory.Memory `json:"memory"`
	Markdown     string        `json:"markdown"`
}

type GetResponse struct {
	Document Document `json:"document"`
}

func (s *Service) Get(_ context.Context, id string) (GetResponse, error) {
	doc, found, err := s.getCanonical(strings.ToLower(strings.TrimSpace(id)))
	if err != nil {
		return GetResponse{}, err
	}
	if !found {
		return GetResponse{}, errorf(CodeNotFound, "get", nil, "memory %s was not found", id)
	}
	return GetResponse{Document: doc}, nil
}

type PrepareMutationRequest struct {
	Query        string   `json:"query,omitempty"`
	CandidateIDs []string `json:"candidate_ids,omitempty"`
	Limit        int      `json:"limit,omitempty"`
}

type PreparedMutation struct {
	ExpectedRevision string     `json:"expected_revision"`
	Candidates       []Document `json:"candidates"`
	LegalOperations  []string   `json:"legal_operations"`
}

func (s *Service) PrepareMutation(ctx context.Context, request PrepareMutationRequest) (PreparedMutation, error) {
	state, err := s.repo.State(ctx)
	if err != nil {
		return PreparedMutation{}, errorf(CodeRepository, "prepare", err, "read repository state: %v", err)
	}
	if !state.Clean {
		return PreparedMutation{}, errorf(CodeRepositoryDirty, "prepare", nil, "working tree has uncommitted changes")
	}
	issues := validation.Validate(s.root)
	if validation.HasErrors(issues) {
		return PreparedMutation{}, errorf(CodeRepositoryInvalid, "prepare", nil, "repository has hard validation errors")
	}

	ids := make([]string, 0, len(request.CandidateIDs)+10)
	seen := map[string]bool{}
	appendID := func(id string) {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range request.CandidateIDs {
		if !memoryIDRE.MatchString(strings.ToLower(strings.TrimSpace(id))) {
			return PreparedMutation{}, errorf(CodeInvalidArgument, "prepare", nil, "invalid candidate UUID %q", id)
		}
		appendID(id)
	}

	query := strings.TrimSpace(request.Query)
	if query != "" {
		limit := request.Limit
		if limit == 0 {
			limit = 10
		}
		if limit < 1 || limit > 100 {
			return PreparedMutation{}, errorf(CodeInvalidArgument, "prepare", nil, "limit must be between 1 and 100")
		}
		results, searchErr := indexer.Search(s.root, query, limit)
		if searchErr != nil {
			return PreparedMutation{}, errorf(CodeIndexUnavailable, "prepare", searchErr, "retrieve mutation candidates: %v", searchErr)
		}
		for _, result := range results {
			appendID(result.ID)
		}
	}

	candidates := make([]Document, 0, len(ids))
	for _, id := range ids {
		doc, found, getErr := s.getCanonical(id)
		if getErr != nil {
			return PreparedMutation{}, getErr
		}
		if !found {
			return PreparedMutation{}, errorf(CodeNotFound, "prepare", nil, "candidate memory %s was not found", id)
		}
		candidates = append(candidates, doc)
	}

	return PreparedMutation{
		ExpectedRevision: state.Revision,
		Candidates:       candidates,
		LegalOperations:  legalOperations(candidates),
	}, nil
}

type StatusResponse struct {
	Root               string             `json:"root"`
	Revision           string             `json:"revision"`
	Branch             string             `json:"branch,omitempty"`
	Clean              bool               `json:"clean"`
	DirtyEntries       []string           `json:"dirty_entries,omitempty"`
	TrustOK            bool               `json:"trust_ok"`
	TrustProblems      int                `json:"trust_problems"`
	ValidationErrors   int                `json:"validation_errors"`
	ValidationWarnings int                `json:"validation_warnings"`
	ValidationIssues   []validation.Issue `json:"validation_issues,omitempty"`
	IndexCurrent       bool               `json:"index_current"`
	StaleIndexPaths    []string           `json:"stale_index_paths,omitempty"`
	IndexError         string             `json:"index_error,omitempty"`
	ReleaseVersion     string             `json:"release_version"`
	RepositoryFormat   int                `json:"repository_format"`
	SchemaVersion      int                `json:"schema_version"`
	ContractVersion    int                `json:"contract_version"`
	IndexFormatVersion int                `json:"index_format_version"`
	TrustLockVersion   int                `json:"trust_lock_version"`
}

func (s *Service) Status(ctx context.Context) (StatusResponse, error) {
	state, err := s.repo.State(ctx)
	if err != nil {
		return StatusResponse{}, errorf(CodeRepository, "status", err, "read repository state: %v", err)
	}
	trustProblems := trust.Check(s.root)
	issues := validation.Validate(s.root)
	errorsCount, warningsCount := issueCounts(issues)
	stale, indexErr := indexer.Check(s.root)
	indexError := ""
	if indexErr != nil {
		indexError = indexErr.Error()
	}
	return StatusResponse{
		Root:               s.root,
		Revision:           state.Revision,
		Branch:             state.Branch,
		Clean:              state.Clean,
		DirtyEntries:       state.DirtyEntries,
		TrustOK:            len(trustProblems) == 0,
		TrustProblems:      len(trustProblems),
		ValidationErrors:   errorsCount,
		ValidationWarnings: warningsCount,
		ValidationIssues:   issues,
		IndexCurrent:       indexErr == nil && len(stale) == 0,
		StaleIndexPaths:    stale,
		IndexError:         indexError,
		ReleaseVersion:     buildinfo.ReleaseVersion,
		RepositoryFormat:   buildinfo.RepositoryFormatVersion,
		SchemaVersion:      buildinfo.SchemaVersion,
		ContractVersion:    buildinfo.ContractVersion,
		IndexFormatVersion: buildinfo.IndexFormatVersion,
		TrustLockVersion:   buildinfo.TrustLockVersion,
	}, nil
}

func (s *Service) getCanonical(id string) (Document, bool, error) {
	if !memoryIDRE.MatchString(id) {
		return Document{}, false, errorf(CodeInvalidArgument, "get", nil, "invalid memory UUID %q", id)
	}

	if result, found, err := indexer.LookupByID(s.root, id); err == nil && found {
		sidecar := strings.TrimSuffix(result.ContentPath, ".md") + ".json"
		doc, loadErr := loadDocument(s.root, sidecar)
		if loadErr != nil {
			return Document{}, false, loadErr
		}
		if doc.Memory.ID != id {
			return Document{}, false, errorf(CodeRepositoryInvalid, "get", nil, "index resolved %s to sidecar containing UUID %s", id, doc.Memory.ID)
		}
		return doc, true, nil
	}

	records, err := memory.LoadAll(s.root)
	if err != nil {
		return Document{}, false, errorf(CodeRepositoryInvalid, "get", err, "load canonical memories: %v", err)
	}
	var match *memory.Record
	for i := range records {
		if records[i].Memory.ID != id {
			continue
		}
		if match != nil {
			return Document{}, false, errorf(CodeRepositoryInvalid, "get", nil, "memory UUID %s occurs more than once", id)
		}
		match = &records[i]
	}
	if match == nil {
		return Document{}, false, nil
	}
	rel, err := filepath.Rel(s.root, match.Path)
	if err != nil {
		return Document{}, false, errorf(CodeRepository, "get", err, "resolve sidecar path: %v", err)
	}
	doc, loadErr := loadDocument(s.root, filepath.ToSlash(rel))
	if loadErr != nil {
		return Document{}, false, loadErr
	}
	return doc, true, nil
}

func loadDocument(root, sidecarRel string) (Document, error) {
	sidecarRel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(sidecarRel)))
	sidecarPath := filepath.Join(root, filepath.FromSlash(sidecarRel))
	m, problems := memory.Load(sidecarPath)
	if len(problems) > 0 {
		return Document{}, errorf(CodeRepositoryInvalid, "get", nil, "%s: %s", sidecarRel, problems[0].Error())
	}
	markdownRel := m.ContentPath
	markdown, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(markdownRel)))
	if err != nil {
		return Document{}, errorf(CodeRepositoryInvalid, "get", err, "read %s: %v", markdownRel, err)
	}
	return Document{
		SidecarPath:  sidecarRel,
		MarkdownPath: markdownRel,
		Memory:       m,
		Markdown:     string(markdown),
	}, nil
}

func legalOperations(candidates []Document) []string {
	operations := map[string]bool{
		"create": true,
		"noop":   true,
	}
	for _, candidate := range candidates {
		if candidate.Memory.Lifecycle != "active" {
			continue
		}
		operations["update"] = true
		operations["correct"] = true
		operations["supersede"] = true
		operations["withdraw"] = true
		if candidate.Memory.Type == "open_loop" && isUnresolvedOpenLoop(candidate.Memory.OpenLoopStatus) {
			operations["resolve"] = true
		}
	}
	out := make([]string, 0, len(operations))
	for operation := range operations {
		out = append(out, operation)
	}
	sort.Strings(out)
	return out
}

func issueCounts(issues []validation.Issue) (errorsCount, warningsCount int) {
	for _, issue := range issues {
		switch issue.Severity {
		case "ERROR":
			errorsCount++
		case "WARNING":
			warningsCount++
		}
	}
	return errorsCount, warningsCount
}
