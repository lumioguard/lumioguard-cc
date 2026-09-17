package engine

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/fingerprint"
)

// buildSnapshot hashes every analyzed file and the complete configuration so
// a later run can prove it compared like with like.
func buildSnapshot(root string, files []*adapter.SourceFile, cfg domain.Config, git domain.GitSnapshot) (domain.Snapshot, error) {
	fileHashes := make(map[string]string, len(files))
	for _, file := range files {
		fileHashes[file.RelativePath] = fingerprint.SHA256String(file.Code)
	}
	configHash, err := fingerprint.HashJSON(cfg)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("hash configuration: %w", err)
	}
	sourceHash, err := fingerprint.HashJSON(fileHashes)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("hash sources: %w", err)
	}
	id, err := fingerprint.HashJSON(map[string]any{"fileHashes": fileHashes, "configHash": configHash})
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("hash snapshot: %w", err)
	}
	return domain.Snapshot{
		ID:         id,
		Kind:       domain.SnapshotWorkingTree,
		Root:       root,
		SourceHash: sourceHash,
		FileHashes: fileHashes,
		ConfigHash: configHash,
		Git:        git,
	}, nil
}

func sortMeasurements(items []domain.Measurement) []domain.Measurement {
	if items == nil {
		return []domain.Measurement{}
	}
	slices.SortStableFunc(items, func(a, b domain.Measurement) int {
		return cmp.Or(
			strings.Compare(string(a.MetricID), string(b.MetricID)),
			strings.Compare(a.Scope.Key, b.Scope.Key),
		)
	})
	return items
}

func sortFindings(items []domain.Finding) []domain.Finding {
	if items == nil {
		return []domain.Finding{}
	}
	slices.SortStableFunc(items, func(a, b domain.Finding) int {
		return cmp.Or(
			strings.Compare(string(a.RuleID), string(b.RuleID)),
			strings.Compare(a.Scope.Key, b.Scope.Key),
			strings.Compare(a.ID, b.ID),
		)
	})
	return items
}

func sortDiagnostics(items []domain.Diagnostic) []domain.Diagnostic {
	if items == nil {
		return []domain.Diagnostic{}
	}
	slices.SortStableFunc(items, func(a, b domain.Diagnostic) int {
		return cmp.Or(
			strings.Compare(a.Code, b.Code),
			strings.Compare(a.File, b.File),
			strings.Compare(a.Message, b.Message),
		)
	})
	return items
}
