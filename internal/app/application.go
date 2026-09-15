package app

import (
	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/engine"
	"github.com/lumiostack/lumioguard-cc/internal/explain"
)

// Dependencies are the collaborators required to build an Application.
type Dependencies struct {
	Tool       domain.ToolInfo
	Config     ConfigLoader
	Engine     Engine
	Baselines  BaselineRepository
	Git        GitReferences
	Comparer   Comparer
	Policy     PolicyEvaluator
	Adapters   *adapter.Registry
	Discoverer engine.FileDiscoverer
	Catalog    *explain.Catalog
}

// Application groups the use-case services.
type Application struct {
	Tool     domain.ToolInfo
	Check    *CheckService
	Baseline *BaselineService
	Doctor   *DoctorService
	Init     *InitService
	Explain  *ExplainService
	Guide    *GuideService
	StopHook *StopHookService
}

// New wires the services from their dependencies.
func New(deps Dependencies) *Application {
	check := NewCheckService(deps.Config, deps.Engine, deps.Baselines, deps.Git, deps.Comparer, deps.Policy)
	return &Application{
		Tool:     deps.Tool,
		Check:    check,
		Baseline: NewBaselineService(deps.Config, deps.Engine, deps.Baselines),
		Doctor:   NewDoctorService(deps.Config, deps.Discoverer, deps.Adapters, deps.Catalog, deps.Git),
		Init:     NewInitService(deps.Config),
		Explain:  NewExplainService(deps.Catalog),
		Guide:    NewGuideService(deps.Tool.Version),
		StopHook: NewStopHookService(check, deps.Git),
	}
}
