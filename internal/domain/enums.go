package domain

// Severity ranks findings and diagnostics.
type Severity string

// Documented severity values.
const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// IsValid reports whether s is a documented severity value.
func (s Severity) IsValid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityError:
		return true
	default:
		return false
	}
}

// MeasurementStatus states whether a value was actually measured. A missing
// value is never converted to zero or one hundred percent.
type MeasurementStatus string

// Documented measurement statuses.
const (
	StatusMeasured      MeasurementStatus = "measured"
	StatusNotApplicable MeasurementStatus = "not_applicable"
	StatusUnavailable   MeasurementStatus = "unavailable"
)

// Classification describes how a measurement or finding relates to a baseline.
type Classification string

// Documented classifications.
const (
	ClassificationNew      Classification = "new"
	ClassificationExisting Classification = "existing"
	ClassificationWorsened Classification = "worsened"
	ClassificationImproved Classification = "improved"
	ClassificationResolved Classification = "resolved"
)

// ScopeKind identifies what a scope points at.
type ScopeKind string

// Documented scope kinds.
const (
	ScopeRepository ScopeKind = "repository"
	ScopeFile       ScopeKind = "file"
	ScopeFunction   ScopeKind = "function"
	ScopeCycle      ScopeKind = "cycle"
	ScopeDependency ScopeKind = "dependency"
)

// Category groups metrics by the maintainability property they inform.
type Category string

// Documented metric categories.
const (
	CategoryComplexity  Category = "complexity"
	CategorySize        Category = "size"
	CategoryCoupling    Category = "coupling"
	CategoryDuplication Category = "duplication"
	CategoryCoverage    Category = "coverage"
)

// Unit is the unit of a measured value or of a finding's current value.
type Unit string

// Documented units.
const (
	UnitCount     Unit = "count"
	UnitLines     Unit = "lines"
	UnitPercent   Unit = "percent"
	UnitViolation Unit = "violation"
)

// PolicyStatus is the outcome of the configured gate.
type PolicyStatus string

// Documented policy statuses. Incomplete takes precedence over failed.
const (
	PolicyPassed     PolicyStatus = "passed"
	PolicyFailed     PolicyStatus = "failed"
	PolicyIncomplete PolicyStatus = "incomplete"
)

// ComparisonMode says what the current analysis was compared with.
type ComparisonMode string

// Documented comparison modes.
const (
	ComparisonNone     ComparisonMode = "none"
	ComparisonBaseline ComparisonMode = "baseline"
	ComparisonGit      ComparisonMode = "git"
)

// SnapshotKind distinguishes a live working tree from a persisted baseline.
type SnapshotKind string

// Documented snapshot kinds.
const (
	SnapshotWorkingTree SnapshotKind = "working-tree"
	SnapshotBaseline    SnapshotKind = "baseline"
)
