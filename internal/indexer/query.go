package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var safeIndexKeyRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var memoryIDRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type SearchResult struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Type         string   `json:"type"`
	Lifecycle    string   `json:"lifecycle"`
	Summary      string   `json:"summary"`
	ContentPath  string   `json:"content_path"`
	Score        int      `json:"score"`
	MatchedTerms []string `json:"matched_terms"`
}

func LookupByID(root, id string) (SearchResult, bool, error) {
	if err := ensureIndexUsable(root); err != nil {
		return SearchResult{}, false, err
	}
	entry, ok, err := lookupEntry(root, strings.ToLower(strings.TrimSpace(id)), nil)
	if err != nil || !ok {
		return SearchResult{}, ok, err
	}
	return resultFromEntry(entry, 0, nil), true, nil
}

func IDsForProject(root, project string) ([]string, error) {
	return idsForKey(root, "by-project", project)
}

func IDsForTopic(root, topic string) ([]string, error) {
	return idsForKey(root, "by-topic", topic)
}

func IDsForTag(root, tag string) ([]string, error) {
	return idsForKey(root, "by-tag", tag)
}

func IDsForType(root, memoryType string) ([]string, error) {
	return idsForKey(root, "by-type", memoryType)
}

func Search(root, query string, limit int) ([]SearchResult, error) {
	if err := ensureIndexUsable(root); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query must not be empty")
	}

	if memoryIDRE.MatchString(strings.ToLower(query)) {
		entry, ok, err := lookupEntry(root, strings.ToLower(query), nil)
		if err != nil {
			return nil, err
		}
		if !ok {
			return []SearchResult{}, nil
		}
		return []SearchResult{resultFromEntry(entry, 0, nil)}, nil
	}

	terms := uniqueTerms(tokenize(query))
	if len(terms) == 0 {
		return nil, fmt.Errorf("search query contains no indexable terms")
	}
	if limit <= 0 {
		limit = 10
	}

	type aggregate struct {
		score   int
		matched map[string]bool
	}
	aggregates := map[string]*aggregate{}
	termCache := map[string]map[string][]termPosting{}
	for _, term := range terms {
		bucket := termBucket(term)
		shard := termCache[bucket]
		if shard == nil {
			loaded, found, err := loadTermShard(root, bucket)
			if err != nil {
				return nil, err
			}
			if !found {
				termCache[bucket] = map[string][]termPosting{}
				continue
			}
			shard = loaded
			termCache[bucket] = shard
		}
		for _, posting := range shard[term] {
			a := aggregates[posting.ID]
			if a == nil {
				a = &aggregate{matched: map[string]bool{}}
				aggregates[posting.ID] = a
			}
			a.score += posting.Score
			a.matched[term] = true
		}
	}

	type ranked struct {
		id      string
		score   int
		matched []string
	}
	rankedIDs := make([]ranked, 0, len(aggregates))
	for id, a := range aggregates {
		matched := make([]string, 0, len(a.matched))
		for term := range a.matched {
			matched = append(matched, term)
		}
		sort.Strings(matched)
		rankedIDs = append(rankedIDs, ranked{id: id, score: a.score, matched: matched})
	}
	sort.Slice(rankedIDs, func(i, j int) bool {
		if len(rankedIDs[i].matched) != len(rankedIDs[j].matched) {
			return len(rankedIDs[i].matched) > len(rankedIDs[j].matched)
		}
		if rankedIDs[i].score != rankedIDs[j].score {
			return rankedIDs[i].score > rankedIDs[j].score
		}
		return rankedIDs[i].id < rankedIDs[j].id
	})
	if len(rankedIDs) > limit {
		rankedIDs = rankedIDs[:limit]
	}

	idCache := map[string]map[string]indexEntry{}
	results := make([]SearchResult, 0, len(rankedIDs))
	for _, rankedID := range rankedIDs {
		entry, ok, err := lookupEntry(root, rankedID.id, idCache)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("term index references missing memory UUID %s", rankedID.id)
		}
		results = append(results, resultFromEntry(entry, rankedID.score, rankedID.matched))
	}
	return results, nil
}

func ensureIndexUsable(root string) error {
	if IsMarkedStale(root) {
		return fmt.Errorf("GitMemo index is explicitly marked stale; regenerate it or fall back to canonical memories/repository search")
	}
	data, err := os.ReadFile(filepath.Join(root, "index", "catalog.json"))
	if err != nil {
		return fmt.Errorf("read index catalog: %w", err)
	}
	var cat catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return fmt.Errorf("parse index catalog: %w", err)
	}
	if cat.IndexVersion != IndexVersion {
		return fmt.Errorf("unsupported GitMemo index version %d; this binary supports %d", cat.IndexVersion, IndexVersion)
	}
	return nil
}

func idsForKey(root, directory, key string) ([]string, error) {
	if err := ensureIndexUsable(root); err != nil {
		return nil, err
	}
	key = strings.ToLower(strings.TrimSpace(key))
	if !safeIndexKeyRE.MatchString(key) {
		return nil, fmt.Errorf("invalid index key %q", key)
	}
	path := filepath.Join(root, "index", directory, key+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var list idList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse %s index %q: %w", directory, key, err)
	}
	return append([]string(nil), list.IDs...), nil
}

func lookupEntry(root, id string, cache map[string]map[string]indexEntry) (indexEntry, bool, error) {
	if !memoryIDRE.MatchString(id) {
		return indexEntry{}, false, fmt.Errorf("invalid memory UUID %q", id)
	}
	prefix, err := idShardPrefix(id)
	if err != nil {
		return indexEntry{}, false, err
	}
	var shard map[string]indexEntry
	if cache != nil {
		shard = cache[prefix]
	}
	if shard == nil {
		path := filepath.Join(root, "index", "by-id", prefix+".json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return indexEntry{}, false, nil
		}
		if err != nil {
			return indexEntry{}, false, err
		}
		if err := json.Unmarshal(data, &shard); err != nil {
			return indexEntry{}, false, fmt.Errorf("parse ID shard %s: %w", prefix, err)
		}
		if cache != nil {
			cache[prefix] = shard
		}
	}
	entry, ok := shard[id]
	return entry, ok, nil
}

func loadTermShard(root, bucket string) (map[string][]termPosting, bool, error) {
	path := filepath.Join(root, "index", "terms", bucket+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var shard map[string][]termPosting
	if err := json.Unmarshal(data, &shard); err != nil {
		return nil, false, fmt.Errorf("parse term shard %q: %w", bucket, err)
	}
	return shard, true, nil
}

func uniqueTerms(terms []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		if !seen[term] {
			seen[term] = true
			out = append(out, term)
		}
	}
	return out
}

func resultFromEntry(entry indexEntry, score int, matched []string) SearchResult {
	return SearchResult{
		ID: entry.ID, Title: entry.Title, Type: entry.Type, Lifecycle: entry.Lifecycle,
		Summary: entry.Summary, ContentPath: entry.ContentPath, Score: score,
		MatchedTerms: append([]string(nil), matched...),
	}
}
