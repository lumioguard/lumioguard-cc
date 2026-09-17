package app

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/fingerprint"
	"github.com/lumioguard/lumioguard-cc/internal/product"
)

// DefaultMaxCorrectionAttempts bounds how many times the Stop hook blocks a
// session before it gives up and reports to the user.
const DefaultMaxCorrectionAttempts = 2

// ErrConflictingHookReferences rejects a hook told to compare with two
// different references at once.
var ErrConflictingHookReferences = errors.New(
	product.BaselineEnvironmentVariable + " and " + product.BaseEnvironmentVariable + " are both set; set only one")

// StopHookInput is the JSON Claude Code passes to a Stop hook on stdin.
type StopHookInput struct {
	SessionID      string `json:"session_id"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// StopHookDecision is the JSON returned to Claude Code. A nil decision means
// "let the session stop" and prints nothing.
type StopHookDecision struct {
	Decision      string `json:"decision,omitempty"`
	Reason        string `json:"reason,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
}

type hookState struct {
	Attempts int `json:"attempts"`
}

// StopHookService implements the opt-in Claude Code Stop hook: it runs the
// check, returns bounded, evidence-based feedback and never bypasses the gate.
type StopHookService struct {
	check       *CheckService
	git         GitReferences
	maxAttempts int
	program     string
	getenv      func(string) string
	workingDir  func() (string, error)
}

// NewStopHookService creates a StopHookService with production defaults.
func NewStopHookService(check *CheckService, git GitReferences) *StopHookService {
	return &StopHookService{
		check:       check,
		git:         git,
		maxAttempts: DefaultMaxCorrectionAttempts,
		program:     invokedAs(os.Args[0]),
		getenv:      os.Getenv,
		workingDir:  os.Getwd,
	}
}

// Handle reads the hook input and returns the decision. Analysis failures
// block with the reason, still bounded by the attempt cap.
func (s *StopHookService) Handle(ctx context.Context, input io.Reader) (*StopHookDecision, error) {
	var parsed StopHookInput
	if err := json.NewDecoder(input).Decode(&parsed); err != nil {
		return &StopHookDecision{Decision: "block", Reason: fmt.Sprintf("%s hook failed: invalid hook input: %v", product.DisplayName, err)}, nil
	}
	cwd, err := s.resolveWorkingDirectory(parsed)
	if err != nil {
		return &StopHookDecision{Decision: "block", Reason: fmt.Sprintf("%s hook failed: %v", product.DisplayName, err)}, nil
	}
	stateFile := s.stateFile(cwd, parsed.SessionID)

	request, err := s.comparison(ctx, cwd)
	if err != nil {
		return &StopHookDecision{Decision: "block", Reason: fmt.Sprintf("%s hook failed: %v", product.DisplayName, err)}, nil
	}
	report, err := s.check.Run(ctx, request)
	if err != nil {
		attempts, stateErr := s.nextAttempt(stateFile)
		if stateErr != nil {
			return nil, stateErr
		}
		return s.bounded(fmt.Sprintf("Technical-debt analysis did not return a valid report. %v", err), attempts), nil
	}
	if report.Policy.Status == domain.PolicyPassed {
		if err := os.Remove(stateFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		return nil, nil
	}
	attempts, err := s.nextAttempt(stateFile)
	if err != nil {
		return nil, err
	}
	return s.bounded(feedback(report, request, s.program), attempts), nil
}

// comparison picks a named baseline, then a Git reference, then HEAD, so only
// what the session introduced can block and the loop can terminate.
func (s *StopHookService) comparison(ctx context.Context, cwd string) (CheckRequest, error) {
	request := CheckRequest{Root: cwd}
	baselineName := s.getenv(product.BaselineEnvironmentVariable)
	gitBase := s.getenv(product.BaseEnvironmentVariable)
	switch {
	case baselineName != "" && gitBase != "":
		return CheckRequest{}, ErrConflictingHookReferences
	case baselineName != "":
		request.BaselineName = baselineName
	case gitBase != "":
		request.GitBase = gitBase
	default:
		if _, err := s.git.RevParse(ctx, cwd, product.DefaultHookBase); err == nil {
			request.GitBase = product.DefaultHookBase
		}
	}
	return request, nil
}

// resolveWorkingDirectory returns the nearest directory at or above the session's
// directory that holds the configuration, stopping at the repository root. The
// agent's shell may be in a subfolder, and checking only that folder would pass
// changes that fail from the root.
func (s *StopHookService) resolveWorkingDirectory(input StopHookInput) (string, error) {
	project := s.getenv("CLAUDE_PROJECT_DIR")
	start := cmp.Or(input.CWD, project)
	if start == "" {
		wd, err := s.workingDir()
		if err != nil {
			return "", err
		}
		start = wd
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for dir := start; ; dir = filepath.Dir(dir) {
		if exists(filepath.Join(dir, product.ConfigFileName)) {
			return dir, nil
		}
		if exists(filepath.Join(dir, ".git")) || filepath.Dir(dir) == dir {
			break
		}
	}
	if project != "" {
		return filepath.Abs(project)
	}
	return start, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *StopHookService) stateFile(cwd, session string) string {
	if session == "" {
		session = "unknown"
	}
	return filepath.Join(cwd, product.StateDirectoryName, product.CacheDirectoryName, "claude-stop-hook", fingerprint.SHA256String(session)+".json")
}

func (s *StopHookService) nextAttempt(stateFile string) (int, error) {
	if err := os.MkdirAll(filepath.Dir(stateFile), 0o755); err != nil {
		return 0, fmt.Errorf("create hook state directory: %w", err)
	}
	var state hookState
	if data, err := os.ReadFile(stateFile); err == nil {
		_ = json.Unmarshal(data, &state) // A corrupt state file simply restarts the count.
	}
	state.Attempts++
	data, err := json.Marshal(state)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(stateFile, append(data, '\n'), 0o644); err != nil {
		return 0, fmt.Errorf("write hook state: %w", err)
	}
	return state.Attempts, nil
}

func (s *StopHookService) bounded(reason string, attempts int) *StopHookDecision {
	if attempts <= s.maxAttempts {
		return &StopHookDecision{Decision: "block", Reason: reason}
	}
	return &StopHookDecision{SystemMessage: fmt.Sprintf(
		"%s gate remains unresolved after %d correction attempts. Review the full report manually.",
		product.DisplayName, s.maxAttempts)}
}

// feedback builds the short prioritised message returned to the agent.
func feedback(report *domain.Report, request CheckRequest, program string) string {
	var blocking, required []string
	for _, finding := range report.Findings {
		// Only what fails the gate: listing excused debt wastes the agent's
		// attempts on work that cannot end the loop.
		if finding.BlocksGate() {
			blocking = append(blocking, fmt.Sprintf("- %s: %s", finding.Scope.Location(), finding.Message))
		}
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.IsRequiredError() {
			required = append(required, fmt.Sprintf("- %s: %s", diagnostic.Code, diagnostic.Message))
		}
	}
	lines := []string{fmt.Sprintf("Technical-debt check is %s.", report.Policy.Status)}
	lines = append(lines, capped(blocking, 5, "blocking findings")...)
	lines = append(lines, capped(required, 3, "required diagnostics")...)
	lines = append(lines, fmt.Sprintf(
		"Address the evidence above, then run `%s` again. Do not change the baseline, exclusions, or thresholds merely to pass.",
		product.RecheckCommandFor(program, request.BaselineName, request.GitBase)))
	return strings.Join(lines, "\n")
}

// capped keeps the message short but says what it left out, so the agent knows
// to read the full report.
func capped(items []string, limit int, noun string) []string {
	if len(items) <= limit {
		return items
	}
	return append(items[:limit:limit], fmt.Sprintf("- ...and %d more %s; the command below lists them all.", len(items)-limit, noun))
}

// invokedAs returns how to run this binary again. A hook often starts it by a
// full path because it is not on PATH, so the bare name would not work.
func invokedAs(arg0 string) string {
	if !strings.ContainsAny(arg0, `/\`) {
		return product.Name
	}
	program := filepath.ToSlash(arg0)
	if strings.Contains(program, " ") {
		return `"` + program + `"`
	}
	return program
}
