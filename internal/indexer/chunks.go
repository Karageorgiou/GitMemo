package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	idListChunkSize       = 1024
	termPostingChunkSize  = 1024
	maxStoredTermPostings = 32768
)

type idList struct {
	Count      int      `json:"count"`
	IDs        []string `json:"ids,omitempty"`
	ChunkSize  int      `json:"chunk_size,omitempty"`
	ChunkCount int      `json:"chunk_count,omitempty"`
}

type idChunk struct {
	IDs []string `json:"ids"`
}

type termIndexValue struct {
	DocumentFrequency int           `json:"document_frequency"`
	Suppressed        bool          `json:"suppressed,omitempty"`
	Postings          []termPosting `json:"postings,omitempty"`
	ChunkSize         int           `json:"chunk_size,omitempty"`
	ChunkCount        int           `json:"chunk_count,omitempty"`
}

type termPostingChunk struct {
	Postings []termPosting `json:"postings"`
}

func renderChunkedIDList(files map[string][]byte, dir, key string, ids []string) error {
	descriptor := idList{Count: len(ids)}
	if len(ids) <= idListChunkSize {
		descriptor.IDs = append([]string(nil), ids...)
	} else {
		descriptor.ChunkSize = idListChunkSize
		descriptor.ChunkCount = chunkCount(len(ids), idListChunkSize)
		for i := 0; i < descriptor.ChunkCount; i++ {
			start := i * idListChunkSize
			end := minInt(start+idListChunkSize, len(ids))
			data, err := marshalLine(idChunk{IDs: append([]string(nil), ids[start:end]...)})
			if err != nil {
				return err
			}
			files[idChunkPath(dir, key, i)] = data
		}
	}
	data, err := marshalLine(descriptor)
	if err != nil {
		return err
	}
	files[dir+"/"+key+".json"] = data
	return nil
}

func loadChunkedIDList(root, directory, key string, descriptor idList) ([]string, error) {
	if descriptor.Count < 0 {
		return nil, fmt.Errorf("negative ID-list count for %s/%s", directory, key)
	}
	if descriptor.IDs != nil {
		if descriptor.ChunkSize != 0 || descriptor.ChunkCount != 0 {
			return nil, fmt.Errorf("inline ID list %s/%s also declares chunks", directory, key)
		}
		if len(descriptor.IDs) != descriptor.Count {
			return nil, fmt.Errorf("inline ID list %s/%s contains %d IDs, expected %d", directory, key, len(descriptor.IDs), descriptor.Count)
		}
		return append([]string(nil), descriptor.IDs...), nil
	}
	if descriptor.Count == 0 {
		return []string{}, nil
	}
	if descriptor.ChunkSize <= 0 || descriptor.ChunkCount != chunkCount(descriptor.Count, descriptor.ChunkSize) {
		return nil, fmt.Errorf("invalid chunk metadata for ID list %s/%s", directory, key)
	}

	ids := make([]string, 0, descriptor.Count)
	for i := 0; i < descriptor.ChunkCount; i++ {
		path := filepath.Join(root, filepath.FromSlash(idChunkPath("index/"+directory, key, i)))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read ID-list chunk %s/%s #%d: %w", directory, key, i, err)
		}
		var chunk idChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			return nil, fmt.Errorf("parse ID-list chunk %s/%s #%d: %w", directory, key, i, err)
		}
		if len(chunk.IDs) == 0 || len(chunk.IDs) > descriptor.ChunkSize {
			return nil, fmt.Errorf("invalid ID-list chunk size for %s/%s #%d", directory, key, i)
		}
		ids = append(ids, chunk.IDs...)
	}
	if len(ids) != descriptor.Count {
		return nil, fmt.Errorf("chunked ID list %s/%s contains %d IDs, expected %d", directory, key, len(ids), descriptor.Count)
	}
	return ids, nil
}

func renderTermIndexValue(files map[string][]byte, term string, postings []termPosting) (termIndexValue, error) {
	value := termIndexValue{DocumentFrequency: len(postings)}
	if shouldSuppressTerm(len(postings)) {
		value.Suppressed = true
		return value, nil
	}
	if len(postings) <= termPostingChunkSize {
		value.Postings = append([]termPosting(nil), postings...)
		return value, nil
	}
	value.ChunkSize = termPostingChunkSize
	value.ChunkCount = chunkCount(len(postings), termPostingChunkSize)
	for i := 0; i < value.ChunkCount; i++ {
		start := i * termPostingChunkSize
		end := minInt(start+termPostingChunkSize, len(postings))
		data, err := marshalLine(termPostingChunk{Postings: append([]termPosting(nil), postings[start:end]...)})
		if err != nil {
			return termIndexValue{}, err
		}
		files[termPostingChunkPath(term, i)] = data
	}
	return value, nil
}

func loadTermPostings(root, term string, value termIndexValue) ([]termPosting, error) {
	if value.DocumentFrequency < 0 {
		return nil, fmt.Errorf("negative document frequency for term %q", term)
	}
	if value.Suppressed {
		return nil, fmt.Errorf("term %q is high-frequency and its postings are intentionally suppressed", term)
	}
	if value.Postings != nil {
		if value.ChunkSize != 0 || value.ChunkCount != 0 {
			return nil, fmt.Errorf("inline term %q also declares chunks", term)
		}
		if len(value.Postings) != value.DocumentFrequency {
			return nil, fmt.Errorf("term %q contains %d postings, expected %d", term, len(value.Postings), value.DocumentFrequency)
		}
		return append([]termPosting(nil), value.Postings...), nil
	}
	if value.DocumentFrequency == 0 {
		return []termPosting{}, nil
	}
	if value.ChunkSize <= 0 || value.ChunkCount != chunkCount(value.DocumentFrequency, value.ChunkSize) {
		return nil, fmt.Errorf("invalid posting chunk metadata for term %q", term)
	}

	postings := make([]termPosting, 0, value.DocumentFrequency)
	for i := 0; i < value.ChunkCount; i++ {
		path := filepath.Join(root, filepath.FromSlash(termPostingChunkPath(term, i)))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read posting chunk for term %q #%d: %w", term, i, err)
		}
		var chunk termPostingChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			return nil, fmt.Errorf("parse posting chunk for term %q #%d: %w", term, i, err)
		}
		if len(chunk.Postings) == 0 || len(chunk.Postings) > value.ChunkSize {
			return nil, fmt.Errorf("invalid posting chunk size for term %q #%d", term, i)
		}
		postings = append(postings, chunk.Postings...)
	}
	if len(postings) != value.DocumentFrequency {
		return nil, fmt.Errorf("term %q contains %d chunked postings, expected %d", term, len(postings), value.DocumentFrequency)
	}
	return postings, nil
}

func shouldSuppressTerm(documentFrequency int) bool {
	return documentFrequency > maxStoredTermPostings
}

func idChunkPath(dir, key string, index int) string {
	return filepath.ToSlash(filepath.Join(dir, key, fmt.Sprintf("%06d.json", index)))
}

func termHash(term string) string {
	sum := sha256.Sum256([]byte(term))
	return hex.EncodeToString(sum[:])
}

func termPostingChunkPath(term string, index int) string {
	hash := termHash(term)
	return filepath.ToSlash(filepath.Join("index", "term-postings", hash[:2], hash, fmt.Sprintf("%06d.json", index)))
}

func chunkCount(count, size int) int {
	if count <= 0 {
		return 0
	}
	return (count + size - 1) / size
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
