// Package analysis defines the contract for analyzers that run across all parsed
// files. New analyzers are registered in the composition root, not the engine.
package analysis

import (
	"context"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

// LineSet is a set of 1-based line numbers.
type LineSet map[int]struct{}

// Add inserts a line.
func (s LineSet) Add(line int) {
	s[line] = struct{}{}
}

// Has reports membership.
func (s LineSet) Has(line int) bool {
	_, ok := s[line]
	return ok
}

// ChangedLines maps repository-relative paths to the lines added since a Git
// base. An empty set means the file changed without adding lines.
type ChangedLines map[string]LineSet

// Input is everything a repository analyzer may consume.
type Input struct {
	Root         string
	Files        []*adapter.SourceFile
	Config       domain.Config
	ChangedLines ChangedLines
	// Adapters locates the adapter (and therefore the import resolver) of a file.
	Adapters adapter.Finder
}

// Result is what a repository analyzer contributes to the report.
type Result struct {
	Measurements []domain.Measurement
	Findings     []domain.Finding
	Diagnostics  []domain.Diagnostic
}

// Merge appends another result into r.
func (r *Result) Merge(other Result) {
	r.Measurements = append(r.Measurements, other.Measurements...)
	r.Findings = append(r.Findings, other.Findings...)
	r.Diagnostics = append(r.Diagnostics, other.Diagnostics...)
}

// RepositoryAnalyzer computes cross-file measurements and findings.
type RepositoryAnalyzer interface {
	// Name identifies the analyzer in error messages.
	Name() string
	// Analyze runs the analyzer. Measurement problems are diagnostics; an error
	// signals an internal failure that must stop the run.
	Analyze(ctx context.Context, in Input) (Result, error)
}
