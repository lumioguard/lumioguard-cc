package domain

// Baseline is a persisted, reviewed snapshot that later analyses are compared
// with. It stores hashes and results, never source text.
type Baseline struct {
	SchemaVersion string           `json:"schemaVersion"`
	Name          string           `json:"name"`
	CreatedAt     string           `json:"createdAt"`
	ToolVersion   string           `json:"toolVersion"`
	Snapshot      Snapshot         `json:"snapshot"`
	Adapters      []AdapterSummary `json:"adapters"`
	Measurements  []Measurement    `json:"measurements"`
	Findings      []Finding        `json:"findings"`
	Diagnostics   []Diagnostic     `json:"diagnostics"`
}
