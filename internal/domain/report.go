package domain

// Schema versions of the three persisted JSON documents.
const (
	ReportSchemaVersion   = "1.0"
	ConfigSchemaVersion   = "1.0"
	BaselineSchemaVersion = "1.0"
)

// ToolInfo identifies the tool that produced a document.
type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// RunInfo is volatile metadata about one execution. It is kept separate from
// comparable results so repeated runs can be diffed after removing it.
type RunInfo struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"createdAt"`
	DurationMs int64  `json:"durationMs"`
}

// GitSnapshot records the Git state of the analyzed tree, when available.
type GitSnapshot struct {
	Available  bool    `json:"available"`
	Commit     *string `json:"commit"`
	Dirty      *bool   `json:"dirty"`
	BaseCommit *string `json:"baseCommit,omitempty"`
}

// Snapshot identifies exactly which source and configuration were analyzed.
type Snapshot struct {
	ID         string            `json:"id"`
	Kind       SnapshotKind      `json:"kind"`
	Root       string            `json:"root"`
	SourceHash string            `json:"sourceHash"`
	FileHashes map[string]string `json:"fileHashes"`
	ConfigHash string            `json:"configHash"`
	Git        GitSnapshot       `json:"git"`
}

// AdapterSummary describes one language adapter and what it analyzed.
type AdapterSummary struct {
	ID            string            `json:"id"`
	Version       string            `json:"version"`
	Languages     []string          `json:"languages"`
	Analyzers     map[string]string `json:"analyzers"`
	FilesAnalyzed int               `json:"filesAnalyzed"`
	Complete      bool              `json:"complete"`
}

// PolicyResult is the gate outcome with its supporting counts.
type PolicyResult struct {
	Status           PolicyStatus `json:"status"`
	BlockingFindings int          `json:"blockingFindings"`
	WarningFindings  int          `json:"warningFindings"`
}

// Comparison states what the report was compared against.
type Comparison struct {
	Mode       ComparisonMode `json:"mode"`
	Reference  *string        `json:"reference"`
	Comparable bool           `json:"comparable"`
}

// ScopeSummary counts discovered and analyzed files per language.
type ScopeSummary struct {
	FilesDiscovered int            `json:"filesDiscovered"`
	FilesAnalyzed   int            `json:"filesAnalyzed"`
	Languages       map[string]int `json:"languages"`
}

// ExclusionSummary discloses what discovery skipped.
type ExclusionSummary struct {
	Patterns            []string `json:"patterns"`
	ExcludedDirectories int      `json:"excludedDirectories"`
	ExcludedFiles       int      `json:"excludedFiles"`
}

// Report is the versioned analysis document written by `check`.
type Report struct {
	SchemaVersion string           `json:"schemaVersion"`
	Tool          ToolInfo         `json:"tool"`
	Run           RunInfo          `json:"run"`
	Snapshot      Snapshot         `json:"snapshot"`
	Comparison    Comparison       `json:"comparison"`
	Scope         ScopeSummary     `json:"scope"`
	Adapters      []AdapterSummary `json:"adapters"`
	Measurements  []Measurement    `json:"measurements"`
	Findings      []Finding        `json:"findings"`
	Diagnostics   []Diagnostic     `json:"diagnostics"`
	Exclusions    ExclusionSummary `json:"exclusions"`
	Policy        PolicyResult     `json:"policy"`
}

// Process exit codes, part of the contract with agents.
const (
	ExitPassed     = 0
	ExitFailed     = 1
	ExitIncomplete = 2
)

// ExitCode maps the policy status to the process exit code.
func (r *Report) ExitCode() int {
	switch r.Policy.Status {
	case PolicyIncomplete:
		return ExitIncomplete
	case PolicyFailed:
		return ExitFailed
	default:
		return ExitPassed
	}
}

// ActiveFindings returns findings that still exist in the current analysis.
func (r *Report) ActiveFindings() []Finding {
	active := make([]Finding, 0, len(r.Findings))
	for _, finding := range r.Findings {
		if finding.IsActive() {
			active = append(active, finding)
		}
	}
	return active
}

// ResolvedCount counts findings that disappeared since the comparison reference.
func (r *Report) ResolvedCount() int {
	count := 0
	for _, finding := range r.Findings {
		if !finding.IsActive() {
			count++
		}
	}
	return count
}
