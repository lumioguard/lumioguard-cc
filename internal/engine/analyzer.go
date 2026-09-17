package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/language"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

const createdAtLayout = "2006-01-02T15:04:05.000Z07:00"

// Dependencies are the collaborators an Analyzer needs.
type Dependencies struct {
	Tool       domain.ToolInfo
	Adapters   *adapter.Registry
	Discoverer FileDiscoverer
	Repository []analysis.RepositoryAnalyzer
	Git        GitInspector
	Comparer   Comparer
	Policy     PolicyEvaluator
}

// Option customises an Analyzer.
type Option func(*Analyzer)

// WithClock replaces the clock.
func WithClock(clock Clock) Option {
	return func(a *Analyzer) { a.clock = clock }
}

// WithIDGenerator replaces the run ID generator.
func WithIDGenerator(ids IDGenerator) Option {
	return func(a *Analyzer) { a.ids = ids }
}

// WithWorkers sets how many files are analyzed concurrently.
func WithWorkers(workers int) Option {
	return func(a *Analyzer) {
		if workers > 0 {
			a.workers = workers
		}
	}
}

// Analyzer runs the analysis pipeline.
type Analyzer struct {
	deps    Dependencies
	clock   Clock
	ids     IDGenerator
	workers int
}

// New creates an Analyzer.
func New(deps Dependencies, options ...Option) *Analyzer {
	analyzer := &Analyzer{
		deps:    deps,
		clock:   systemClock{},
		ids:     randomIDs{},
		workers: runtime.NumCPU(),
	}
	for _, option := range options {
		option(analyzer)
	}
	return analyzer
}

// Request describes one analysis.
type Request struct {
	Root         string
	Config       domain.Config
	Baseline     *domain.Baseline
	BaselineName string
	GitBase      string
}

type selection struct {
	path    string
	adapter adapter.LanguageAdapter
}

// Analyze runs the pipeline. Measurement problems become diagnostics inside
// the report; only I/O and internal failures return an error.
func (a *Analyzer) Analyze(ctx context.Context, req Request) (*domain.Report, error) {
	started := a.clock.Now()
	root, err := filepath.Abs(req.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	discovered, err := a.deps.Discoverer.Discover(root, req.Config.Source)
	if err != nil {
		return nil, fmt.Errorf("discover source files: %w", err)
	}
	selections := a.assign(discovered.Files)
	files, diagnostics, err := a.analyzeFiles(ctx, root, selections)
	if err != nil {
		return nil, err
	}

	var changed analysis.ChangedLines
	baseCommit := ""
	if req.GitBase != "" {
		var changeDiagnostics []domain.Diagnostic
		changed, baseCommit, changeDiagnostics = a.deps.Git.ChangedLines(ctx, root, req.GitBase)
		diagnostics = append(diagnostics, changeDiagnostics...)
	}

	repository, err := a.runRepositoryAnalyzers(ctx, analysis.Input{
		Root:         root,
		Files:        files,
		Config:       req.Config,
		ChangedLines: changed,
		Adapters:     a.deps.Adapters,
	})
	if err != nil {
		return nil, err
	}
	diagnostics = append(diagnostics, repository.Diagnostics...)

	// Per-file measurements outnumber repository ones by orders of magnitude,
	// so size the slice from both rather than regrowing it per file.
	total := len(repository.Measurements)
	for _, file := range files {
		total += len(file.Measurements)
	}
	measurements := make([]domain.Measurement, 0, total)
	for _, file := range files {
		measurements = append(measurements, file.Measurements...)
	}
	measurements = append(measurements, repository.Measurements...)

	gitSnapshot, gitDiagnostics := a.deps.Git.Inspect(ctx, root, baseCommit)
	diagnostics = append(diagnostics, gitDiagnostics...)
	snapshot, err := buildSnapshot(root, files, req.Config, gitSnapshot)
	if err != nil {
		return nil, err
	}

	findings := slices.Concat(repository.Findings, a.deps.Policy.ThresholdFindings(measurements, req.Config))
	report := &domain.Report{
		SchemaVersion: domain.ReportSchemaVersion,
		Tool:          a.deps.Tool,
		Run:           domain.RunInfo{ID: a.ids.NewRunID(), CreatedAt: started.UTC().Format(createdAtLayout)},
		Snapshot:      snapshot,
		Comparison:    comparisonFor(req.GitBase, baseCommit),
		Scope:         scopeSummary(discovered.Files, len(files)),
		Adapters:      a.adapterSummaries(selections, files),
		Measurements:  sortMeasurements(measurements),
		Findings:      sortFindings(findings),
		Diagnostics:   sortDiagnostics(diagnostics),
		Exclusions:    discovered.Exclusions,
		Policy:        a.deps.Policy.Result(findings, domain.HasRequiredError(diagnostics)),
	}
	if req.Baseline != nil {
		report = a.deps.Comparer.Compare(report, req.Baseline, domain.ComparisonBaseline, req.BaselineName)
		report.Policy = a.deps.Policy.Result(report.Findings, domain.HasRequiredError(report.Diagnostics))
	}
	report.Run.DurationMs = a.clock.Now().Sub(started).Milliseconds()
	return report, nil
}

func (a *Analyzer) assign(files []string) []selection {
	selections := make([]selection, len(files))
	for i, file := range files {
		selections[i].path = file
		if found, ok := a.deps.Adapters.Find(file); ok {
			selections[i].adapter = found
		}
	}
	return selections
}

// analyzeFiles runs adapters concurrently and returns results in discovery
// order. Files without an adapter produce a required diagnostic.
func (a *Analyzer) analyzeFiles(ctx context.Context, root string, selections []selection) ([]*adapter.SourceFile, []domain.Diagnostic, error) {
	results := make([]*adapter.SourceFile, len(selections))
	errs := make([]error, len(selections))
	semaphore := make(chan struct{}, a.workers)
	var wait sync.WaitGroup
	for i, item := range selections {
		if item.adapter == nil {
			continue
		}
		wait.Add(1)
		semaphore <- struct{}{}
		go func(index int, item selection) {
			defer wait.Done()
			defer func() { <-semaphore }()
			results[index], errs[index] = item.adapter.AnalyzeFile(ctx, root, item.path)
		}(i, item)
	}
	wait.Wait()

	files := make([]*adapter.SourceFile, 0, len(selections))
	var diagnostics []domain.Diagnostic
	for i, item := range selections {
		if item.adapter == nil {
			diagnostics = append(diagnostics, domain.RequiredError("adapter.unsupported_language",
				paths.Relative(root, item.path), "No installed language adapter supports this included source file"))
			continue
		}
		if errs[i] != nil {
			return nil, nil, fmt.Errorf("analyze %s: %w", paths.Relative(root, item.path), errs[i])
		}
		files = append(files, results[i])
		diagnostics = append(diagnostics, results[i].Diagnostics...)
	}
	return files, diagnostics, nil
}

func (a *Analyzer) runRepositoryAnalyzers(ctx context.Context, input analysis.Input) (analysis.Result, error) {
	var merged analysis.Result
	for _, analyzer := range a.deps.Repository {
		result, err := analyzer.Analyze(ctx, input)
		if err != nil {
			return analysis.Result{}, fmt.Errorf("%s analysis: %w", analyzer.Name(), err)
		}
		merged.Merge(result)
	}
	return merged, nil
}

func (a *Analyzer) adapterSummaries(selections []selection, files []*adapter.SourceFile) []domain.AdapterSummary {
	byPath := make(map[string]*adapter.SourceFile, len(files))
	for _, file := range files {
		byPath[file.AbsolutePath] = file
	}
	summaries := make([]domain.AdapterSummary, 0)
	for _, languageAdapter := range a.deps.Adapters.All() {
		summary := adapter.Summary(languageAdapter)
		summary.Complete = true
		for _, item := range selections {
			if item.adapter == nil || item.adapter.ID() != languageAdapter.ID() {
				continue
			}
			file, ok := byPath[item.path]
			if !ok {
				continue
			}
			summary.FilesAnalyzed++
			if file.HasRequiredDiagnostic() {
				summary.Complete = false
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func comparisonFor(gitBase, baseCommit string) domain.Comparison {
	if gitBase == "" {
		return domain.Comparison{Mode: domain.ComparisonNone, Reference: nil, Comparable: false}
	}
	reference := gitBase
	if baseCommit != "" {
		reference = baseCommit
	}
	return domain.Comparison{Mode: domain.ComparisonGit, Reference: &reference, Comparable: false}
}

func scopeSummary(discovered []string, analyzed int) domain.ScopeSummary {
	languages := make(map[string]int)
	for _, file := range discovered {
		languages[language.NameFor(file)]++
	}
	return domain.ScopeSummary{FilesDiscovered: len(discovered), FilesAnalyzed: analyzed, Languages: languages}
}
