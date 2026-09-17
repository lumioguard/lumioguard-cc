package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/glob"
)

var metricNames = []string{
	"cyclomatic",
	"cognitive",
	"nestingDepth",
	"functionLines",
	"parameterCount",
	"fileTokens",
	"duplicationPercent",
	"fanOut",
}

// Parse validates raw JSON strictly and returns the typed configuration, so a
// typo or unknown field can never silently weaken a policy.
func Parse(data []byte) (domain.Config, error) {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return domain.Config{}, fmt.Errorf("configuration is not valid JSON: %w", err)
	}
	if err := validateDocument(document); err != nil {
		return domain.Config{}, err
	}
	var cfg domain.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return domain.Config{}, fmt.Errorf("decode configuration: %w", err)
	}
	return cfg, nil
}

func validateDocument(value any) error {
	root, err := object(value, "configuration")
	if err != nil {
		return errors.New("configuration must be a JSON object")
	}
	if err := noExtras(root, []string{"schemaVersion", "source", "metrics", "architecture", "coverage"}, "configuration"); err != nil {
		return err
	}
	if root["schemaVersion"] != domain.ConfigSchemaVersion {
		return fmt.Errorf("unsupported configuration schemaVersion: %v", root["schemaVersion"])
	}
	if err := validateSource(root["source"]); err != nil {
		return err
	}
	if err := validateMetrics(root["metrics"]); err != nil {
		return err
	}
	if err := validateArchitecture(root["architecture"]); err != nil {
		return err
	}
	return validateCoverage(root["coverage"])
}

func validateSource(value any) error {
	source, err := object(value, "source")
	if err != nil {
		return err
	}
	if err := noExtras(source, []string{"include", "exclude"}, "source"); err != nil {
		return err
	}
	for _, field := range []string{"include", "exclude"} {
		patterns, err := stringArray(source[field], "source."+field)
		if err != nil {
			return err
		}
		if err := validGlobs(patterns, "source."+field); err != nil {
			return err
		}
	}
	return nil
}

func validateMetrics(value any) error {
	metrics, err := object(value, "metrics")
	if err != nil {
		return err
	}
	if err := noExtras(metrics, metricNames, "metrics"); err != nil {
		return err
	}
	for _, name := range metricNames {
		var extra []string
		if name == "duplicationPercent" {
			extra = []string{"minTokens", "minLines"}
		}
		if err := validatePolicy(metrics[name], "metrics."+name, extra); err != nil {
			return err
		}
	}
	duplication, _ := metrics["duplicationPercent"].(map[string]any)
	for _, field := range []string{"minTokens", "minLines"} {
		if err := positiveInteger(duplication[field], "metrics.duplicationPercent."+field); err != nil {
			return err
		}
	}
	return nil
}

func validatePolicy(value any, field string, extraFields []string) error {
	policy, err := object(value, field)
	if err != nil {
		return err
	}
	allowed := append([]string{"enabled", "threshold", "severity", "block"}, extraFields...)
	if err := noExtras(policy, allowed, field); err != nil {
		return err
	}
	if _, ok := policy["enabled"].(bool); !ok {
		return fmt.Errorf("%s.enabled must be boolean", field)
	}
	threshold, ok := policy["threshold"].(float64)
	if !ok || math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 {
		return fmt.Errorf("%s.threshold must be a non-negative number", field)
	}
	severity, ok := policy["severity"].(string)
	if !ok || !domain.Severity(severity).IsValid() {
		return fmt.Errorf("%s.severity must be info, warning, or error", field)
	}
	if _, ok := policy["block"].(bool); !ok {
		return fmt.Errorf("%s.block must be boolean", field)
	}
	return nil
}

func validateArchitecture(value any) error {
	architecture, err := object(value, "architecture")
	if err != nil {
		return err
	}
	if err := noExtras(architecture, []string{"boundaries", "blockCycles", "blockViolations"}, "architecture"); err != nil {
		return err
	}
	boundaries, ok := architecture["boundaries"].([]any)
	if !ok {
		return errors.New("architecture.boundaries must be an array")
	}
	names := make(map[string]struct{}, len(boundaries))
	dependencies := make(map[string][]string, len(boundaries))
	for index, item := range boundaries {
		field := fmt.Sprintf("architecture.boundaries[%d]", index)
		boundary, err := object(item, field)
		if err != nil {
			return err
		}
		if err := noExtras(boundary, []string{"name", "include", "mayDependOn"}, field); err != nil {
			return err
		}
		name, ok := boundary["name"].(string)
		if !ok || name == "" {
			return fmt.Errorf("%s.name must be a non-empty string", field)
		}
		include, err := stringArray(boundary["include"], field+".include")
		if err != nil {
			return err
		}
		if err := validGlobs(include, field+".include"); err != nil {
			return err
		}
		mayDependOn, err := stringArray(boundary["mayDependOn"], field+".mayDependOn")
		if err != nil {
			return err
		}
		if _, duplicate := names[name]; duplicate {
			return errors.New("architecture boundary names must be unique")
		}
		names[name] = struct{}{}
		dependencies[name] = mayDependOn
	}
	for _, flag := range []string{"blockCycles", "blockViolations"} {
		if _, ok := architecture[flag].(bool); !ok {
			return errors.New("architecture block flags must be boolean")
		}
	}
	for _, item := range boundaries {
		boundary := item.(map[string]any)
		name := boundary["name"].(string)
		for _, dependency := range dependencies[name] {
			if _, known := names[dependency]; !known {
				return fmt.Errorf("boundary %s references unknown boundary %s", name, dependency)
			}
		}
	}
	return nil
}

func validateCoverage(value any) error {
	coverage, err := object(value, "coverage")
	if err != nil {
		return err
	}
	if err := noExtras(coverage, []string{"lcovFile", "required"}, "coverage"); err != nil {
		return err
	}
	lcov, present := coverage["lcovFile"]
	if !present {
		return errors.New("coverage.lcovFile must be a string or null")
	}
	if lcov != nil {
		if _, ok := lcov.(string); !ok {
			return errors.New("coverage.lcovFile must be a string or null")
		}
	}
	if _, ok := coverage["required"].(bool); !ok {
		return errors.New("coverage.required must be boolean")
	}
	return nil
}

func object(value any, field string) (map[string]any, error) {
	candidate, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", field)
	}
	return candidate, nil
}

func noExtras(value map[string]any, allowed []string, field string) error {
	var extras []string
	for key := range value {
		if !slices.Contains(allowed, key) {
			extras = append(extras, key)
		}
	}
	if len(extras) == 0 {
		return nil
	}
	slices.Sort(extras)
	return fmt.Errorf("%s contains unknown fields: %s", field, strings.Join(extras, ", "))
}

func stringArray(value any, field string) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array of strings", field)
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be an array of strings", field)
		}
		result = append(result, text)
	}
	return result, nil
}

func validGlobs(patterns []string, field string) error {
	for index, pattern := range patterns {
		if !glob.Valid(pattern) {
			return fmt.Errorf("%s[%d] is not a valid glob pattern: %q", field, index, pattern)
		}
	}
	return nil
}

func positiveInteger(value any, field string) error {
	number, ok := value.(float64)
	if !ok || number != math.Trunc(number) || number < 1 {
		return fmt.Errorf("%s must be a positive integer", field)
	}
	return nil
}
