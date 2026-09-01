package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Karageorgiou/GitMemo/internal/memory"
)

var IndexPaths = []string{
	"index/memories.jsonl",
	"index/projects.md",
	"index/open-loops.md",
	"index/preferences.md",
}

type indexRelationship struct {
	Type     string `json:"type"`
	TargetID string `json:"target_id"`
}

type indexEntry struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Type            string              `json:"type"`
	Lifecycle       string              `json:"lifecycle"`
	Summary         string              `json:"summary"`
	Projects        []string            `json:"projects"`
	Topics          []string            `json:"topics"`
	Tags            []string            `json:"tags"`
	Aliases         []string            `json:"aliases"`
	Entities        []memory.Entity     `json:"entities"`
	Importance      string              `json:"importance"`
	UpdatedAt       string              `json:"updated_at"`
	EffectiveFrom   *string             `json:"effective_from"`
	ProvenanceBasis string              `json:"provenance_basis"`
	Confidence      string              `json:"confidence"`
	Relationships   []indexRelationship `json:"relationships"`
	ContentPath     string              `json:"content_path"`
	Sensitivity     string              `json:"sensitivity"`
	OpenLoopStatus  *string             `json:"open_loop_status,omitempty"`
}

type Result struct {
	Files map[string][]byte
}

func Generate(root string) (Result, error) {
	records, err := memory.LoadAll(root)
	if err != nil {
		return Result{}, err
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Memory.ID < records[j].Memory.ID })

	files := map[string][]byte{}
	memories, err := renderMemoriesJSONL(records)
	if err != nil {
		return Result{}, err
	}
	files["index/memories.jsonl"] = memories
	files["index/projects.md"] = renderProjects(root)
	files["index/open-loops.md"] = renderOpenLoops(root, records)
	files["index/preferences.md"] = renderPreferences(root, records)
	return Result{Files: files}, nil
}

func Write(root string) error {
	result, err := Generate(root)
	if err != nil {
		return err
	}
	for _, rel := range IndexPaths {
		data := result.Files[rel]
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func Check(root string) ([]string, error) {
	result, err := Generate(root)
	if err != nil {
		return nil, err
	}
	var stale []string
	for _, rel := range IndexPaths {
		expected := result.Files[rel]
		actual, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil || !bytes.Equal(actual, expected) {
			stale = append(stale, rel)
		}
	}
	return stale, nil
}

func renderMemoriesJSONL(records []memory.Record) ([]byte, error) {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	for _, record := range records {
		m := record.Memory
		relationships := make([]indexRelationship, 0, len(m.Relationships))
		for _, r := range m.Relationships {
			relationships = append(relationships, indexRelationship{Type: r.Type, TargetID: r.TargetID})
		}
		entry := indexEntry{
			ID: m.ID, Title: m.Title, Type: m.Type, Lifecycle: m.Lifecycle, Summary: m.Summary,
			Projects: nonNil(m.Projects), Topics: nonNil(m.Topics), Tags: nonNil(m.Tags), Aliases: nonNil(m.Aliases),
			Entities: nonNilEntities(m.Entities), Importance: m.Importance, UpdatedAt: m.Temporal.UpdatedAt,
			EffectiveFrom: m.Temporal.EffectiveFrom, ProvenanceBasis: m.Provenance.Basis, Confidence: m.Provenance.Confidence,
			Relationships: nonNilRelationships(relationships), ContentPath: m.ContentPath, Sensitivity: m.Sensitivity,
			OpenLoopStatus: m.OpenLoopStatus,
		}
		if err := enc.Encode(entry); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func renderProjects(root string) []byte {
	base := filepath.Join(root, "projects")
	entries, _ := os.ReadDir(base)
	var slugs []string
	for _, entry := range entries {
		if entry.IsDir() {
			slugs = append(slugs, entry.Name())
		}
	}
	sort.Strings(slugs)
	var b strings.Builder
	b.WriteString("# Projects\n")
	if len(slugs) == 0 {
		b.WriteString("\nNo projects have been registered yet.\n")
		return []byte(b.String())
	}
	for _, slug := range slugs {
		b.WriteString("\n## " + projectDisplayName(root, slug) + "\n\n")
		b.WriteString("- Project slug: `" + slug + "`\n")
		for _, item := range []struct{ label, file string }{{"Overview", "overview.md"}, {"Current state", "current-state.md"}, {"Roadmap", "roadmap.md"}} {
			rel := filepath.ToSlash(filepath.Join("projects", slug, item.file))
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
				b.WriteString("- " + item.label + ": `" + rel + "`\n")
			}
		}
	}
	return []byte(b.String())
}

func renderOpenLoops(root string, records []memory.Record) []byte {
	groups := map[string][]memory.Memory{}
	for _, r := range records {
		m := r.Memory
		if m.Type != "open_loop" || m.Lifecycle != "active" || m.OpenLoopStatus == nil || !isUnresolved(*m.OpenLoopStatus) {
			continue
		}
		if len(m.Projects) == 0 {
			groups[""] = append(groups[""], m)
		} else {
			for _, p := range m.Projects {
				groups[p] = append(groups[p], m)
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Open Loops\n")
	if len(groups) == 0 {
		b.WriteString("\nNo unresolved open-loop memories.\n\nResolved/cancelled open-loop memories remain discoverable through `index/memories.jsonl` and atomic memory retrieval.\n")
		return []byte(b.String())
	}
	keys := sortedKeys(groups)
	for _, project := range keys {
		title := "General"
		if project != "" {
			title = projectDisplayName(root, project)
		}
		b.WriteString("\n## " + title + "\n\n")
		sort.Slice(groups[project], func(i, j int) bool { return groups[project][i].ID < groups[project][j].ID })
		for _, m := range groups[project] {
			b.WriteString(fmt.Sprintf("- `%s` — %s.\n  - Status: `%s`\n  - Memory: `%s`\n", m.ID, strings.TrimSuffix(m.Title, "."), *m.OpenLoopStatus, m.ContentPath))
		}
	}
	b.WriteString("\nResolved/cancelled open-loop memories are intentionally omitted from this active-work index and remain discoverable through `index/memories.jsonl` and atomic memory retrieval.\n")
	return []byte(b.String())
}

func renderPreferences(root string, records []memory.Record) []byte {
	groups := map[string][]memory.Memory{}
	for _, r := range records {
		m := r.Memory
		if m.Type != "preference" || m.Lifecycle != "active" {
			continue
		}
		if len(m.Projects) == 0 {
			groups[""] = append(groups[""], m)
		} else {
			for _, p := range m.Projects {
				groups[p] = append(groups[p], m)
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Preferences\n")
	if len(groups) == 0 {
		b.WriteString("\nNo atomic preference memories have been imported yet.\n\nThis index is intentionally empty rather than populated from unreviewed historical context.\n")
		return []byte(b.String())
	}
	for _, project := range sortedKeys(groups) {
		title := "General"
		if project != "" {
			title = projectDisplayName(root, project)
		}
		b.WriteString("\n## " + title + "\n\n")
		sort.Slice(groups[project], func(i, j int) bool { return groups[project][i].ID < groups[project][j].ID })
		for _, m := range groups[project] {
			b.WriteString(fmt.Sprintf("- `%s` — %s.\n  - Memory: `%s`\n", m.ID, strings.TrimSuffix(m.Title, "."), m.ContentPath))
		}
	}
	return []byte(b.String())
}

func projectDisplayName(root, slug string) string {
	data, err := os.ReadFile(filepath.Join(root, "projects", slug, "overview.md"))
	if err == nil {
		lines := strings.Split(string(data), "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
			title := strings.TrimPrefix(lines[0], "# ")
			if i := strings.Index(title, " —"); i >= 0 {
				title = title[:i]
			}
			if i := strings.Index(title, " - "); i >= 0 {
				title = title[:i]
			}
			if strings.TrimSpace(title) != "" {
				return strings.TrimSpace(title)
			}
		}
	}
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func sortedKeys[T any](m map[string][]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func isUnresolved(status string) bool {
	return status == "open" || status == "blocked" || status == "deferred"
}
func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
func nonNilEntities(v []memory.Entity) []memory.Entity {
	if v == nil {
		return []memory.Entity{}
	}
	return v
}
func nonNilRelationships(v []indexRelationship) []indexRelationship {
	if v == nil {
		return []indexRelationship{}
	}
	return v
}
