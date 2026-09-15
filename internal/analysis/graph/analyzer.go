package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/fingerprint"
	"github.com/lumiostack/lumioguard-cc/internal/glob"
)

const (
	// Name identifies the analyzer in error messages.
	Name = "dependency-graph"
	// Variant is the exact fan-in/fan-out algorithm identifier.
	Variant = "distinct-resolved-internal-modules-v1"
)

// Analyzer implements analysis.RepositoryAnalyzer for coupling, cycles and
// architecture boundaries.
type Analyzer struct {
	recheck string
}

// New creates the analyzer; recheck is the command shown in findings.
func New(recheck string) *Analyzer {
	return &Analyzer{recheck: recheck}
}

// Name implements analysis.RepositoryAnalyzer.
func (a *Analyzer) Name() string {
	return Name
}

// Analyze implements analysis.RepositoryAnalyzer.
func (a *Analyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	g := build(in)
	result := analysis.Result{Diagnostics: g.diagnostics}
	result.Measurements = couplingMeasurements(g)
	result.Findings = append(result.Findings, a.cycleFindings(g, in.Config.Architecture)...)
	result.Findings = append(result.Findings, a.boundaryFindings(g, in.Config.Architecture)...)
	return result, nil
}

func couplingMeasurements(g *moduleGraph) []domain.Measurement {
	fanIn := g.fanIn()
	measurements := make([]domain.Measurement, 0, 2*len(g.modules))
	for _, module := range g.modules {
		dependencies := g.targets(module)
		measurements = append(measurements,
			domain.Measurement{
				MetricID: domain.MetricModuleFanOut,
				Variant:  Variant,
				Category: domain.CategoryCoupling,
				Scope:    domain.FileScope(module),
				Value:    domain.Float(float64(len(dependencies))),
				Unit:     domain.UnitCount,
				Status:   domain.StatusMeasured,
				Evidence: domain.Evidence{"dependencies": dependencies},
			},
			domain.Measurement{
				MetricID: domain.MetricModuleFanIn,
				Variant:  Variant,
				Category: domain.CategoryCoupling,
				Scope:    domain.FileScope(module),
				Value:    domain.Float(float64(fanIn[module])),
				Unit:     domain.UnitCount,
				Status:   domain.StatusMeasured,
			},
		)
	}
	return measurements
}

func (a *Analyzer) cycleFindings(g *moduleGraph, architecture domain.ArchitectureConfig) []domain.Finding {
	var findings []domain.Finding
	for _, component := range stronglyConnectedComponents(g.modules, g.edges) {
		key := strings.Join(component, "|")
		evidence := domain.Evidence{"modules": component}
		findings = append(findings, domain.Finding{
			ID:             findingID(domain.RuleDependencyCycle, key, evidence),
			RuleID:         domain.RuleDependencyCycle,
			Title:          "Dependency cycle",
			Severity:       domain.SeverityError,
			Blocking:       architecture.BlockCycles,
			Scope:          domain.CycleScope(key),
			Message:        fmt.Sprintf("A dependency cycle connects %s", strings.Join(component, ", ")),
			Classification: domain.ClassificationNew,
			Current:        domain.Float(1),
			Threshold:      domain.Float(0),
			Unit:           domain.UnitViolation,
			Evidence:       evidence,
			Recheck:        a.recheck,
		})
	}
	return findings
}

func (a *Analyzer) boundaryFindings(g *moduleGraph, architecture domain.ArchitectureConfig) []domain.Finding {
	if len(architecture.Boundaries) == 0 {
		return nil
	}
	owner := newBoundaryOwner(architecture.Boundaries)
	var findings []domain.Finding
	for _, source := range g.modules {
		sourceBoundary, owned := owner.of(source)
		if !owned {
			continue
		}
		for _, target := range g.targets(source) {
			targetBoundary, owned := owner.of(target)
			if !owned || sourceBoundary.Allows(targetBoundary.Name) {
				continue
			}
			line := g.edgeLines[edgeKey{source: source, target: target}]
			evidence := domain.Evidence{
				"source":         source,
				"sourceBoundary": sourceBoundary.Name,
				"target":         target,
				"targetBoundary": targetBoundary.Name,
				"line":           line,
			}
			key := source + "->" + target
			findings = append(findings, domain.Finding{
				ID:             findingID(domain.RuleBoundaryViolation, key, evidence),
				RuleID:         domain.RuleBoundaryViolation,
				Title:          "Architecture boundary violation",
				Severity:       domain.SeverityError,
				Blocking:       architecture.BlockViolations,
				Scope:          domain.DependencyScope(source, line, key),
				Message:        fmt.Sprintf("%s is not allowed to depend on %s", sourceBoundary.Name, targetBoundary.Name),
				Classification: domain.ClassificationNew,
				Current:        domain.Float(1),
				Threshold:      domain.Float(0),
				Unit:           domain.UnitViolation,
				Evidence:       evidence,
				Recheck:        a.recheck,
			})
		}
	}
	return findings
}

// boundaryOwner finds the boundary that owns a module and caches the answer,
// because the same module is looked up once per incoming edge.
type boundaryOwner struct {
	boundaries []domain.Boundary
	known      map[string]ownership
}

type ownership struct {
	boundary domain.Boundary
	owned    bool
}

func newBoundaryOwner(boundaries []domain.Boundary) *boundaryOwner {
	return &boundaryOwner{boundaries: boundaries, known: map[string]ownership{}}
}

// of returns the first declared boundary whose include patterns match.
func (o *boundaryOwner) of(file string) (domain.Boundary, bool) {
	if hit, seen := o.known[file]; seen {
		return hit.boundary, hit.owned
	}
	var hit ownership
	for _, boundary := range o.boundaries {
		if glob.MatchesAny(file, boundary.Include) {
			hit = ownership{boundary: boundary, owned: true}
			break
		}
	}
	o.known[file] = hit
	return hit.boundary, hit.owned
}

func findingID(rule domain.RuleID, key string, evidence domain.Evidence) string {
	encoded, err := fingerprint.StableJSON(evidence)
	if err != nil {
		encoded = []byte(fmt.Sprint(evidence))
	}
	return fingerprint.ShortID(string(rule), key, string(encoded))
}
