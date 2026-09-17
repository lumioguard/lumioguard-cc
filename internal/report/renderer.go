// Package report renders reports for people or as JSON. JSON output is exactly
// one versioned document on stdout.
package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

// Format selects a renderer.
type Format string

// Supported output formats.
const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
	FormatSARIF Format = "sarif"
)

// ParseFormat validates a --format value.
func ParseFormat(value string) (Format, error) {
	switch Format(value) {
	case "", FormatHuman:
		return FormatHuman, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatSARIF:
		return FormatSARIF, nil
	default:
		return "", fmt.Errorf("--format must be human, json or sarif")
	}
}

// Renderer writes a report to a writer.
type Renderer interface {
	Render(w io.Writer, report *domain.Report) error
}

// NewRenderer returns the renderer for a format.
func NewRenderer(format Format) Renderer {
	switch format {
	case FormatJSON:
		return JSONRenderer{}
	case FormatSARIF:
		return SARIFRenderer{}
	default:
		return HumanRenderer{}
	}
}

// JSONRenderer writes the report as indented JSON followed by a newline.
type JSONRenderer struct{}

// Render implements Renderer.
func (JSONRenderer) Render(w io.Writer, report *domain.Report) error {
	return WriteJSON(w, report)
}

// WriteJSON encodes any value as indented JSON. HTML escaping is disabled so
// paths and messages stay readable.
func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
