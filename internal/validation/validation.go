package validation

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/runethread/core/internal/indexer"
	"github.com/runethread/core/internal/memory"
	"github.com/runethread/core/internal/trust"
)

type Issue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

var (
	filenameRE     = regexp.MustCompile(`^(?P<slug>[a-z0-9]+(?:-[a-z0-9]+)*)--(?P<short>[0-9a-f]{8})\.(?:json|md)$`)
	markdownIDRE   = regexp.MustCompile(`(?m)^\*\*Memory ID:\*\* ` + "`" + `([^` + "`" + `]+)` + "`" + `\s*$`)
	markdownTypeRE = regexp.MustCompile(`(?m)^\*\*Type:\*\* ` + "`" + `([^` + "`" + `]+)` + "`" + `\s*$`)
)

var templateMarkers = []string{"<Canonical title>", "<full UUID>", "<memory type>", "<State ", "<Explain ", "<Describe ", "<Record ", "<Provide ", "<Preserve ", "<List ", "<Add ", "<Give "}

func Validate(root string) []Issue {
	rootAbs, _ := filepath.Abs(root)
	root = rootAbs
	var issues []Issue
	add := func(code, path, message string) {
		issues = append(issues, Issue{Severity: "ERROR", Code: code, Path: rel(root, path), Message: message})
	}
	warn := func(code, path, message string) {
		issues = append(issues, Issue{Severity: "WARNING", Code: code, Path: rel(root, path), Message: message})
	}

	for _, problem := range trust.Check(root) {
		add("TRUST_LOCK", filepath.Join(root, filepath.FromSlash(problem.Path)), problem.Message)
	}

	if err := validateSchemaContractForRepository(root); err != nil {
		add("SCHEMA_CONTRACT", filepath.Join(root, "schema", "memory-item.schema.json"), err.Error())
	}
	sidecars, markdown, err := discoverMemoryFiles(root)
	if err != nil {
		add("DISCOVERY", filepath.Join(root, "memories"), err.Error())
		return sortIssues(issues)
	}

	sidecarSet := map[string]bool{}
	markdownSet := map[string]bool{}
	for _, p := range sidecars {
		sidecarSet[clean(p)] = true
	}
	for _, p := range markdown {
		markdownSet[clean(p)] = true
	}
	for _, p := range sidecars {
		if !markdownSet[clean(strings.TrimSuffix(p, ".json")+".md")] {
			add("MISSING_MARKDOWN", p, "missing paired Markdown file")
		}
	}
	for _, p := range markdown {
		if !sidecarSet[clean(strings.TrimSuffix(p, ".md")+".json")] {
			add("ORPHAN_MARKDOWN", p, "missing paired JSON sidecar")
		}
	}

	records := make([]memory.Record, 0, len(sidecars))
	byID := map[string][]memory.Record{}
	for _, p := range sidecars {
		m, problems := loadMemorySidecar(root, p)
		if len(problems) > 0 {
			for _, problem := range problems {
				add("SCHEMA", p, problem.Error())
			}
			continue
		}
		record := memory.Record{Path: p, Memory: m}
		records = append(records, record)
		byID[m.ID] = append(byID[m.ID], record)
	}
	for id, matches := range byID {
		if len(matches) > 1 {
			for _, r := range matches {
				add("UUID_DUPLICATE", r.Path, fmt.Sprintf("UUID %s occurs in %d sidecars", id, len(matches)))
			}
		}
	}

	for _, r := range records {
		checkFilenameAndPath(root, r, &issues)
		checkMarkdown(root, r, &issues)
	}

	incoming := map[string]int{}
	for _, r := range records {
		seen := map[string]bool{}
		for _, edge := range r.Memory.Relationships {
			key := edge.Type + "\x00" + edge.TargetID
			if seen[key] {
				add("REL_DUPLICATE", r.Path, fmt.Sprintf("duplicate logical relationship (%s, %s)", edge.Type, edge.TargetID))
			}
			seen[key] = true
			if edge.TargetID == r.Memory.ID {
				add("REL_SELF", r.Path, "memory must not target itself")
			}
			targets := byID[edge.TargetID]
			if len(targets) == 0 {
				add("REL_TARGET_MISSING", r.Path, "relationship target does not exist: "+edge.TargetID)
			}
			if len(targets) > 1 {
				add("REL_TARGET_AMBIGUOUS", r.Path, "relationship target resolves to duplicate UUID: "+edge.TargetID)
			}
			if edge.Type == "supersedes" {
				incoming[edge.TargetID]++
				if len(targets) == 1 && targets[0].Memory.Lifecycle != "superseded" {
					add("SUPERSEDES_TARGET_NOT_SUPERSEDED", r.Path, "supersedes target lifecycle must be superseded")
				}
			}
		}
	}
	for _, r := range records {
		if r.Memory.Lifecycle == "superseded" && incoming[r.Memory.ID] == 0 {
			add("SUPERSEDED_NO_INCOMING", r.Path, "superseded memory has no incoming supersedes relationship")
		}
	}
	checkSupersessionCycles(root, records, byID, &issues)
	checkTemporal(root, records, &issues)

	if len(records) == len(sidecars) {
		stale, idxErr := indexer.Check(root)
		if idxErr != nil {
			add("INDEX_CHECK", filepath.Join(root, "index"), idxErr.Error())
		} else {
			for _, p := range stale {
				warn("INDEX_STALE", filepath.Join(root, filepath.FromSlash(p)), "generated index is missing or stale; canonical memories/projects remain authoritative and the index may be regenerated")
			}
		}
	}
	return sortIssues(issues)
}

func checkFilenameAndPath(root string, r memory.Record, issues *[]Issue) {
	add := func(code, path, message string) {
		*issues = append(*issues, Issue{Severity: "ERROR", Code: code, Path: rel(root, path), Message: message})
	}
	base := filepath.Base(r.Path)
	match := filenameRE.FindStringSubmatch(base)
	if match == nil {
		add("FILENAME_FORMAT", r.Path, "sidecar filename must be <slug>--<8 hex>.json")
	} else {
		short := match[filenameRE.SubexpIndex("short")]
		if len(r.Memory.ID) >= 8 && short != r.Memory.ID[:8] {
			add("UUID_FILENAME", r.Path, fmt.Sprintf("filename UUID suffix %s does not match UUID prefix %s", short, r.Memory.ID[:8]))
		}
	}
	expectedRel, _ := filepath.Rel(root, strings.TrimSuffix(r.Path, ".json")+".md")
	expectedRel = filepath.ToSlash(expectedRel)
	if r.Memory.ContentPath != expectedRel {
		add("CONTENT_PATH_MISMATCH", r.Path, fmt.Sprintf("content_path must be %q", expectedRel))
	}
	resolved := filepath.Clean(filepath.Join(root, filepath.FromSlash(r.Memory.ContentPath)))
	relPath, err := filepath.Rel(root, resolved)
	if err != nil || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
		add("CONTENT_PATH_ESCAPE", r.Path, "content_path escapes repository root")
	} else if err := requireContentPathFile(root, r.Memory.ContentPath); err != nil {
		add("CONTENT_PATH_MISSING", r.Path, "content_path does not resolve to a regular repository file")
	}
}

func checkMarkdown(root string, r memory.Record, issues *[]Issue) {
	path := strings.TrimSuffix(r.Path, ".json") + ".md"
	data, err := readMarkdownFile(root, path)
	if err != nil {
		return
	}
	text := string(data)
	add := func(code, message string) {
		*issues = append(*issues, Issue{Severity: "ERROR", Code: code, Path: rel(root, path), Message: message})
	}
	lines := strings.Split(text, "\n")
	first := ""
	if len(lines) > 0 {
		first = lines[0]
	}
	if first != "# "+r.Memory.Title {
		add("MARKDOWN_TITLE", "Markdown H1 must exactly equal JSON title")
	}
	if m := markdownIDRE.FindStringSubmatch(text); len(m) != 2 || m[1] != r.Memory.ID {
		add("MARKDOWN_UUID", "Markdown Memory ID must equal JSON id")
	}
	if m := markdownTypeRE.FindStringSubmatch(text); len(m) != 2 || m[1] != r.Memory.Type {
		add("MARKDOWN_TYPE", "Markdown Type must equal JSON type")
	}
	if strings.Contains(text, "<!--") || strings.Contains(text, "-->") {
		add("TEMPLATE_SCAFFOLD", "finalized memory contains instructional HTML comments")
	}
	for _, marker := range templateMarkers {
		if strings.Contains(text, marker) {
			add("TEMPLATE_SCAFFOLD", "finalized memory contains template placeholder: "+marker)
			break
		}
	}
	if r.Memory.Type == "open_loop" && r.Memory.OpenLoopStatus != nil {
		checkOpenLoopForm(*r.Memory.OpenLoopStatus, text, add)
	}
}

func checkOpenLoopForm(status, text string, add func(string, string)) {
	headings := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			headings[strings.TrimSpace(line)] = true
		}
	}
	require := func(names ...string) {
		for _, h := range names {
			if !headings[h] {
				add("OPEN_LOOP_MARKDOWN_FORM", fmt.Sprintf("status %q requires heading %q", status, h))
			}
	}
	forbid := func(names ...string) {
		for _, h := range names {
			if headings[h] {
				add("OPEN_LOOP_MARKDOWN_FORM", fmt.Sprintf("status %q must not use heading %q", status, h))
			}
		}
	}
	switch status {
	case "open", "blocked", "deferred":
		require("## Open question or task", "## Why it remains open", "## Next useful action")
		forbid("## Original question or task", "## Outcome", "## Closure basis")
	case "resolved", "cancelled":
		require("## Original question or task", "## Outcome", "## Closure basis")
		forbid("## Open question or task", "## Why it remains open", "## Next useful action")
	}
}

func checkTemporal(root string, records []memory.Record, issues *[]Issue) {
	for _, r := range records {
		created, e1 := time.Parse(time.RFC3339, r.Memory.Temporal.CreatedAt)
		updated, e2 := time.Parse(time.RFC3339, r.Memory.Temporal.UpdatedAt)
		if e1 == nil && e2 == nil && updated.Before(created) {
			*issues = append(*issues, Issue{Severity: "ERROR", Code: "UPDATED_BEFORE_CREATED", Path: rel(root, r.Path), Message: "updated_at is earlier than created_at"})
		}
		if r.Memory.Temporal.EffectiveFrom != nil && r.Memory.Temporal.EffectiveUntil != nil {
			from, e1 := time.Parse("2006-01-02", *r.Memory.Temporal.EffectiveFrom)
			until, e2 := time.Parse("2006-01-02", *r.Memory.Temporal.EffectiveUntil)
			if e1 == nil && e2 == nil && until.Before(from) {
				*issues = append(*issues, Issue{Severity: "ERROR", Code: "EFFECTIVE_RANGE_REVERSED", Path: rel(root, r.Path), Message: "effective_until is earlier than effective_from"})
			}
		}
	}
}

func checkSupersessionCycles(root string, records []memory.Record, byID map[string][]memory.Record, issues *[]Issue) {
	graph := map[string][]string{}
	pathByID := map[string]string{}
	for _, r := range records {
		if len(byID[r.Memory.ID]) != 1 {
			continue
		}
		pathByID[r.Memory.ID] = r.Path
		for _, edge := range r.Memory.Relationships {
			if edge.Type == "supersedes" && len(byID[edge.TargetID]) == 1 {
				graph[r.Memory.ID] = append(graph[r.Memory.ID], edge.TargetID)
			}
		}
	}
	state := map[string]int{}
	stack := []string{}
	reported := map[string]bool{}
	var dfs func(string)
	dfs = func(node string) {
		state[node] = 1
		stack = append(stack, node)
		for _, target := range graph[node] {
			if state[target] == 0 {
				dfs(target)
			} else if state[target] == 1 {
				start := 0
				for i, v := range stack {
					if v == target {
						start = i
						break
					}
				}
				cycle := append(append([]string{}, stack[start:]...), target)
				key := strings.Join(cycle, "->")
				if !reported[key] {
					reported[key] = true
					*issues = append(*issues, Issue{Severity: "ERROR", Code: "SUPERSESSION_CYCLE", Path: rel(root, pathByID[node]), Message: "supersession cycle: " + strings.Join(cycle, " -> ")})
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[node] = 2
	}
	nodes := make([]string, 0, len(graph))
	for n := range graph {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	for _, n := range nodes {
		if state[n] == 0 {
			dfs(n)
		}
	}
}

func MarshalJSONReport(root string, issues []Issue) ([]byte, error) {
	errors := 0
	warnings := 0
	for _, i := range issues {
		if i.Severity == "ERROR" {
			errors++
		} else if i.Severity == "WARNING" {
			warnings++
		}
	}
	payload := struct {
		Root     string  `json:"root"`
		Errors   int     `json:"errors"`
		Warnings int     `json:"warnings"`
		Issues   []Issue `json:"issues"`
	}{root, errors, warnings, issues}
	return json.MarshalIndent(payload, "", "  ")
}

func RenderText(issues []Issue) string {
	if len(issues) == 0 {
		return "Runethread validation passed: 0 errors, 0 warnings."
	}
	errors := 0
	warnings := 0
	for _, i := range issues {
		if i.Severity == "ERROR" {
			errors++
		} else if i.Severity == "WARNING" {
			warnings++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Runethread validation: %d error(s), %d warning(s).\n", errors, warnings)
	for _, i := range issues {
		fmt.Fprintf(&b, "%s [%s] %s: %s\n", i.Severity, i.Code, i.Path, i.Message)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func HasErrors(issues []Issue) bool {
	for _, i := range issues {
		if i.Severity == "ERROR" {
			return true
		}
	}
	return false
}

func sortIssues(issues []Issue) []Issue {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity < issues[j].Severity
		}
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		return issues[i].Message < issues[j].Message
	})
	return issues
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

func clean(path string) string {
	p, _ := filepath.Abs(path)
	return filepath.Clean(p)
}
