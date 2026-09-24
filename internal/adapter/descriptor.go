package adapter

import (
	"maps"
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// Descriptor is the fixed identity of a language adapter. Embedding it provides ID,
// Version, Languages, Analyzers and Supports; the adapter writes Resolver and AnalyzeFile.
type Descriptor struct {
	AdapterID        string
	AdapterVersion   string
	LanguageNames    []string
	AnalyzerVersions map[string]string
	// Extensions holds lower-case extensions with their dot, such as ".c".
	Extensions map[string]bool
}

// ID implements LanguageAdapter.
func (d Descriptor) ID() string { return d.AdapterID }

// Version implements LanguageAdapter.
func (d Descriptor) Version() string { return d.AdapterVersion }

// Languages implements LanguageAdapter.
func (d Descriptor) Languages() []string { return slices.Clone(d.LanguageNames) }

// Analyzers implements LanguageAdapter.
func (d Descriptor) Analyzers() map[string]string { return maps.Clone(d.AnalyzerVersions) }

// Supports implements LanguageAdapter.
func (d Descriptor) Supports(filename string) bool { return d.Extensions[paths.Extension(filename)] }
