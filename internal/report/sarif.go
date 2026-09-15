package report

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/explain"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// SARIF 2.1.0 identifiers.
const (
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
	sarifVersion = "2.1.0"
	// sourceRootID names the analyzed root, so every result URI stays relative.
	sourceRootID = "%SRCROOT%"
	// fingerprintKey carries the finding ID, which is stable across runs, so
	// code scanning tools can match a result with the one they already show.
	fingerprintKey = "lumioguardFindingId/v1"
)

// SARIFRenderer writes the active findings as one SARIF 2.1.0 log, the format
// GitHub code scanning, GitLab and IDE viewers read.
type SARIFRenderer struct{}

// Render implements Renderer.
func (SARIFRenderer) Render(w io.Writer, report *domain.Report) error {
	return WriteJSON(w, buildSARIF(report, explain.NewCatalog()))
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool               sarifTool                        `json:"tool"`
	Invocations        []sarifInvocation                `json:"invocations"`
	OriginalURIBaseIDs map[string]sarifArtifactLocation `json:"originalUriBaseIds,omitempty"`
	Results            []sarifResult                    `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	ShortDescription     sarifMessage       `json:"shortDescription"`
	FullDescription      sarifMessage       `json:"fullDescription"`
	HelpURI              string             `json:"helpUri"`
	DefaultConfiguration sarifConfiguration `json:"defaultConfiguration"`
}

type sarifConfiguration struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifInvocation struct {
	ExecutionSuccessful        bool                `json:"executionSuccessful"`
	ExitCode                   int                 `json:"exitCode"`
	ToolExecutionNotifications []sarifNotification `json:"toolExecutionNotifications,omitempty"`
}

type sarifNotification struct {
	Level      string          `json:"level"`
	Message    sarifMessage    `json:"message"`
	Descriptor sarifDescriptor `json:"descriptor"`
	Locations  []sarifLocation `json:"locations,omitempty"`
}

type sarifDescriptor struct {
	ID string `json:"id"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	RuleIndex           int               `json:"ruleIndex"`
	Level               string            `json:"level"`
	Message             sarifMessage      `json:"message"`
	Locations           []sarifLocation   `json:"locations,omitempty"`
	RelatedLocations    []sarifLocation   `json:"relatedLocations,omitempty"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Properties          map[string]any    `json:"properties"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId,omitempty"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine,omitempty"`
}

func buildSARIF(report *domain.Report, catalog *explain.Catalog) sarifLog {
	run := sarifRun{
		Tool: sarifTool{Driver: sarifDriver{
			Name:           report.Tool.Name,
			Version:        report.Tool.Version,
			InformationURI: product.RepositoryURL,
			Rules:          []sarifRule{},
		}},
		Invocations: []sarifInvocation{{
			ExecutionSuccessful:        report.Policy.Status != domain.PolicyIncomplete,
			ExitCode:                   report.ExitCode(),
			ToolExecutionNotifications: notifications(report.Diagnostics),
		}},
		Results: []sarifResult{},
	}
	if base := fileURI(report.Snapshot.Root); base != "" {
		run.OriginalURIBaseIDs = map[string]sarifArtifactLocation{sourceRootID: {URI: base}}
	}
	ruleIndex := map[string]int{}
	for _, finding := range report.ActiveFindings() {
		rule := string(finding.RuleID)
		index, known := ruleIndex[rule]
		if !known {
			index = len(run.Tool.Driver.Rules)
			ruleIndex[rule] = index
			run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, describeRule(rule, level(finding.Severity), catalog))
		}
		run.Results = append(run.Results, result(finding, index, report.Comparison))
	}
	return sarifLog{Schema: sarifSchema, Version: sarifVersion, Runs: []sarifRun{run}}
}

func describeRule(id, defaultLevel string, catalog *explain.Catalog) sarifRule {
	rule := sarifRule{
		ID:                   id,
		Name:                 id,
		ShortDescription:     sarifMessage{Text: id},
		FullDescription:      sarifMessage{Text: id},
		HelpURI:              helpURI(id),
		DefaultConfiguration: sarifConfiguration{Level: defaultLevel},
	}
	if entry, ok := catalog.Lookup(id); ok {
		rule.Name = entry.Name
		rule.ShortDescription.Text = entry.Name
		rule.FullDescription.Text = entry.Definition
	}
	return rule
}

// helpURI points a rule at its documentation page; the rule ID's first
// segment names the rule group.
func helpURI(rule string) string {
	pages := map[string]string{
		"complexity":   "rules/complexity/",
		"size":         "rules/size/",
		"duplication":  "rules/duplication/",
		"coupling":     "rules/dependencies/",
		"dependency":   "rules/dependencies/",
		"architecture": "rules/boundaries/",
		"coverage":     "rules/coverage/",
	}
	group, _, _ := strings.Cut(rule, ".")
	return product.DocumentationURL + pages[group]
}

func result(finding domain.Finding, ruleIndex int, comparison domain.Comparison) sarifResult {
	text := finding.Message
	if comparison.Mode != domain.ComparisonNone {
		text = fmt.Sprintf("%s (%s)", finding.Message, finding.Classification)
	}
	primary, related := locations(finding)
	return sarifResult{
		RuleID:              string(finding.RuleID),
		RuleIndex:           ruleIndex,
		Level:               level(finding.Severity),
		Message:             sarifMessage{Text: text},
		Locations:           primary,
		RelatedLocations:    related,
		PartialFingerprints: map[string]string{fingerprintKey: finding.ID},
		Properties: map[string]any{
			"blocking":       finding.Blocking,
			"classification": finding.Classification,
			"current":        finding.Current,
			"baseline":       finding.Baseline,
			"delta":          finding.Delta,
			"threshold":      finding.Threshold,
			"unit":           finding.Unit,
			"recheck":        finding.Recheck,
		},
	}
}

// evidenceLocations is the part of a finding's evidence that names files:
// the copies of a duplicated block, or the members of a cycle.
type evidenceLocations struct {
	Occurrences []struct {
		File      string `json:"file"`
		StartLine int    `json:"startLine"`
		EndLine   int    `json:"endLine"`
	} `json:"occurrences"`
	Modules []string `json:"modules"`
}

// locations returns the primary location of a finding and any related ones.
// Code scanning tools show a result at its first location only, so a
// duplicated block or a cycle is placed at its first file.
func locations(finding domain.Finding) (primary, related []sarifLocation) {
	if finding.Scope.File != "" {
		var region *sarifRegion
		if finding.Scope.Line > 0 {
			region = &sarifRegion{StartLine: finding.Scope.Line, EndLine: max(finding.Scope.EndLine, finding.Scope.Line)}
		}
		return []sarifLocation{location(finding.Scope.File, region)}, nil
	}
	encoded, err := json.Marshal(finding.Evidence)
	if err != nil {
		return nil, nil
	}
	var evidence evidenceLocations
	if err := json.Unmarshal(encoded, &evidence); err != nil {
		return nil, nil
	}
	var all []sarifLocation
	for _, occurrence := range evidence.Occurrences {
		all = append(all, location(occurrence.File, &sarifRegion{StartLine: occurrence.StartLine, EndLine: max(occurrence.EndLine, occurrence.StartLine)}))
	}
	for _, module := range evidence.Modules {
		all = append(all, location(module, nil))
	}
	if len(all) == 0 {
		return nil, nil
	}
	return all[:1], all[1:]
}

func location(file string, region *sarifRegion) sarifLocation {
	return sarifLocation{PhysicalLocation: sarifPhysicalLocation{
		ArtifactLocation: sarifArtifactLocation{URI: file, URIBaseID: sourceRootID},
		Region:           region,
	}}
}

// notifications reports analysis problems, so an incomplete run is visible in
// the log rather than looking like a clean one.
func notifications(diagnostics []domain.Diagnostic) []sarifNotification {
	var out []sarifNotification
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == domain.SeverityInfo {
			continue
		}
		notification := sarifNotification{
			Level:      level(diagnostic.Severity),
			Message:    sarifMessage{Text: diagnostic.Message},
			Descriptor: sarifDescriptor{ID: diagnostic.Code},
		}
		if diagnostic.File != "" {
			notification.Locations = []sarifLocation{location(diagnostic.File, nil)}
		}
		out = append(out, notification)
	}
	return out
}

func level(severity domain.Severity) string {
	switch severity {
	case domain.SeverityError:
		return "error"
	case domain.SeverityWarning:
		return "warning"
	default:
		return "note"
	}
}

// fileURI renders the analyzed root as a file URI with a trailing slash, as
// SARIF requires of a base URI. An empty root gives "".
func fileURI(root string) string {
	if root == "" {
		return ""
	}
	path := filepath.ToSlash(root)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}
