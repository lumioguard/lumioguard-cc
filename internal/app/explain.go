package app

import (
	"fmt"

	"github.com/lumiostack/lumioguard-cc/internal/explain"
)

// ExplainService looks up metric and rule definitions.
type ExplainService struct {
	catalog *explain.Catalog
}

// NewExplainService creates an ExplainService.
func NewExplainService(catalog *explain.Catalog) *ExplainService {
	return &ExplainService{catalog: catalog}
}

// Explain returns the catalog entry for an identifier.
func (s *ExplainService) Explain(id string) (explain.Entry, error) {
	entry, ok := s.catalog.Lookup(id)
	if !ok {
		return explain.Entry{}, fmt.Errorf("unknown metric '%s'", id)
	}
	return entry, nil
}
