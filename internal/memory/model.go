package memory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/runethread/core/internal/fsafety"
)

const ExpectedSchemaContractSHA256 = "064ad5ef7fcdaacd086a0bf21ae4de6b62006498f6a451a2fcf7b9bcbdb05f5b"

var (
	uuidV4RE      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	slugRE        = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	contentPathRE = regexp.MustCompile(`^memories(?:/[a-z0-9]+(?:-[a-z0-9]+)*)+/[a-z0-9]+(?:-[a-z0-9]+)*--[0-9a-f]{8}\.md$`)
)

type Entity struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type Temporal struct {
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	EffectiveFrom  *string `json:"effective_from"`
	EffectiveUntil *string `json:"effective_until"`
}

type Source struct {
	Kind     string  `json:"kind"`
	Locator  string  `json:"locator"`
	Revision *string `json:"revision"`
	Note     *string `json:"note"`
}

type Provenance struct {
	Basis                 string   `json:"basis"`
	Confidence            string   `json:"confidence"`
	ExplicitMemoryRequest bool     `json:"explicit_memory_request"`
	Sources               []Source `json:"sources"`
}

type Relationship struct {
	Type     string  `json:"type"`
	TargetID string  `json:"target_id"`
	Note     *string `json:"note"`
}

type Memory struct {
	SchemaVersion  int            `json:"schema_version"`
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Type           string         `json:"type"`
	Lifecycle      string         `json:"lifecycle"`
	Summary        string         `json:"summary"`
	Projects       []string       `json:"projects"`
	Topics         []string       `json:"topics"`
	Tags           []string       `json:"tags"`
	Aliases        []string       `json:"aliases"`
	Entities       []Entity       `json:"entities"`
	Importance     string         `json:"importance"`
	Temporal       Temporal       `json:"temporal"`
	Provenance     Provenance     `json:"provenance"`
	Relationships  []Relationship `json:"relationships"`
	ContentPath    string         `json:"content_path"`
	Sensitivity    string         `json:"sensitivity"`
	OpenLoopStatus *string        `json:"open_loop_status,omitempty"`
}

type Record struct {
	Path   string
	Memory Memory
}

type SchemaProblem struct {
	Field   string
	Message string
}

func (p SchemaProblem) Error() string {
	if p.Field == "" {
		return p.Message
	}
	return p.Field + ": " + p.Message
}

func ValidateSchemaContract(root string) error {
	data, err := fsafety.ReadRegularFileUnder(root, "schema/memory-item.schema.json")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	hash, err := canonicalJSONSHA256(data)
	if err != nil {
		return fmt.Errorf("parse schema JSON: %w", err)
	}
	if hash != ExpectedSchemaContractSHA256 {
		return fmt.Errorf("schema contract hash is %s, expected %s; review and update Go validation rules with the schema", hash, ExpectedSchemaContractSHA256)
	}
	return nil
}

func canonicalJSONSHA256(data []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return "", err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return "", errors.New("multiple JSON values")
		}
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func Discover(root string) (sidecars []string, markdown []string, err error) {
	base := filepath.Join(root, "memories")
	if _, statErr := os.Lstat(base); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, nil, nil
		}
		return nil, nil, statErr
	}
	if err := fsafety.RequireDirectory(base); err != nil {
		return nil, nil, fmt.Errorf("unsafe memories directory: %w", err)
	}

	err = filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symbolic link", path)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json":
			sidecars = append(sidecars, path)
		case ".md":
			markdown = append(markdown, path)
		}
		return nil
	})
	sort.Strings(sidecars)
	sort.Strings(markdown)
	return sidecars, markdown, err
}

func Load(path string) (Memory, []SchemaProblem) {
	data, err := fsafety.ReadRegularFile(path)
	if err != nil {
		return Memory{}, []SchemaProblem{{Message: err.Error()}}
	}
	return Decode(data)
}

func LoadAll(root string) ([]Record, error) {
	sidecars, _, err := Discover(root)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(sidecars))
	for _, path := range sidecars {
		m, problems := Load(path)
		if len(problems) > 0 {
			return nil, fmt.Errorf("%s: %s", relative(root, path), problems[0].Error())
		}
		records = append(records, Record{Path: path, Memory: m})
	}
	return records, nil
}

func Decode(data []byte) (Memory, []SchemaProblem) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Memory{}, []SchemaProblem{{Message: "invalid JSON object: " + err.Error()}}
	}
	if raw == nil {
		return Memory{}, []SchemaProblem{{Message: "sidecar root must be a JSON object"}}
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m Memory
	if err := dec.Decode(&m); err != nil {
		return Memory{}, []SchemaProblem{{Message: "strict decode failed: " + err.Error()}}
	}
	if err := consumeEOF(dec); err != nil {
		return Memory{}, []SchemaProblem{{Message: err.Error()}}
	}

	problems := validateRequired(raw)
	problems = append(problems, validateFields(m)...)
	return m, problems
}

func consumeEOF(dec *json.Decoder) error {
	var extra any
	err := dec.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}

func validateRequired(root map[string]json.RawMessage) []SchemaProblem {
	var out []SchemaProblem
	require := func(obj map[string]json.RawMessage, prefix string, names ...string) {
		for _, name := range names {
			raw, ok := obj[name]
			if !ok {
				out = append(out, SchemaProblem{Field: prefix + name, Message: "required field is missing"})
				continue
			}
			if string(bytes.TrimSpace(raw)) == "null" && name != "effective_from" && name != "effective_until" && name != "revision" && name != "note" {
				out = append(out, SchemaProblem{Field: prefix + name, Message: "must not be null"})
			}
		}
	}
	require(root, "", "schema_version", "id", "title", "type", "lifecycle", "summary", "projects", "topics", "tags", "aliases", "entities", "importance", "temporal", "provenance", "relationships", "content_path", "sensitivity")

	if raw, ok := root["temporal"]; ok && string(bytes.TrimSpace(raw)) != "null" {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) == nil {
			require(obj, "temporal.", "created_at", "updated_at", "effective_from", "effective_until")
		}
	}
	if raw, ok := root["provenance"]; ok && string(bytes.TrimSpace(raw)) != "null" {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) == nil {
			require(obj, "provenance.", "basis", "confidence", "explicit_memory_request", "sources")
			if sourcesRaw, ok := obj["sources"]; ok && string(bytes.TrimSpace(sourcesRaw)) != "null" {
				var items []map[string]json.RawMessage
				if json.Unmarshal(sourcesRaw, &items) == nil {
					for i, item := range items {
						require(item, fmt.Sprintf("provenance.sources[%d].", i), "kind", "locator", "revision", "note")
					}
				}
			}
		}
	}
	if raw, ok := root["relationships"]; ok && string(bytes.TrimSpace(raw)) != "null" {
		var items []map[string]json.RawMessage
		if json.Unmarshal(raw, &items) == nil {
			for i, item := range items {
				require(item, fmt.Sprintf("relationships[%d].", i), "type", "target_id", "note")
			}
		}
	}
	if raw, ok := root["entities"]; ok && string(bytes.TrimSpace(raw)) != "null" {
		var items []map[string]json.RawMessage
		if json.Unmarshal(raw, &items) == nil {
			for i, item := range items {
				require(item, fmt.Sprintf("entities[%d].", i), "kind", "name")
			}
		}
	}
	return out
}

func validateFields(m Memory) []SchemaProblem {
	var out []SchemaProblem
	add := func(field, message string) { out = append(out, SchemaProblem{Field: field, Message: message}) }

	if m.SchemaVersion != 1 {
		add("schema_version", "must equal 1")
	}
	if !uuidV4RE.MatchString(m.ID) {
		add("id", "must be a lowercase canonical UUIDv4")
	}
	checkString(&out, "title", m.Title, 1, 200)
	checkEnum(&out, "type", m.Type, "fact", "preference", "decision", "state", "open_loop", "correction", "milestone", "reference")
	checkEnum(&out, "lifecycle", m.Lifecycle, "active", "superseded", "withdrawn")
	checkString(&out, "summary", m.Summary, 1, 1000)
	checkSlugArray(&out, "projects", m.Projects, 20)
	checkSlugArray(&out, "topics", m.Topics, 30)
	checkSlugArray(&out, "tags", m.Tags, 50)
	checkStringArray(&out, "aliases", m.Aliases, 50, 1, 200)
	if len(m.Entities) > 50 {
		add("entities", "must contain at most 50 items")
	}
	seenEntities := map[Entity]bool{}
	for i, e := range m.Entities {
		if !slugRE.MatchString(e.Kind) || utf8.RuneCountInString(e.Kind) > 100 {
			add(fmt.Sprintf("entities[%d].kind", i), "must be a canonical kebab-case slug up to 100 characters")
		}
		checkString(&out, fmt.Sprintf("entities[%d].name", i), e.Name, 1, 200)
		if seenEntities[e] {
			add("entities", "must not contain duplicate items")
		}
		seenEntities[e] = true
	}
	checkEnum(&out, "importance", m.Importance, "normal", "high", "critical")
	checkDateTime(&out, "temporal.created_at", m.Temporal.CreatedAt)
	checkDateTime(&out, "temporal.updated_at", m.Temporal.UpdatedAt)
	if m.Temporal.EffectiveFrom != nil {
		checkDate(&out, "temporal.effective_from", *m.Temporal.EffectiveFrom)
	}
	if m.Temporal.EffectiveUntil != nil {
		checkDate(&out, "temporal.effective_until", *m.Temporal.EffectiveUntil)
	}
	checkEnum(&out, "provenance.basis", m.Provenance.Basis, "user_stated", "project_verified", "external_verified", "derived", "inferred", "migrated")
	checkEnum(&out, "provenance.confidence", m.Provenance.Confidence, "high", "medium", "low")
	if len(m.Provenance.Sources) < 1 || len(m.Provenance.Sources) > 50 {
		add("provenance.sources", "must contain 1 to 50 items")
	}
	for i, s := range m.Provenance.Sources {
		checkEnum(&out, fmt.Sprintf("provenance.sources[%d].kind", i), s.Kind, "conversation", "project_repository", "external", "file", "migration", "other")
		checkString(&out, fmt.Sprintf("provenance.sources[%d].locator", i), s.Locator, 1, 1000)
		checkOptionalString(&out, fmt.Sprintf("provenance.sources[%d].revision", i), s.Revision, 1, 200)
		checkOptionalString(&out, fmt.Sprintf("provenance.sources[%d].note", i), s.Note, 1, 1000)
	}
	if len(m.Relationships) > 100 {
		add("relationships", "must contain at most 100 items")
	}
	for i, r := range m.Relationships {
		checkEnum(&out, fmt.Sprintf("relationships[%d].type", i), r.Type, "related_to", "depends_on", "supersedes", "corrects", "conflicts_with")
		if !uuidV4RE.MatchString(r.TargetID) {
			add(fmt.Sprintf("relationships[%d].target_id", i), "must be a lowercase canonical UUIDv4")
		}
		checkOptionalString(&out, fmt.Sprintf("relationships[%d].note", i), r.Note, 1, 1000)
	}
	if !contentPathRE.MatchString(m.ContentPath) {
		add("content_path", "does not match the canonical memory Markdown path pattern")
	}
	checkEnum(&out, "sensitivity", m.Sensitivity, "routine", "private", "sensitive")
	if m.Type == "open_loop" {
		if m.OpenLoopStatus == nil {
			add("open_loop_status", "is required for open_loop memories")
		} else {
			checkEnum(&out, "open_loop_status", *m.OpenLoopStatus, "open", "blocked", "deferred", "resolved", "cancelled")
		}
	} else if m.OpenLoopStatus != nil {
		add("open_loop_status", "is only allowed for open_loop memories")
	}
	return out
}

func checkString(out *[]SchemaProblem, field, value string, min, max int) {
	n := utf8.RuneCountInString(value)
	if n < min || n > max {
		*out = append(*out, SchemaProblem{Field: field, Message: fmt.Sprintf("length must be between %d and %d", min, max)})
	}
}

func checkOptionalString(out *[]SchemaProblem, field string, value *string, min, max int) {
	if value != nil {
		checkString(out, field, *value, min, max)
	}
}

func checkEnum(out *[]SchemaProblem, field, value string, allowed ...string) {
	for _, candidate := range allowed {
		if value == candidate {
			return
		}
	}
	*out = append(*out, SchemaProblem{Field: field, Message: "invalid value " + fmt.Sprintf("%q", value)})
}

func checkSlugArray(out *[]SchemaProblem, field string, values []string, max int) {
	if len(values) > max {
		*out = append(*out, SchemaProblem{Field: field, Message: fmt.Sprintf("must contain at most %d items", max)})
	}
	seen := map[string]bool{}
	for i, value := range values {
		if !slugRE.MatchString(value) || utf8.RuneCountInString(value) > 100 {
			*out = append(*out, SchemaProblem{Field: fmt.Sprintf("%s[%d]", field, i), Message: "must be a canonical kebab-case slug up to 100 characters"})
		}
		if seen[value] {
			*out = append(*out, SchemaProblem{Field: field, Message: "must not contain duplicate items"})
		}
		seen[value] = true
	}
}

func checkStringArray(out *[]SchemaProblem, field string, values []string, max, minLen, maxLen int) {
	if len(values) > max {
		*out = append(*out, SchemaProblem{Field: field, Message: fmt.Sprintf("must contain at most %d items", max)})
	}
	seen := map[string]bool{}
	for i, value := range values {
		checkString(out, fmt.Sprintf("%s[%d]", field, i), value, minLen, maxLen)
		if seen[value] {
			*out = append(*out, SchemaProblem{Field: field, Message: "must not contain duplicate items"})
		}
		seen[value] = true
	}
}

func checkDateTime(out *[]SchemaProblem, field, value string) {
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		*out = append(*out, SchemaProblem{Field: field, Message: "must be an RFC3339 date-time"})
	}
}

func checkDate(out *[]SchemaProblem, field, value string) {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		*out = append(*out, SchemaProblem{Field: field, Message: "must be an ISO date (YYYY-MM-DD)"})
	}
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
