package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/cli"
	"github.com/lumiostack/lumioguard-cc/internal/compose"
	"github.com/lumiostack/lumioguard-cc/internal/config"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/guide"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

type run struct {
	code   int
	stdout string
	stderr string
}

func execute(t *testing.T, stdin string, args ...string) run {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := cli.New(compose.NewApplication()).Execute(context.Background(), args, strings.NewReader(stdin), &stdout, &stderr)
	return run{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func project(t *testing.T, source string, cfg domain.Config) string {
	t.Helper()
	root := t.TempDir()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".lumioguard-cc.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSample(t, root, source)
	return root
}

func writeSample(t *testing.T, root, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "sample.ts"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func parseReport(t *testing.T, stdout string) domain.Report {
	t.Helper()
	var report domain.Report
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not a JSON report: %v\n%s", err, stdout)
	}
	return report
}

func blockingCyclomatic() domain.Config {
	cfg := config.Default()
	cfg.Metrics.Cyclomatic.Threshold = 1
	cfg.Metrics.Cyclomatic.Block = true
	return cfg
}

func findingFor(report domain.Report, rule domain.RuleID) *domain.Finding {
	for i := range report.Findings {
		if report.Findings[i].RuleID == rule {
			return &report.Findings[i]
		}
	}
	return nil
}

func TestInvalidSyntaxReturnsJSONIncompleteReportAndExit2(t *testing.T) {
	root := project(t, "export function broken( {\n", config.Default())
	result := execute(t, "", "check", "--format", "json", "--root", root)
	if result.code != 2 {
		t.Fatalf("exit code = %d, stderr = %q", result.code, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("stderr must stay empty for JSON output, got %q", result.stderr)
	}
	if report := parseReport(t, result.stdout); report.Policy.Status != domain.PolicyIncomplete {
		t.Fatalf("expected incomplete policy, got %+v", report.Policy)
	}
}

func TestBaselineGateBlocksOnlyWorsenedDebt(t *testing.T) {
	root := project(t, "export function check(value: number) { if (value > 0) return 1; return 0; }", blockingCyclomatic())
	created := execute(t, "", "baseline", "create", "--name", "initial", "--root", root)
	if created.code != 0 || !strings.Contains(created.stdout, "Created baseline 'initial'") {
		t.Fatalf("baseline create failed: %+v", created)
	}
	if again := execute(t, "", "baseline", "create", "--name", "initial", "--root", root); again.code != 2 || !strings.Contains(again.stderr, "already exists") {
		t.Fatalf("baseline must not be replaced silently: %+v", again)
	}

	unchanged := execute(t, "", "check", "--baseline", "initial", "--format", "json", "--root", root)
	if unchanged.code != 0 {
		t.Fatalf("unchanged baseline debt must not block: %+v", unchanged)
	}
	if f := findingFor(parseReport(t, unchanged.stdout), domain.RuleID(domain.MetricCyclomatic)); f == nil || f.Classification != domain.ClassificationExisting {
		t.Fatalf("expected existing finding, got %+v", f)
	}

	writeSample(t, root, "export function check(value: number) { if (value > 0) return 1; if (value < 0) return -1; return 0; }")
	worsened := execute(t, "", "check", "--baseline", "initial", "--format", "json", "--root", root)
	report := parseReport(t, worsened.stdout)
	if worsened.code != 1 || report.Policy.Status != domain.PolicyFailed {
		t.Fatalf("worsened blocking debt must fail: code=%d policy=%+v", worsened.code, report.Policy)
	}
	f := findingFor(report, domain.RuleID(domain.MetricCyclomatic))
	if f == nil || f.Classification != domain.ClassificationWorsened || *f.Baseline != 2 || *f.Current != 3 {
		t.Fatalf("unexpected finding %+v", f)
	}
}

func TestConfigurationChangeMakesBaselineIncomparable(t *testing.T) {
	root := project(t, "export const value = () => 1;", config.Default())
	if created := execute(t, "", "baseline", "create", "--name", "initial", "--root", root); created.code != 0 {
		t.Fatalf("baseline create failed: %+v", created)
	}
	changed := config.Default()
	changed.Metrics.Cyclomatic.Threshold = 2
	data, _ := json.Marshal(changed)
	if err := os.WriteFile(filepath.Join(root, ".lumioguard-cc.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	result := execute(t, "", "check", "--baseline", "initial", "--format", "json", "--root", root)
	report := parseReport(t, result.stdout)
	if result.code != 2 || report.Comparison.Comparable || report.Policy.Status != domain.PolicyIncomplete {
		t.Fatalf("expected incomparable incomplete result, got code=%d %+v", result.code, report.Comparison)
	}
}

// gitFixture makes root a Git repository whose single commit holds whatever is
// already on disk, and returns a runner for further git commands.
func gitFixture(t *testing.T, root string) func(args ...string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	git("init", "-q")
	git("config", "user.email", "fixture@example.invalid")
	git("config", "user.name", "Fixture")
	git("add", ".")
	git("commit", "-q", "-m", "baseline")
	return git
}

func TestGitBaseComparisonDoesNotModifyCheckout(t *testing.T) {
	root := project(t, "export function check(value: number) { return value; }", blockingCyclomatic())
	git := gitFixture(t, root)
	if before := strings.TrimSpace(git("status", "--porcelain")); before != "" {
		t.Fatalf("fixture must start clean: %q", before)
	}
	writeSample(t, root, "export function check(value: number) { if (value > 0) return 1; return 0; }")

	result := execute(t, "", "check", "--base", "HEAD", "--format", "json", "--root", root)
	report := parseReport(t, result.stdout)
	if result.code != 1 || report.Comparison.Mode != domain.ComparisonGit || !report.Comparison.Comparable {
		t.Fatalf("expected failed git comparison, got code=%d comparison=%+v diagnostics=%+v", result.code, report.Comparison, report.Diagnostics)
	}
	if after := strings.TrimSpace(git("status", "--porcelain")); after != "M sample.ts" {
		t.Fatalf("checkout changed unexpectedly: %q", after)
	}
	if f := findingFor(report, domain.RuleID(domain.MetricCyclomatic)); f == nil || f.Classification != domain.ClassificationNew {
		t.Fatalf("expected a new finding against the base commit, got %+v", f)
	}
}

// With no stored baseline and no configured reference, the hook compares with
// the last commit, so only the session's own regressions block it.
func TestClaudeStopHookComparesWithLastCommitWithoutABaselineFile(t *testing.T) {
	root := project(t, "export function check(value: number) { if (value > 0) return 1; return 0; }", blockingCyclomatic())
	gitFixture(t, root)
	input := `{"session_id":"no-baseline","cwd":` + quote(root) + `,"hook_event_name":"Stop","stop_hook_active":false}`

	inherited := execute(t, input, "hook", "claude-stop")
	if inherited.code != 0 || strings.TrimSpace(inherited.stdout) != "" {
		t.Fatalf("committed debt must not block the session: %+v", inherited)
	}
	if _, err := os.Stat(filepath.Join(root, product.StateDirectoryName, product.BaselineDirectoryName)); !os.IsNotExist(err) {
		t.Fatalf("the loop must need no baseline file, got %v", err)
	}

	writeSample(t, root, "export function check(a: number, b: number) { if (a > 0) { if (b > 0) return 1; } return 0; }")
	blocked := execute(t, input, "hook", "claude-stop")
	var decision map[string]any
	if err := json.Unmarshal([]byte(blocked.stdout), &decision); err != nil {
		t.Fatalf("hook output is not JSON: %v\n%s", err, blocked.stdout)
	}
	if decision["decision"] != "block" {
		t.Fatalf("a regression the session introduced must block: %+v", decision)
	}
	reason, _ := decision["reason"].(string)
	if !strings.Contains(reason, "complexity.cyclomatic") {
		t.Fatalf("feedback must name the regression: %q", reason)
	}
	if !strings.Contains(reason, "check --base "+product.DefaultHookBase) {
		t.Fatalf("feedback must tell the agent how to reproduce the numbers: %q", reason)
	}
}

// The agent's shell may have moved into a subfolder; the hook must still use the
// project's configuration instead of passing with defaults.
func TestClaudeStopHookFindsConfigurationAboveSessionDirectory(t *testing.T) {
	root := project(t, "export function check(value: number) { if (value > 0) return 1; return 0; }", blockingCyclomatic())
	nested := filepath.Join(root, "src", "deep")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	input := `{"session_id":"nested","cwd":` + quote(nested) + `,"hook_event_name":"Stop","stop_hook_active":false}`

	result := execute(t, input, "hook", "claude-stop")
	var decision map[string]any
	if err := json.Unmarshal([]byte(result.stdout), &decision); err != nil {
		t.Fatalf("hook from a subfolder must still check the project: %v\n%+v", err, result)
	}
	if reason, _ := decision["reason"].(string); decision["decision"] != "block" || !strings.Contains(reason, "complexity.cyclomatic") {
		t.Fatalf("expected the root configuration to block, got %+v", decision)
	}
}

func TestClaudeStopHookRejectsTwoComparisonReferences(t *testing.T) {
	root := project(t, "export const value = 1;", blockingCyclomatic())
	t.Setenv(product.BaselineEnvironmentVariable, "reviewed")
	t.Setenv(product.BaseEnvironmentVariable, "main")
	input := `{"session_id":"conflict","cwd":` + quote(root) + `,"hook_event_name":"Stop","stop_hook_active":false}`

	result := execute(t, input, "hook", "claude-stop")
	var decision map[string]any
	if err := json.Unmarshal([]byte(result.stdout), &decision); err != nil {
		t.Fatalf("hook output is not JSON: %v\n%s", err, result.stdout)
	}
	reason, _ := decision["reason"].(string)
	if decision["decision"] != "block" || !strings.Contains(reason, "both set") {
		t.Fatalf("two references must be reported, not silently resolved: %+v", decision)
	}
}

func TestClaudeStopHookCapsBlockingAtTwoAttempts(t *testing.T) {
	root := project(t, "export function check(value: number) { if (value > 0) return 1; return 0; }", blockingCyclomatic())
	input := `{"session_id":"test-session","cwd":` + quote(root) + `,"hook_event_name":"Stop","stop_hook_active":false}`
	var decisions []map[string]any
	for attempt := 0; attempt < 3; attempt++ {
		result := execute(t, input, "hook", "claude-stop")
		if result.code != 0 {
			t.Fatalf("hook must exit 0, got %+v", result)
		}
		var decision map[string]any
		if err := json.Unmarshal([]byte(result.stdout), &decision); err != nil {
			t.Fatalf("hook output is not JSON: %v\n%s", err, result.stdout)
		}
		decisions = append(decisions, decision)
	}
	if decisions[0]["decision"] != "block" || decisions[1]["decision"] != "block" {
		t.Fatalf("first two attempts must block: %+v", decisions)
	}
	if _, blocked := decisions[2]["decision"]; blocked {
		t.Fatalf("third attempt must not block: %+v", decisions[2])
	}
	if message, _ := decisions[2]["systemMessage"].(string); !strings.Contains(message, "after 2 correction attempts") {
		t.Fatalf("expected unresolved system message, got %+v", decisions[2])
	}

	writeSample(t, root, "export const value = 1;")
	passed := execute(t, input, "hook", "claude-stop")
	if passed.code != 0 || strings.TrimSpace(passed.stdout) != "" {
		t.Fatalf("a passing gate must print nothing, got %+v", passed)
	}
}

func TestInitDoctorExplainAndVersion(t *testing.T) {
	root := t.TempDir()
	if result := execute(t, "", "init", "--root", root); result.code != 0 || !strings.Contains(result.stdout, "Created") {
		t.Fatalf("init failed: %+v", result)
	}
	if result := execute(t, "", "init", "--root", root); result.code != 2 || !strings.Contains(result.stderr, "already exists") {
		t.Fatalf("init must not overwrite: %+v", result)
	}
	doctor := execute(t, "", "doctor", "--format", "json", "--root", root)
	if doctor.code != 0 {
		t.Fatalf("doctor failed: %+v", doctor)
	}
	var diagnosis map[string]any
	if err := json.Unmarshal([]byte(doctor.stdout), &diagnosis); err != nil || diagnosis["status"] != "ok" {
		t.Fatalf("doctor output invalid: %v %s", err, doctor.stdout)
	}
	if explain := execute(t, "", "explain", "complexity.cyclomatic"); explain.code != 0 || !strings.Contains(explain.stdout, "Cyclomatic complexity") {
		t.Fatalf("explain failed: %+v", explain)
	}
	if unknown := execute(t, "", "explain", "nope"); unknown.code != 2 || !strings.Contains(unknown.stderr, "unknown metric") {
		t.Fatalf("unknown metric must fail: %+v", unknown)
	}
	if version := execute(t, "", "--version"); version.code != 0 || strings.TrimSpace(version.stdout) == "" {
		t.Fatalf("version failed: %+v", version)
	}
	if bad := execute(t, "", "check", "--format", "xml", "--root", root); bad.code != 2 {
		t.Fatalf("invalid format must fail: %+v", bad)
	}
	if bad := execute(t, "", "check", "--baseline", "initial", "--base", "HEAD", "--root", root); bad.code != 2 || !strings.Contains(bad.stderr, "use either") {
		t.Fatalf("conflicting comparison references must fail: %+v", bad)
	}
	if bad := execute(t, "", "nonsense"); bad.code != 2 {
		t.Fatalf("unknown command must fail: %+v", bad)
	}
}

func TestSarifOutputListsActiveFindingsWithLocations(t *testing.T) {
	root := project(t, "export function check(value: number) { if (value > 0) return 1; return 0; }", blockingCyclomatic())
	result := execute(t, "", "check", "--format", "sarif", "--root", root)
	if result.code != 1 || result.stderr != "" {
		t.Fatalf("expected a failed check with an empty stderr, got %+v", result)
	}
	var log struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name  string `json:"name"`
					Rules []struct {
						ID      string `json:"id"`
						HelpURI string `json:"helpUri"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
				PartialFingerprints map[string]string `json:"partialFingerprints"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &log); err != nil {
		t.Fatalf("stdout is not a SARIF log: %v\n%s", err, result.stdout)
	}
	if log.Version != "2.1.0" || len(log.Runs) != 1 || log.Runs[0].Tool.Driver.Name != product.Name {
		t.Fatalf("unexpected log header %+v", log)
	}
	run := log.Runs[0]
	if len(run.Tool.Driver.Rules) != 1 || run.Tool.Driver.Rules[0].ID != string(domain.MetricCyclomatic) || !strings.HasPrefix(run.Tool.Driver.Rules[0].HelpURI, product.DocumentationURL+"rules/") {
		t.Fatalf("unexpected rules %+v", run.Tool.Driver.Rules)
	}
	if len(run.Results) != 1 || run.Results[0].RuleID != string(domain.MetricCyclomatic) || run.Results[0].Level != "warning" {
		t.Fatalf("unexpected results %+v", run.Results)
	}
	location := run.Results[0].Locations
	if len(location) != 1 || location[0].PhysicalLocation.ArtifactLocation.URI != "sample.ts" || location[0].PhysicalLocation.Region.StartLine != 1 {
		t.Fatalf("unexpected location %+v", location)
	}
	if run.Results[0].PartialFingerprints["lumioguardFindingId/v1"] == "" {
		t.Fatalf("results need a stable fingerprint: %+v", run.Results[0])
	}
	if other := execute(t, "", "explain", "complexity.cyclomatic", "--format", "sarif"); other.code != 2 || !strings.Contains(other.stderr, "only supported by check") {
		t.Fatalf("sarif must be refused outside check: %+v", other)
	}
}

func TestGuideListsAndPrintsTopics(t *testing.T) {
	list := execute(t, "", "guide")
	if list.code != 0 || !strings.Contains(list.stdout, "cleanup") {
		t.Fatalf("guide must list topics: %+v", list)
	}
	if topic := execute(t, "", "guide", "check"); topic.code != 0 || !strings.Contains(topic.stdout, "check --base HEAD --format json") {
		t.Fatalf("guide check failed: %+v", topic)
	}
	if unknown := execute(t, "", "guide", "nope"); unknown.code != 2 || !strings.Contains(unknown.stderr, "unknown guide") {
		t.Fatalf("unknown guide must fail: %+v", unknown)
	}
}

// Guides are instructions agents follow literally, so every command and flag they
// mention must exist.
func TestGuidesMentionOnlyRealCommandsAndFlags(t *testing.T) {
	var topics []guide.Topic
	if err := json.Unmarshal([]byte(execute(t, "", "guide", "--format", "json").stdout), &topics); err != nil {
		t.Fatal(err)
	}
	usage := regexp.MustCompile("lumioguard-cc ([^`\"\n|]*)")
	word := regexp.MustCompile(`^[a-z][a-z-]*$`)
	for _, listed := range topics {
		text := execute(t, "", "guide", listed.Name).stdout
		for _, match := range usage.FindAllStringSubmatch(text, -1) {
			// Command words come first, then flags; an argument such as <rule-id> ends the words.
			var words, flags []string
			for _, token := range strings.Fields(match[1]) {
				if strings.HasPrefix(token, "--") {
					flags = append(flags, strings.SplitN(token, "=", 2)[0])
				} else if len(flags) == 0 && word.MatchString(token) {
					words = append(words, token)
				} else if len(flags) == 0 {
					break
				}
			}
			help := execute(t, "", append(words, "--help")...)
			if help.code != 0 {
				t.Errorf("guide %s mentions `lumioguard-cc %s`, which is not a command: %s", listed.Name, match[1], help.stderr)
				continue
			}
			for _, flag := range flags {
				if !strings.Contains(help.stdout, flag+" ") {
					t.Errorf("guide %s mentions %s for `lumioguard-cc %s`, which has no such flag", listed.Name, flag, strings.Join(words, " "))
				}
			}
		}
	}
}

func quote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
