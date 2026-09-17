package coverage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/fingerprint"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// Analyzer name and exact measurement variants.
const (
	Name                 = "lcov-coverage"
	VariantLine          = "LCOV-DA-v1"
	VariantBranch        = "LCOV-BRDA-known-outcomes-v1"
	VariantChangedLine   = "LCOV-DA-on-git-added-lines-v1"
	VariantChangedBranch = "LCOV-BRDA-on-git-added-lines-v1"
)

// Importer implements analysis.RepositoryAnalyzer for LCOV reports.
type Importer struct{}

// New creates an Importer.
func New() *Importer {
	return &Importer{}
}

// Name implements analysis.RepositoryAnalyzer.
func (i *Importer) Name() string {
	return Name
}

// Analyze implements analysis.RepositoryAnalyzer. A missing or stale report is
// a diagnostic whose severity depends on whether coverage is required.
func (i *Importer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	if in.Config.Coverage.LCOVFile == nil {
		return analysis.Result{}, nil
	}
	reportName := *in.Config.Coverage.LCOVFile
	reportPath := paths.Resolve(in.Root, reportName)
	severity := domain.SeverityWarning
	if in.Config.Coverage.Required {
		severity = domain.SeverityError
	}
	diagnostic := func(code, file, message string) domain.Diagnostic {
		return domain.Diagnostic{Code: code, Severity: severity, Message: message, File: file, Required: in.Config.Coverage.Required}
	}

	contents, reportTime, err := readReport(reportPath)
	if err != nil {
		return analysis.Result{Diagnostics: []domain.Diagnostic{
			diagnostic("coverage.report_unavailable", "", fmt.Sprintf("Could not read LCOV report %s: %v", reportName, err)),
		}}, nil
	}
	records, err := ParseLCOV(bytes.NewReader(contents))
	if err != nil {
		return analysis.Result{Diagnostics: []domain.Diagnostic{
			diagnostic("coverage.report_unavailable", "", fmt.Sprintf("Could not parse LCOV report %s: %v", reportName, err)),
		}}, nil
	}

	sources := make(map[string]*adapter.SourceFile, len(in.Files))
	for _, file := range in.Files {
		sources[file.RelativePath] = file
	}
	evidence := domain.Evidence{"report": reportName, "reportSha256": fingerprint.SHA256(contents)}
	var result analysis.Result
	for _, record := range records {
		relative, source := locateSource(in.Root, reportPath, record.Source, sources)
		if source == nil {
			continue
		}
		info, err := os.Stat(source.AbsolutePath)
		if err != nil {
			return analysis.Result{}, fmt.Errorf("stat %s: %w", source.AbsolutePath, err)
		}
		if info.ModTime().After(reportTime.Add(time.Millisecond)) {
			result.Diagnostics = append(result.Diagnostics, diagnostic("coverage.report_stale", relative,
				"Source file is newer than the LCOV report; coverage was not used for this file"))
			continue
		}
		result.Measurements = append(result.Measurements, fileMeasurements(record, relative, evidence)...)
		if changed, ok := in.ChangedLines[relative]; ok {
			result.Measurements = append(result.Measurements, changedMeasurements(record, relative, changed, evidence)...)
		}
	}
	if len(result.Measurements) == 0 && len(result.Diagnostics) == 0 {
		result.Diagnostics = append(result.Diagnostics, diagnostic("coverage.no_sources_mapped", "",
			"The LCOV report did not map to any analyzed source files"))
	}
	return result, nil
}

// readReport reads the report and its modification time from one open file, so
// the staleness check always sees the revision that was parsed.
func readReport(path string) ([]byte, time.Time, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer func() { _ = file.Close() }() // read-only
	info, err := file.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, time.Time{}, err
	}
	return contents, info.ModTime(), nil
}

// locateSource maps an SF path onto an analyzed file: first relative to the
// report's directory, then relative to the repository root.
func locateSource(root, reportPath, recorded string, sources map[string]*adapter.SourceFile) (string, *adapter.SourceFile) {
	relative := paths.Relative(root, paths.Resolve(filepath.Dir(reportPath), recorded))
	if source, ok := sources[relative]; ok {
		return relative, source
	}
	rootRelative := paths.Relative(root, paths.Resolve(root, recorded))
	if source, ok := sources[rootRelative]; ok {
		return rootRelative, source
	}
	return relative, nil
}

func fileMeasurements(record FileRecord, relative string, evidence domain.Evidence) []domain.Measurement {
	lineCovered := 0
	for _, hits := range record.Lines {
		if hits > 0 {
			lineCovered++
		}
	}
	branchTotal, branchCovered := 0, 0
	for branch := range record.KnownBranches() {
		branchTotal++
		if *branch.Taken > 0 {
			branchCovered++
		}
	}
	return []domain.Measurement{
		percentMeasurement(domain.MetricCoverageLine, VariantLine, relative, lineCovered, len(record.Lines),
			"No eligible lines in LCOV record", evidence),
		percentMeasurement(domain.MetricCoverageBranch, VariantBranch, relative, branchCovered, branchTotal,
			"No eligible branch outcomes in LCOV record", evidence),
	}
}

func changedMeasurements(record FileRecord, relative string, changed analysis.LineSet, evidence domain.Evidence) []domain.Measurement {
	lineTotal, lineCovered := 0, 0
	for line, hits := range record.Lines {
		if !changed.Has(line) {
			continue
		}
		lineTotal++
		if hits > 0 {
			lineCovered++
		}
	}
	branchTotal, branchCovered := 0, 0
	for branch := range record.KnownBranches() {
		if !changed.Has(branch.Line) {
			continue
		}
		branchTotal++
		if *branch.Taken > 0 {
			branchCovered++
		}
	}
	return []domain.Measurement{
		percentMeasurement(domain.MetricCoverageChangedLine, VariantChangedLine, relative, lineCovered, lineTotal,
			"No changed executable lines were mapped", evidence),
		percentMeasurement(domain.MetricCoverageChangedBranch, VariantChangedBranch, relative, branchCovered, branchTotal,
			"No changed branch outcomes were mapped", evidence),
	}
}

func percentMeasurement(id domain.MetricID, variant, file string, covered, total int, emptyReason string, base domain.Evidence) domain.Measurement {
	evidence := make(domain.Evidence, len(base)+2)
	for key, value := range base {
		evidence[key] = value
	}
	evidence["covered"] = covered
	evidence["total"] = total
	measurement := domain.Measurement{
		MetricID: id,
		Variant:  variant,
		Category: domain.CategoryCoverage,
		Scope:    domain.FileScope(file),
		Value:    domain.Percent(covered, total),
		Unit:     domain.UnitPercent,
		Status:   domain.StatusMeasured,
		Evidence: evidence,
	}
	if total == 0 {
		measurement.Status = domain.StatusNotApplicable
		measurement.Reason = emptyReason
	}
	return measurement
}
