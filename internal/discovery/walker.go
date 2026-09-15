// Package discovery walks a repository and selects source files with the
// configured include and exclude glob patterns.
package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/glob"
	"github.com/lumiostack/lumioguard-cc/internal/paths"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// Result lists the selected files in deterministic order and what was skipped.
type Result struct {
	Files      []string
	Exclusions domain.ExclusionSummary
}

// Walker discovers source files under a repository root.
type Walker struct {
	hardSkipped map[string]struct{}
}

// NewWalker creates a Walker that always skips VCS, dependency and tool state
// directories regardless of configuration.
func NewWalker() *Walker {
	return &Walker{hardSkipped: map[string]struct{}{
		".git":                     {},
		"node_modules":             {},
		"__pycache__":              {},
		product.StateDirectoryName: {},
	}}
}

// Discover walks root in sorted order and returns files matching an include
// pattern and no exclude pattern. Symbolic links are never followed.
func (w *Walker) Discover(root string, source domain.SourceConfig) (Result, error) {
	result := Result{
		Files:      []string{},
		Exclusions: domain.ExclusionSummary{Patterns: slices.Clone(source.Exclude)},
	}
	if result.Exclusions.Patterns == nil {
		result.Exclusions.Patterns = []string{}
	}
	if err := w.visit(root, root, source, &result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (w *Walker) visit(root, directory string, source domain.SourceConfig, result *Result) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read directory %s: %w", directory, err)
	}
	for _, entry := range entries {
		absolute := filepath.Join(directory, entry.Name())
		relative := paths.Relative(root, absolute)
		if entry.Type()&fs.ModeSymlink != 0 {
			result.Exclusions.ExcludedFiles++
			continue
		}
		if entry.IsDir() {
			if w.isHardSkipped(entry.Name()) || glob.MatchesDirectory(relative, source.Exclude) {
				result.Exclusions.ExcludedDirectories++
				continue
			}
			if err := w.visit(root, absolute, source, result); err != nil {
				return err
			}
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}
		if glob.MatchesAny(relative, source.Exclude) {
			result.Exclusions.ExcludedFiles++
			continue
		}
		if glob.MatchesAny(relative, source.Include) {
			result.Files = append(result.Files, absolute)
		}
	}
	return nil
}

func (w *Walker) isHardSkipped(name string) bool {
	_, skipped := w.hardSkipped[name]
	return skipped
}
