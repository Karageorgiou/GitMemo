package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/Karageorgiou/GitMemo/internal/memory"
)

const termShardHashHexCharacters = 3

const idShardHashHexCharacters = 3

type catalogLayout struct {
	ByID             string `json:"by_id"`
	ByProject        string `json:"by_project"`
	ByTopic          string `json:"by_topic"`
	ByTag            string `json:"by_tag"`
	ByType           string `json:"by_type"`
	ByLifecycle      string `json:"by_lifecycle"`
	ByOpenLoopStatus string `json:"by_open_loop_status"`
	Terms            string `json:"terms"`
	TermPostings     string `json:"term_postings"`
}

type catalog struct {
	IndexVersion               int           `json:"index_version"`
	RecordCount                int           `json:"record_count"`
	MemorySourceSHA256         string        `json:"memory_source_sha256"`
	IDShardHashHexCharacters   int           `json:"id_shard_hash_hex_characters"`
	TermShardHashHexCharacters int           `json:"term_shard_hash_hex_characters"`
	IDListChunkSize            int           `json:"id_list_chunk_size"`
	TermPostingChunkSize       int           `json:"term_posting_chunk_size"`
	MaxStoredTermPostings      int           `json:"max_stored_term_postings"`
	TermFields                 []string      `json:"term_fields"`
	Layout                     catalogLayout `json:"layout"`
}

type termPosting struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
}

func renderMachineIndexes(root string, records []memory.Record) (map[string][]byte, error) {
	digest, err := memorySourceDigest(root, records)
	if err != nil {
		return nil, err
	}
	return renderMachineIndexesWithDigest(records, digest)
}

func renderMachineIndexesWithDigest(records []memory.Record, digest string) (map[string][]byte, error) {
	files := map[string][]byte{}
	byID := map[string]map[string]indexEntry{}
	byProject := map[string][]string{}
	byTopic := map[string][]string{}
	byTag := map[string][]string{}
	byType := map[string][]string{}
	byLifecycle := map[string][]string{}
	byOpenLoopStatus := map[string][]string{}
	termShards := map[string]map[string][]termPosting{}

	for _, record := range records {
		m := record.Memory
		entry := makeIndexEntry(m)
		prefix, err := idShardPrefix(m.ID)
		if err != nil {
			return nil, err
		}
		if byID[prefix] == nil {
			byID[prefix] = map[string]indexEntry{}
		}
		byID[prefix][m.ID] = entry

		appendIndexKeys(byProject, m.Projects, m.ID)
		appendIndexKeys(byTopic, m.Topics, m.ID)
		appendIndexKeys(byTag, m.Tags, m.ID)
		byType[m.Type] = append(byType[m.Type], m.ID)
		byLifecycle[m.Lifecycle] = append(byLifecycle[m.Lifecycle], m.ID)
		if m.OpenLoopStatus != nil {
			byOpenLoopStatus[*m.OpenLoopStatus] = append(byOpenLoopStatus[*m.OpenLoopStatus], m.ID)
		}

		for term, score := range weightedTerms(m) {
			bucket := termBucket(term)
			if termShards[bucket] == nil {
				termShards[bucket] = map[string][]termPosting{}
			}
			termShards[bucket][term] = append(termShards[bucket][term], termPosting{ID: m.ID, Score: score})
		}
	}

	for prefix, entries := range byID {
		data, err := marshalLine(entries)
		if err != nil {
			return nil, err
		}
		path, err := idShardPath(prefix)
		if err != nil {
			return nil, err
		}
		files[path] = data
	}
	if err := renderIDLists(files, "index/by-project", byProject); err != nil {
		return nil, err
	}
	if err := renderIDLists(files, "index/by-topic", byTopic); err != nil {
		return nil, err
	}
	if err := renderIDLists(files, "index/by-tag", byTag); err != nil {
		return nil, err
	}
	if err := renderIDLists(files, "index/by-type", byType); err != nil {
		return nil, err
	}
	if err := renderIDLists(files, "index/by-lifecycle", byLifecycle); err != nil {
		return nil, err
	}
	if err := renderIDLists(files, "index/by-open-loop-status", byOpenLoopStatus); err != nil {
		return nil, err
	}

	for bucket, terms := range termShards {
		rendered := make(map[string]termIndexValue, len(terms))
		for term := range terms {
			sort.Slice(terms[term], func(i, j int) bool {
				if terms[term][i].Score != terms[term][j].Score {
					return terms[term][i].Score > terms[term][j].Score
				}
				return terms[term][i].ID < terms[term][j].ID
			})
			value, err := renderTermIndexValue(files, term, terms[term])
			if err != nil {
				return nil, err
			}
			rendered[term] = value
		}
		data, err := marshalLine(rendered)
		if err != nil {
			return nil, err
		}
		files["index/terms/"+bucket+".json"] = data
	}

	cat := catalog{
		IndexVersion:               IndexVersion,
		RecordCount:                len(records),
		MemorySourceSHA256:         digest,
		IDShardHashHexCharacters:   idShardHashHexCharacters,
		TermShardHashHexCharacters: termShardHashHexCharacters,
		IDListChunkSize:            idListChunkSize,
		TermPostingChunkSize:       termPostingChunkSize,
		MaxStoredTermPostings:      maxStoredTermPostings,
		TermFields: []string{
			"title", "aliases", "projects", "topics", "tags", "entities", "type", "lifecycle", "open_loop_status", "summary",
		},
		Layout: catalogLayout{
			ByID:             "index/by-id/<first-2-sha256-hex>/<third-sha256-hex>.json",
			ByProject:        "index/by-project/<project-slug>.json",
			ByTopic:          "index/by-topic/<topic-slug>.json",
			ByTag:            "index/by-tag/<tag-slug>.json",
			ByType:           "index/by-type/<memory-type>.json",
			ByLifecycle:      "index/by-lifecycle/<lifecycle>.json",
			ByOpenLoopStatus: "index/by-open-loop-status/<status>.json",
			Terms:            "index/terms/<sha256-term-prefix>.json",
			TermPostings:     "index/term-postings/<sha256-prefix>/<full-sha256>/<chunk>.json",
		},
	}
	catalogData, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return nil, err
	}
	files["index/catalog.json"] = append(catalogData, '\n')
	return files, nil
}

func makeIndexEntry(m memory.Memory) indexEntry {
	relationships := make([]indexRelationship, 0, len(m.Relationships))
	for _, r := range m.Relationships {
		relationships = append(relationships, indexRelationship{Type: r.Type, TargetID: r.TargetID})
	}
	return indexEntry{
		ID: m.ID, Title: m.Title, Type: m.Type, Lifecycle: m.Lifecycle, Summary: m.Summary,
		Projects: nonNil(m.Projects), Topics: nonNil(m.Topics), Tags: nonNil(m.Tags), Aliases: nonNil(m.Aliases),
		Entities: nonNilEntities(m.Entities), Importance: m.Importance, UpdatedAt: m.Temporal.UpdatedAt,
		EffectiveFrom: m.Temporal.EffectiveFrom, ProvenanceBasis: m.Provenance.Basis, Confidence: m.Provenance.Confidence,
		Relationships: nonNilRelationships(relationships), ContentPath: m.ContentPath, Sensitivity: m.Sensitivity,
		OpenLoopStatus: m.OpenLoopStatus,
	}
}

func renderIDLists(files map[string][]byte, dir string, values map[string][]string) error {
	for key, ids := range values {
		sort.Strings(ids)
		if err := renderChunkedIDList(files, dir, key, ids); err != nil {
			return err
		}
	}
	return nil
}

func appendIndexKeys(index map[string][]string, keys []string, id string) {
	for _, key := range keys {
		index[key] = append(index[key], id)
	}
}

func weightedTerms(m memory.Memory) map[string]int {
	scores := map[string]int{}
	addWeightedGroup(scores, []string{m.Title}, 8)
	addWeightedGroup(scores, m.Aliases, 6)
	addWeightedGroup(scores, m.Projects, 5)
	addWeightedGroup(scores, m.Topics, 5)
	addWeightedGroup(scores, m.Tags, 5)

	entityTexts := make([]string, 0, len(m.Entities)*2)
	for _, entity := range m.Entities {
		entityTexts = append(entityTexts, entity.Kind, entity.Name)
	}
	addWeightedGroup(scores, entityTexts, 4)

	classification := []string{m.Type, m.Lifecycle}
	if m.OpenLoopStatus != nil {
		classification = append(classification, *m.OpenLoopStatus)
	}
	addWeightedGroup(scores, classification, 3)
	addWeightedGroup(scores, []string{m.Summary}, 2)
	return scores
}

func addWeightedGroup(scores map[string]int, texts []string, weight int) {
	seen := map[string]bool{}
	for _, text := range texts {
		for _, term := range tokenize(text) {
			seen[term] = true
		}
	}
	for term := range seen {
		scores[term] += weight
	}
}

func tokenize(text string) []string {
	var terms []string
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		terms = append(terms, string(current))
		current = current[:0]
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, unicode.ToLower(r))
		} else {
			flush()
		}
	}
	flush()
	return terms
}

func termBucket(term string) string {
	return termHash(term)[:termShardHashHexCharacters]
}

func idShardPrefix(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", fmt.Errorf("memory UUID is empty")
	}
	hash := termHash(id)
	return hash[:idShardHashHexCharacters], nil
}

func idShardPath(prefix string) (string, error) {
	if len(prefix) != idShardHashHexCharacters {
		return "", fmt.Errorf("invalid ID shard hash prefix length %d: %q", len(prefix), prefix)
	}
	return filepath.ToSlash(filepath.Join("index", "by-id", prefix[:2], prefix[2:]+".json")), nil
}

func memorySourceDigest(root string, records []memory.Record) (string, error) {
	h := sha256.New()
	for _, record := range records {
		rel, err := filepath.Rel(root, record.Path)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(record.Path)
		if err != nil {
			return "", err
		}
		_, _ = h.Write([]byte(filepath.ToSlash(rel)))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func marshalLine(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
