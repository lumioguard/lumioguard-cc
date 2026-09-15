// Package baseline stores named baselines in the repository state directory and
// never replaces one unless asked to.
package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// ErrExists is returned when a baseline would be overwritten without --replace.
var ErrExists = errors.New("baseline already exists")

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// Store reads and writes baseline files.
type Store struct {
	toolVersion string
	now         func() time.Time
}

// NewStore creates a Store that stamps baselines with the given tool version.
func NewStore(toolVersion string) *Store {
	return &Store{toolVersion: toolVersion, now: time.Now}
}

// Path returns the file for a baseline name after validating the name.
func (s *Store) Path(root, name string) (string, error) {
	if !validName.MatchString(name) {
		return "", errors.New("baseline name must be 1-64 letters, numbers, dots, underscores, or hyphens")
	}
	return filepath.Join(root, product.StateDirectoryName, product.BaselineDirectoryName, name+".json"), nil
}

// Load reads and minimally validates a named baseline.
func (s *Store) Load(root, name string) (*domain.Baseline, error) {
	path, err := s.Path(root, name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read baseline '%s': %w", name, err)
	}
	var loaded domain.Baseline
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, fmt.Errorf("could not read baseline '%s': %w", name, err)
	}
	if loaded.SchemaVersion != domain.BaselineSchemaVersion {
		return nil, fmt.Errorf("baseline '%s' has an unsupported or missing schemaVersion", name)
	}
	return &loaded, nil
}

// Save persists a report as a named baseline. It returns ErrExists when the
// baseline exists and replace is false.
func (s *Store) Save(root, name string, report *domain.Report, replace bool) (string, error) {
	path, err := s.Path(root, name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create baseline directory: %w", err)
	}
	entry := s.fromReport(name, report, s.now().UTC().Format(time.RFC3339Nano), s.toolVersion)
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return path, fmt.Errorf("encode baseline: %w", err)
	}
	flags := os.O_WRONLY | os.O_CREATE
	if replace {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return path, fmt.Errorf("baseline '%s' %w; pass --replace to overwrite it", name, ErrExists)
	}
	if err != nil {
		return path, fmt.Errorf("create baseline file: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return path, fmt.Errorf("write baseline: %w", err)
	}
	if err := file.Close(); err != nil {
		return path, fmt.Errorf("write baseline: %w", err)
	}
	return path, nil
}

// FromReport converts a report into an in-memory baseline, for example the
// analysis of a Git base commit. Comparison fields are stripped.
func (s *Store) FromReport(name string, report *domain.Report) *domain.Baseline {
	return s.fromReport(name, report, report.Run.CreatedAt, report.Tool.Version)
}

func (s *Store) fromReport(name string, report *domain.Report, createdAt, toolVersion string) *domain.Baseline {
	measurements := make([]domain.Measurement, len(report.Measurements))
	for i, measurement := range report.Measurements {
		measurements[i] = measurement.WithoutComparison()
	}
	findings := make([]domain.Finding, len(report.Findings))
	for i, finding := range report.Findings {
		findings[i] = finding.AsBaselineEntry()
	}
	snapshot := report.Snapshot
	snapshot.Kind = domain.SnapshotBaseline
	return &domain.Baseline{
		SchemaVersion: domain.BaselineSchemaVersion,
		Name:          name,
		CreatedAt:     createdAt,
		ToolVersion:   toolVersion,
		Snapshot:      snapshot,
		Adapters:      report.Adapters,
		Measurements:  measurements,
		Findings:      findings,
		Diagnostics:   report.Diagnostics,
	}
}
