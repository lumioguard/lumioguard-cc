package app

import (
	"context"
	"fmt"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/explain"
	"github.com/lumioguard/lumioguard-cc/internal/glob"
	"github.com/lumioguard/lumioguard-cc/internal/product"
	"github.com/lumioguard/lumioguard-cc/internal/worklist"
)

// WorklistRequest describes a `worklist` invocation: a check plus filters.
type WorklistRequest struct {
	Check CheckRequest
	Rules []string
	Paths []string
	Top   int
}

// WorklistService runs a check and orders its findings into a worklist.
type WorklistService struct {
	check   *CheckService
	catalog *explain.Catalog
}

// NewWorklistService creates a WorklistService.
func NewWorklistService(check *CheckService, catalog *explain.Catalog) *WorklistService {
	return &WorklistService{check: check, catalog: catalog}
}

// Run validates the filters, runs the check and returns the ordered list.
func (s *WorklistService) Run(ctx context.Context, req WorklistRequest) (*worklist.Worklist, error) {
	options := worklist.Options{Top: req.Top}
	for _, rule := range req.Rules {
		if _, known := s.catalog.Lookup(rule); !known {
			return nil, fmt.Errorf("unknown rule %q; run `%s explain` for the list", rule, product.Name)
		}
		options.Rules = append(options.Rules, domain.RuleID(rule))
	}
	for _, pattern := range req.Paths {
		if !glob.Valid(pattern) {
			return nil, fmt.Errorf("invalid --path pattern %q", pattern)
		}
		options.Paths = append(options.Paths, pattern)
	}
	if req.Top < 0 {
		return nil, fmt.Errorf("--top must be 0 or more")
	}
	report, err := s.check.Run(ctx, req.Check)
	if err != nil {
		return nil, err
	}
	list := worklist.Build(report, options)
	return &list, nil
}
