package domain

// Finding is an actionable rule result: a threshold breach, a dependency
// cycle, a boundary violation or a clone group. Nullable numbers are pointers.
type Finding struct {
	ID             string         `json:"id"`
	RuleID         RuleID         `json:"ruleId"`
	Title          string         `json:"title"`
	Severity       Severity       `json:"severity"`
	Blocking       bool           `json:"blocking"`
	Scope          Scope          `json:"scope"`
	Message        string         `json:"message"`
	Classification Classification `json:"classification"`
	Current        *float64       `json:"current"`
	Baseline       *float64       `json:"baseline"`
	Delta          *float64       `json:"delta"`
	Threshold      *float64       `json:"threshold"`
	Unit           Unit           `json:"unit"`
	Evidence       Evidence       `json:"evidence"`
	Recheck        string         `json:"recheck"`
}

// IsActive reports whether the finding still exists in the current analysis.
func (f Finding) IsActive() bool {
	return f.Classification != ClassificationResolved
}

// IsRegression reports whether the comparison reference shows the finding as
// introduced or made worse, rather than as pre-existing debt.
func (f Finding) IsRegression() bool {
	return f.Classification == ClassificationNew || f.Classification == ClassificationWorsened
}

// BlocksGate reports whether the finding fails the gate: only blocking findings
// that are new or worsened relative to the comparison reference block.
func (f Finding) BlocksGate() bool {
	return f.Blocking && f.IsRegression()
}

// AsBaselineEntry normalises the finding for persistence in a baseline.
func (f Finding) AsBaselineEntry() Finding {
	f.Baseline = nil
	f.Delta = nil
	f.Classification = ClassificationExisting
	return f
}
