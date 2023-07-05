package embed

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/binary"
	"github.com/sourcegraph/sourcegraph/internal/paths"
)

const (
	MIN_EMBEDDABLE_FILE_SIZE = 32
	MAX_LINE_LENGTH          = 2048
)

var autogeneratedFileHeaders = [][]byte{
	[]byte("autogenerated file"),
	[]byte("lockfile"),
	[]byte("generated by"),
	[]byte("do not edit"),
}

var TextFileExtensions = map[string]struct{}{
	"md":       {},
	"markdown": {},
	"rst":      {},
	"txt":      {},
}

var DefaultExcludedFilePathPatterns = []string{
	".*ignore", // Files like .gitignore, .eslintignore
	".gitattributes",
	".mailmap",
	"*.csv",
	"*.svg",
	"*.xml",
	"__fixtures__/",
	"node_modules/",
	"testdata/",
	"mocks/",
	"vendor/",
}

func GetDefaultExcludedFilePathPatterns() []*paths.GlobPattern {
	return CompileGlobPatterns(DefaultExcludedFilePathPatterns)
}

func CompileGlobPatterns(patterns []string) []*paths.GlobPattern {
	globPatterns := make([]*paths.GlobPattern, 0, len(patterns))
	for _, pattern := range patterns {
		globPattern, err := paths.Compile(pattern)
		if err != nil {
			continue
		}
		globPatterns = append(globPatterns, globPattern)
	}
	return globPatterns
}

func isExcludedFilePath(filePath string, excludedFilePathPatterns []*paths.GlobPattern) bool {
	for _, excludedFilePathPattern := range excludedFilePathPatterns {
		if excludedFilePathPattern.Match(filePath) {
			return true
		}
	}
	return false
}

type SkipReason = string

const (
	// File was not skipped
	SkipReasonNone SkipReason = ""

	// File is binary
	SkipReasonBinary SkipReason = "binary"

	// File is too small to provide useful embeddings
	SkipReasonSmall SkipReason = "small"

	// File is larger than the max file size
	SkipReasonLarge SkipReason = "large"

	// File is autogenerated
	SkipReasonAutogenerated SkipReason = "autogenerated"

	// File has a line that is too long
	SkipReasonLongLine SkipReason = "longLine"

	// File was excluded by configuration rules
	SkipReasonExcluded SkipReason = "excluded"

	// File was excluded because we hit the max embedding limit for the repo
	SkipReasonMaxEmbeddings SkipReason = "maxEmbeddings"
)

func isEmbeddableFileContent(content []byte) (embeddable bool, reason SkipReason) {
	if binary.IsBinary(content) {
		return false, SkipReasonBinary
	}

	if len(bytes.TrimSpace(content)) < MIN_EMBEDDABLE_FILE_SIZE {
		return false, SkipReasonSmall
	}

	lines := bytes.Split(content, []byte("\n"))

	fileHeader := bytes.ToLower(bytes.Join(lines[0:min(5, len(lines))], []byte("\n")))
	for _, header := range autogeneratedFileHeaders {
		if bytes.Contains(fileHeader, header) {
			return false, SkipReasonAutogenerated
		}
	}

	for _, line := range lines {
		if len(line) > MAX_LINE_LENGTH {
			return false, SkipReasonLongLine
		}
	}

	return true, SkipReasonNone
}

func IsValidTextFile(fileName string) bool {
	ext := strings.TrimPrefix(filepath.Ext(fileName), ".")
	_, ok := TextFileExtensions[strings.ToLower(ext)]
	if ok {
		return true
	}
	basename := strings.ToLower(filepath.Base(fileName))
	return strings.HasPrefix(basename, "license")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}