package app

import (
	"context"
	"runtime"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/engine"
	"github.com/lumiostack/lumioguard-cc/internal/explain"
)

// RuntimeInfo describes the executing environment.
type RuntimeInfo struct {
	Go           string `json:"go"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
}

// AdapterStatus describes one installed adapter.
type AdapterStatus struct {
	ID        string            `json:"id"`
	Version   string            `json:"version"`
	Status    string            `json:"status"`
	Languages []string          `json:"languages"`
	Analyzers map[string]string `json:"analyzers"`
}

// GitStatus reports whether Git-based comparison is possible.
type GitStatus struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
}

// DoctorReport is the `doctor` output.
type DoctorReport struct {
	Status       string          `json:"status"`
	Runtime      RuntimeInfo     `json:"runtime"`
	Config       string          `json:"config"`
	ConfigExists bool            `json:"configExists"`
	SourceFiles  int             `json:"sourceFiles"`
	Adapters     []AdapterStatus `json:"adapters"`
	Metrics      []string        `json:"metrics"`
	Git          GitStatus       `json:"git"`
}

// DoctorService reports available analyzers, configuration state and inputs.
type DoctorService struct {
	config     ConfigLoader
	discoverer engine.FileDiscoverer
	adapters   *adapter.Registry
	catalog    *explain.Catalog
	git        GitReferences
}

// NewDoctorService creates a DoctorService.
func NewDoctorService(config ConfigLoader, discoverer engine.FileDiscoverer, adapters *adapter.Registry, catalog *explain.Catalog, git GitReferences) *DoctorService {
	return &DoctorService{config: config, discoverer: discoverer, adapters: adapters, catalog: catalog, git: git}
}

// Diagnose inspects the environment and repository. An invalid configuration
// is an error because every other command would fail on it too.
func (s *DoctorService) Diagnose(ctx context.Context, root string) (*DoctorReport, error) {
	loaded, err := s.config.Load(root)
	if err != nil {
		return nil, err
	}
	discovered, err := s.discoverer.Discover(root, loaded.Config.Source)
	if err != nil {
		return nil, err
	}
	report := &DoctorReport{
		Status:       "ok",
		Runtime:      RuntimeInfo{Go: runtime.Version(), Platform: runtime.GOOS, Architecture: runtime.GOARCH},
		Config:       loaded.Path,
		ConfigExists: loaded.Exists,
		SourceFiles:  len(discovered.Files),
		Adapters:     []AdapterStatus{},
		Metrics:      s.catalog.IDs(),
	}
	for _, languageAdapter := range s.adapters.All() {
		report.Adapters = append(report.Adapters, AdapterStatus{
			ID:        languageAdapter.ID(),
			Version:   languageAdapter.Version(),
			Status:    "available",
			Languages: languageAdapter.Languages(),
			Analyzers: languageAdapter.Analyzers(),
		})
	}
	if version, err := s.git.Version(ctx); err == nil {
		report.Git = GitStatus{Available: true, Version: version}
	}
	return report, nil
}
