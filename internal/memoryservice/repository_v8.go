package memoryservice

import (
	"path/filepath"

	"github.com/runethread/core/internal/buildinfo"
	"github.com/runethread/core/internal/fsafety"
	"github.com/runethread/core/internal/memory"
)

func loadCanonicalRecords(root string) ([]memory.Record, error) {
	if buildinfo.ContractVersion >= 8 {
		return memory.LoadAllStrict(root)
	}
	return memory.LoadAll(root)
}

func loadDocumentStrict(root, sidecarRel string) (Document, error) {
	sidecarRel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(sidecarRel)))
	m, problems := memory.LoadUnder(root, sidecarRel)
	if len(problems) > 0 {
		return Document{}, errorf(CodeRepositoryInvalid, "get", nil, "%s: %s", sidecarRel, problems[0].Error())
	}
	markdownRel := m.ContentPath
	markdown, err := fsafety.ReadRegularFileUnder(root, markdownRel)
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
