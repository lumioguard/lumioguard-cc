package app

import (
	"context"
	"errors"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/engine"
)

// ErrIncompleteAnalysis is returned when a baseline cannot be written because
// required analysis did not complete.
var ErrIncompleteAnalysis = errors.New("required analysis was incomplete")

// CreateBaselineRequest describes a `baseline create` invocation.
type CreateBaselineRequest struct {
	Root    string
	Name    string
	Replace bool
}

// CreateBaselineResult reports where the baseline was written. Report is
// always present so callers can explain an incomplete analysis.
type CreateBaselineResult struct {
	Path   string
	Report *domain.Report
}

// BaselineService creates reviewed baselines.
type BaselineService struct {
	config    ConfigLoader
	engine    Engine
	baselines BaselineRepository
}

// NewBaselineService creates a BaselineService.
func NewBaselineService(config ConfigLoader, engine Engine, baselines BaselineRepository) *BaselineService {
	return &BaselineService{config: config, engine: engine, baselines: baselines}
}

// Create analyzes the working tree and persists the result. An incomplete
// analysis is never written as a baseline.
func (s *BaselineService) Create(ctx context.Context, req CreateBaselineRequest) (CreateBaselineResult, error) {
	loaded, err := s.config.Load(req.Root)
	if err != nil {
		return CreateBaselineResult{}, err
	}
	report, err := s.engine.Analyze(ctx, engine.Request{Root: req.Root, Config: loaded.Config})
	if err != nil {
		return CreateBaselineResult{}, err
	}
	if report.Policy.Status == domain.PolicyIncomplete {
		return CreateBaselineResult{Report: report}, ErrIncompleteAnalysis
	}
	path, err := s.baselines.Save(req.Root, req.Name, report, req.Replace)
	if err != nil {
		return CreateBaselineResult{Report: report}, err
	}
	return CreateBaselineResult{Path: path, Report: report}, nil
}
