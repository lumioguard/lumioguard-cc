// Package config loads, validates and writes the repository configuration
// file. Defaults are advisory review prompts, not universal standards.
package config

import (
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/language"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

func advisory(threshold float64) domain.MetricPolicy {
	return domain.MetricPolicy{
		Enabled:   true,
		Threshold: threshold,
		Severity:  domain.SeverityWarning,
		Block:     false,
	}
}

// Default returns a fresh copy of the default configuration: every supported
// language, without dependency, build, cache and virtual-environment directories.
func Default() domain.Config {
	return domain.Config{
		SchemaVersion: domain.ConfigSchemaVersion,
		Source: domain.SourceConfig{
			Include: language.IncludePatterns(),
			Exclude: []string{
				"**/node_modules/**",
				"**/dist/**",
				"**/build/**",
				"**/out/**",
				"**/coverage/**",
				"**/.git/**",
				"**/" + product.StateDirectoryName + "/**",
				"**/*.min.js",
				"**/*.generated.*",
				// Framework build output and caches that hold compiled copies of the sources.
				"**/.next/**",
				"**/.nuxt/**",
				"**/.output/**",
				"**/.svelte-kit/**",
				"**/.turbo/**",
				"**/.cache/**",
				"**/.parcel-cache/**",
				"**/__pycache__/**",
				"**/.venv/**",
				"**/venv/**",
				"**/site-packages/**",
				"**/target/**",
				"**/.gradle/**",
				"**/vendor/**",
				"**/testdata/**",
			},
		},
		Metrics: domain.MetricsConfig{
			Cyclomatic:     advisory(12),
			Cognitive:      advisory(15),
			NestingDepth:   advisory(4),
			FunctionLines:  advisory(80),
			ParameterCount: advisory(5),
			FileTokens:     advisory(4000),
			DuplicationPercent: domain.DuplicationPolicy{
				MetricPolicy: advisory(5),
				MinTokens:    50,
				MinLines:     5,
			},
			FanOut: advisory(20),
		},
		Architecture: domain.ArchitectureConfig{
			Boundaries:      []domain.Boundary{},
			BlockCycles:     true,
			BlockViolations: true,
		},
		Coverage: domain.CoverageConfig{LCOVFile: nil, Required: false},
	}
}
