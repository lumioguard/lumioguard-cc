// Package product holds the product identity and on-disk conventions shared by
// every layer: names, file locations and the recheck command shown to agents.
package product

const (
	// Name is the executable and tool name recorded in reports.
	Name = "lumioguard-cc"
	// DisplayName is the human-facing product name. CC stands for Crap Cleaner.
	DisplayName = "lumioguard CC"
	// Tagline is a one-line description used in help output.
	Tagline = "keeps AI-written code clean as the project grows"
	// RepositoryURL is the public source repository.
	RepositoryURL = "https://github.com/lumiostack/lumioguard-cc"
	// DocumentationURL is the published documentation site, with a trailing slash.
	DocumentationURL = "https://lumiostack.github.io/lumioguard-cc/"

	// ConfigFileName is the repository-level configuration file.
	ConfigFileName = ".lumioguard-cc.json"
	// StateDirectoryName holds baselines and caches inside a repository.
	StateDirectoryName = ".lumioguard-cc"
	// BaselineDirectoryName is the baseline folder inside StateDirectoryName.
	BaselineDirectoryName = "baselines"
	// CacheDirectoryName is the cache folder inside StateDirectoryName.
	CacheDirectoryName = "cache"

	// RecheckCommand is the command agents are told to run after acting on
	// evidence, when the check used no comparison reference.
	RecheckCommand = "lumioguard-cc check --format json"
	// BaselineEnvironmentVariable names a stored baseline for the Claude Code
	// Stop hook to compare with.
	BaselineEnvironmentVariable = "LUMIOGUARD_CC_BASELINE"
	// BaseEnvironmentVariable names a Git reference for the Claude Code Stop
	// hook to compare with, so an agent loop needs no baseline file at all.
	BaseEnvironmentVariable = "LUMIOGUARD_CC_BASE"
	// DefaultHookBase is the Stop hook's reference when neither variable is set,
	// so an agent is judged on its own session, not on inherited debt.
	DefaultHookBase = "HEAD"
)

// RecheckCommandFor returns the command that reproduces a check with program,
// including its comparison reference, because the same tree scores differently
// against each.
func RecheckCommandFor(program, baselineName, gitBase string) string {
	switch {
	case baselineName != "":
		return program + " check --baseline " + baselineName + " --format json"
	case gitBase != "":
		return program + " check --base " + gitBase + " --format json"
	default:
		return program + " check --format json"
	}
}

// Version is the tool version. Release builds may override it with
// -ldflags "-X github.com/lumiostack/lumioguard-cc/internal/product.Version=x.y.z".
var Version = "0.1.0"
