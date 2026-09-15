package domain

import "encoding/json"

// Evidence carries metric-specific supporting data. Keys are serialised in
// sorted order so reports stay deterministic.
type Evidence map[string]any

// MarshalJSON renders a nil map as an empty object, because the report schema
// requires evidence to be an object.
func (e Evidence) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]any(e))
}

// Measurement is one metric value for one scope, together with its exact
// variant and status. Value is nil whenever Status is not measured.
type Measurement struct {
	MetricID       MetricID          `json:"metricId"`
	Variant        string            `json:"variant"`
	Category       Category          `json:"category"`
	Scope          Scope             `json:"scope"`
	Value          *float64          `json:"value"`
	Unit           Unit              `json:"unit"`
	Status         MeasurementStatus `json:"status"`
	Reason         string            `json:"reason,omitempty"`
	Baseline       *float64          `json:"baseline,omitempty"`
	Delta          *float64          `json:"delta,omitempty"`
	Classification Classification    `json:"classification,omitempty"`
	Evidence       Evidence          `json:"evidence,omitempty"`
}

// IsMeasured reports whether the measurement carries a usable numeric value.
func (m Measurement) IsMeasured() bool {
	return m.Status == StatusMeasured && m.Value != nil
}

// WithoutComparison strips baseline-relative fields so the measurement can be
// persisted as baseline data.
func (m Measurement) WithoutComparison() Measurement {
	m.Baseline = nil
	m.Delta = nil
	m.Classification = ""
	return m
}
