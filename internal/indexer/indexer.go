package indexer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/fsafety"
	"github.com/runethread/core/internal/memory"
)

const IndexVersion = buildinfo.IndexFormatVersion

const StaleMarkerPath = "index/STALE"

var HumanIndexPaths = []string{
	"index/projects.md",
	"index/open-loops.md",
	"index/preferences.md",
}

const staleMarkerText = "Runethread generated indexes are stale.\nCanonical memories and project files remain authoritative.\nRegenerate with: runethread index --write .\n"

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
	if buildinfo.ContractVersion >= 8 {
		if err := requireOptionalRegularTree(root, "projects"); err != nil {
			return Result{}, fmt.Errorf("unsafe projects tree: %w", err)
		}
	}
	var records []memory.Record
	var err error
	if buildinfo.ContractVersion >= 8 {
		records, err = memory.LoadAllStrict(root)
	} else {
		records, err = memory.LoadAll(root)
	}
	if err != nil {
		return Result{}, err
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Memory.ID != records[j].Memory.ID {
			return records[i].Memory.ID < records[j].Memory.ID
		}
		return records[i].Path < records[j].Path
	})

	files, err := renderMachineIndexes(root, records)
	if err != nil {
		return Result{}, err
	}
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
	return replaceIndexDirectory(root, result)
}

func Check(root string) ([]string, error) {
	result, err := Generate(root)
	if err != nil {
		return nil, err
	}
	actualPaths, err := ExistingPaths(root)
	if err != nil {
		return nil, err
	}

	actualSet := make(map[string]bool, len(actualPaths))
	for _, rel := range actualPaths {
		actualSet[rel] = true
	}

	stale := make([]string, 0)
	for rel, expected := range result.Files {
		actual, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil || !bytes.Equal(actual, expected) {
			stale = append(stale, rel)
		}
		delete(actualSet, rel)
	}
	for rel := range actualSet {
		stale = append(stale, rel)
	}
	sort.Strings(stale)
	return uniqueStrings(stale), nil
}

func ExistingPaths(root string) ([]string, error) {
	base := filepath.Join(root, "index")
	if _, err := os.Lstat(base); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if buildinfo.ContractVersion >= 8 {
		if err := requireOptionalRegularTree(root, "index"); err != nil {
			return nil, err
		}
	}

	var paths []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func GeneratedPaths(root string) ([]string, error) {
	result, err := Generate(root)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(result.Files))
	for rel := range result.Files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	return paths, nil
}

func MarkStale(root string) error {
	path := filepath.Join(root, filepath.FromSlash(StaleMarkerPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(staleMarkerText), 0o644)
}

func IsMarkedStale(root string) bool {
	if buildinfo.ContractVersion >= 8 {
		if _, err := ExistingPaths(root); err != nil {
			return true
		}
		_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(StaleMarkerPath)))
		return err == nil
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(StaleMarkerPath)))
	return err == nil
}

func replaceIndexDirectory(root string, result Result) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	newDir, err := os.MkdirTemp(root, ".runethread-index-new-*")
	if err != nil {
		return fmt.Errorf("create temporary index directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(newDir) }()

	paths := make([]string, 0, len(result.Files))
	for rel := range result.Files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		if !strings.HasPrefix(rel, "index/") {
			return fmt.Errorf("generated index path escapes index directory: %s", rel)
		}
		inside := strings.TrimPrefix(rel, "index/")
		path := filepath.Join(newDir, filepath.FromSlash(inside))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create parent for %s: %w", rel, err)
		}
		if err := os.WriteFile(path, result.Files[rel], 0o644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}

	indexDir := filepath.Join(root, "index")
	backupDir, err := os.MkdirTemp(root, ".runethread-index-old-*")
	if err != nil {
		return fmt.Errorf("reserve backup index path: %w", err)
	}
	if err := os.Remove(backupDir); err != nil {
		return fmt.Errorf("prepare backup index path: %w", err)
	}

	hadOld := false
	if _, err := os.Lstat(indexDir); err == nil {
		if err := os.Rename(indexDir, backupDir); err != nil {
			return fmt.Errorf("move old index aside: %w", err)
		}
		hadOld = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect old index: %w", err)
	}

	if err := os.Rename(newDir, indexDir); err != nil {
		if hadOld {
			_ = os.Rename(backupDir, indexDir)
		}
		return fmt.Errorf("activate regenerated index: %w", err)
	}
	newDir = ""
	if hadOld {
		_ = os.RemoveAll(backupDir)
	}
	return nil
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
		b.WriteString("\nNo unresolved open-loop memories.\n\nResolved/cancelled open-loop memories remain discoverable through the sharded machine indexes and atomic memory retrieval.\n")
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
	b.WriteString("\nResolved/cancelled open-loop memories are intentionally omitted from this active-work index and remain discoverable through the sharded machine indexes and atomic memory retrieval.\n")
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

func requireOptionalRegularTree(root, rel string) error {
	base := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Lstat(base); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return fsafety.RequireTree(root, rel)
}

func sortedKeys[T any](m map[string][]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:0]
	var previous string
	for i, value := range values {
		if i == 0 || value != previous {
			out = append(out, value)
		}
		previous = value
	}
	return out
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
