package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/engine"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// ErrConflictingReferences rejects a check that names two comparison points.
var ErrConflictingReferences = errors.New("use either --baseline or --base, not both")

// CheckRequest describes a `check` invocation.
type CheckRequest struct {
	Root         string
	BaselineName string
	GitBase      string
}

func (r CheckRequest) validate() error {
	if r.BaselineName != "" && r.GitBase != "" {
		return ErrConflictingReferences
	}
	return nil
}

// CheckService analyzes the working tree, optionally against a baseline or Git
// reference. It never modifies source, configuration, baselines or Git state.
type CheckService struct {
	config    ConfigLoader
	engine    Engine
	baselines BaselineRepository
	git       GitReferences
	comparer  Comparer
	policy    PolicyEvaluator
}

// NewCheckService creates a CheckService.
func NewCheckService(config ConfigLoader, engine Engine, baselines BaselineRepository, git GitReferences, comparer Comparer, policy PolicyEvaluator) *CheckService {
	return &CheckService{config: config, engine: engine, baselines: baselines, git: git, comparer: comparer, policy: policy}
}

// Run executes the check and returns the report. The caller maps the policy
// status to an exit code.
func (s *CheckService) Run(ctx context.Context, req CheckRequest) (*domain.Report, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	loaded, err := s.config.Load(req.Root)
	if err != nil {
		return nil, err
	}
	request := engine.Request{Root: req.Root, Config: loaded.Config, GitBase: req.GitBase}
	if req.BaselineName != "" {
		baseline, err := s.baselines.Load(req.Root, req.BaselineName)
		if err != nil {
			return nil, err
		}
		request.Baseline = baseline
		request.BaselineName = req.BaselineName
	}
	report, err := s.engine.Analyze(ctx, request)
	if err != nil {
		return nil, err
	}
	if req.GitBase == "" {
		return report, nil
	}
	return s.compareWithGitBase(ctx, req, loaded.Config, report)
}

// compareWithGitBase analyzes an isolated export of the base commit with the
// current configuration, so the index and checkout are never touched.
func (s *CheckService) compareWithGitBase(ctx context.Context, req CheckRequest, cfg domain.Config, report *domain.Report) (*domain.Report, error) {
	reference := req.GitBase
	if report.Comparison.Reference != nil {
		reference = *report.Comparison.Reference
	}
	commit, err := s.git.RevParse(ctx, req.Root, reference)
	if err != nil {
		return nil, err
	}
	tree, err := os.MkdirTemp("", product.Name+"-base-")
	if err != nil {
		return nil, fmt.Errorf("create temporary directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tree) }()
	if err := s.git.ExportTree(ctx, req.Root, commit, tree); err != nil {
		return nil, err
	}
	baseReport, err := s.engine.Analyze(ctx, engine.Request{Root: tree, Config: cfg})
	if err != nil {
		return nil, fmt.Errorf("analyze base commit %s: %w", commit, err)
	}
	compared := s.comparer.Compare(report, s.baselines.FromReport(commit, baseReport), domain.ComparisonGit, commit)
	compared.Policy = s.policy.Result(compared.Findings, domain.HasRequiredError(compared.Diagnostics))
	return compared, nil
}
