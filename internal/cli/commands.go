package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lumioguard/lumioguard-cc/internal/app"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/product"
	"github.com/lumioguard/lumioguard-cc/internal/report"
)

func (c *CLI) initCommand(s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create " + product.ConfigFileName + " with documented defaults (never overwrites)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			path, err := c.app.Init.Initialize(s.root)
			if err != nil {
				return err
			}
			fmt.Fprintf(s.stdout, "Created %s\n", path)
			return nil
		},
	}
}

func (c *CLI) doctorCommand(s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Show available adapters, metrics, configuration state and inputs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := c.app.Doctor.Diagnose(cmd.Context(), s.root)
			if err != nil {
				return err
			}
			if s.format == report.FormatJSON {
				return report.WriteJSON(s.stdout, result)
			}
			configState := "loaded"
			if !result.ConfigExists {
				configState = "not found, defaults in use"
			}
			fmt.Fprintf(s.stdout, "Environment OK\nGo %s %s/%s\nConfiguration: %s (%s)\n%d source files detected\n",
				result.Runtime.Go, result.Runtime.Platform, result.Runtime.Architecture, result.Config, configState, result.SourceFiles)
			for _, adapter := range result.Adapters {
				fmt.Fprintf(s.stdout, "Adapter %s %s: %s\n", adapter.ID, adapter.Version, adapter.Status)
			}
			if result.Git.Available {
				fmt.Fprintf(s.stdout, "Git: available (%s)\n", result.Git.Version)
			} else {
				fmt.Fprintln(s.stdout, "Git: not available; --base comparisons will fail")
			}
			return nil
		},
	}
}

func (c *CLI) checkCommand(s *session) *cobra.Command {
	var baselineName, gitBase string
	cmd := &cobra.Command{
		Use:   "check [--baseline NAME | --base REF]",
		Short: "Analyze the working tree and evaluate the configured gate",
		Long: "Analyze the current working tree without modifying source, configuration, baselines or Git state.\n\n" +
			"--baseline NAME compares with a named baseline under " + product.StateDirectoryName + "/" + product.BaselineDirectoryName + ".\n" +
			"--base REF compares with a Git reference. The comparison point is the merge base of HEAD and REF,\n" +
			"so `--base main` measures what the current branch introduced since it diverged from main, even if\n" +
			"main has moved on. The base commit is analyzed from an isolated archive; the checkout is untouched.\n" +
			"Untracked files count as changed for changed-code coverage.\n\n" +
			"Existing baseline debt stays visible but does not block; new and worsened blocking findings do.\n" +
			"Incomplete required analysis (exit 2) takes precedence over policy failure (exit 1).\n\n" +
			"Agents should read --format json: the human summary lists at most ten findings.\n" +
			"See `" + product.Name + " guide check` and `" + product.Name + " guide report`.",
		Example: "  " + product.Name + " check\n" +
			"  " + product.Name + " check --base HEAD --format json\n" +
			"  " + product.Name + " check --base main",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := c.app.Check.Run(cmd.Context(), app.CheckRequest{
				Root:         s.root,
				BaselineName: baselineName,
				GitBase:      gitBase,
			})
			if err != nil {
				return err
			}
			if err := report.NewRenderer(s.format).Render(s.stdout, result); err != nil {
				return err
			}
			if code := result.ExitCode(); code != domain.ExitPassed {
				return &exitError{code: code}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&baselineName, "baseline", "", "named baseline to compare with")
	cmd.Flags().StringVar(&gitBase, "base", "", "Git reference to compare with (merge base of HEAD and REF)")
	return cmd
}

func (c *CLI) worklistCommand(s *session) *cobra.Command {
	var baselineName, gitBase string
	var rules, paths []string
	var top int
	cmd := &cobra.Command{
		Use:   "worklist [--base REF | --baseline NAME] [--rule ID]... [--path GLOB]... [--top N]",
		Short: "Order the places to fix: structure, then hotspots, then duplicated blocks",
		Long: "Run a check and list where to start a cleanup. Findings are grouped by place and ordered:\n\n" +
			"  1. dependency cycles and boundary violations;\n" +
			"  2. hotspots: functions and files, the ones breaking the most rules first, then the ones\n" +
			"     furthest over their own limit. A function inside another function is listed under it;\n" +
			"  3. duplicated blocks, the ones taking the most lines first.\n\n" +
			"The order is a place to start, not a score. --base and --baseline work as for `check` and add each\n" +
			"place's classification. Exit code 2 means the analysis was incomplete, so the list is too.",
		Example: "  " + product.Name + " worklist\n" +
			"  " + product.Name + " worklist --rule complexity.cognitive --top 5\n" +
			"  " + product.Name + " worklist --path 'src/api/**' --format json",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			list, err := c.app.Worklist.Run(cmd.Context(), app.WorklistRequest{
				Check: app.CheckRequest{Root: s.root, BaselineName: baselineName, GitBase: gitBase},
				Rules: rules,
				Paths: paths,
				Top:   top,
			})
			if err != nil {
				return err
			}
			if s.format == report.FormatJSON {
				if err := report.WriteJSON(s.stdout, list); err != nil {
					return err
				}
			} else if _, err := fmt.Fprintln(s.stdout, report.WorklistText(list)); err != nil {
				return err
			}
			if list.Policy.Status == domain.PolicyIncomplete {
				return &exitError{code: domain.ExitIncomplete}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&baselineName, "baseline", "", "named baseline to compare with")
	cmd.Flags().StringVar(&gitBase, "base", "", "Git reference to compare with (merge base of HEAD and REF)")
	cmd.Flags().StringArrayVar(&rules, "rule", nil, "keep only this rule; repeat for several")
	cmd.Flags().StringArrayVar(&paths, "path", nil, "keep only files matching this glob; repeat for several")
	cmd.Flags().IntVar(&top, "top", 20, "places to show per section; 0 shows all")
	return cmd
}

func (c *CLI) baselineCommand(s *session) *cobra.Command {
	baseline := &cobra.Command{
		Use:   "baseline",
		Short: "Manage named baselines",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	var name string
	var replace bool
	create := &cobra.Command{
		Use:   "create --name NAME [--replace]",
		Short: "Analyze the working tree and store the result as a reviewed baseline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := c.app.Baseline.Create(cmd.Context(), app.CreateBaselineRequest{Root: s.root, Name: name, Replace: replace})
			if errors.Is(err, app.ErrIncompleteAnalysis) {
				text := report.HumanRenderer{}.Text(result.Report)
				return &exitError{code: domain.ExitIncomplete, message: text + "\nBaseline was not written because required analysis was incomplete."}
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(s.stdout, "Created baseline '%s' at %s\n", name, result.Path)
			return nil
		},
	}
	create.Flags().StringVar(&name, "name", "", "baseline name (letters, numbers, dots, underscores, hyphens)")
	create.Flags().BoolVar(&replace, "replace", false, "replace an existing baseline with the same name")
	_ = create.MarkFlagRequired("name")
	baseline.AddCommand(create)
	return baseline
}

func (c *CLI) explainCommand(s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "explain METRIC_ID",
		Short: "Show the definition, variant, limitations and source of a metric or rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			entry, err := c.app.Explain.Explain(args[0])
			if err != nil {
				return err
			}
			if s.format == report.FormatJSON {
				return report.WriteJSON(s.stdout, entry)
			}
			fmt.Fprintf(s.stdout, "%s (%s)\n%s\nVariant: %s\nLimitations: %s\nSource: %s\n",
				entry.Name, entry.Classification, entry.Definition, entry.Variant, entry.Limitations, entry.Source)
			return nil
		},
	}
}

func (c *CLI) guideCommand(s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "guide [TOPIC]",
		Short: "Show step-by-step guides for setup, checking changes and cleaning up code",
		Long:  "Prints a task guide written for people and coding agents. Run it without a topic to list them.",
		Example: "  " + product.Name + " guide\n" +
			"  " + product.Name + " guide check",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				topics := c.app.Guide.Topics()
				if s.format == report.FormatJSON {
					return report.WriteJSON(s.stdout, topics)
				}
				fmt.Fprintln(s.stdout, "Guides:")
				for _, topic := range topics {
					fmt.Fprintf(s.stdout, "  %-8s %s\n", topic.Name, topic.Summary)
				}
				fmt.Fprintf(s.stdout, "\nRun `%s guide TOPIC` to read one.\n", product.Name)
				return nil
			}
			topic, err := c.app.Guide.Topic(args[0])
			if err != nil {
				return err
			}
			if s.format == report.FormatJSON {
				return report.WriteJSON(s.stdout, topic)
			}
			fmt.Fprint(s.stdout, topic.Text)
			return nil
		},
	}
}

func (c *CLI) hookCommand(s *session) *cobra.Command {
	hook := &cobra.Command{
		Use:   "hook",
		Short: "Coding-agent integrations (opt-in)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	claudeStop := &cobra.Command{
		Use:   "claude-stop",
		Short: "Claude Code Stop hook: reads hook JSON on stdin and prints a bounded decision",
		Long: "Runs the check for the project that holds " + product.ConfigFileName + ", found from the session's\n" +
			"working directory upwards. When the gate did not pass, it returns a blocking decision with\n" +
			"evidence. After " + fmt.Sprint(app.DefaultMaxCorrectionAttempts) + " failed correction attempts per session it stops blocking and\n" +
			"reports the unresolved state to the user.\n\n" +
			"It compares with " + product.DefaultHookBase + ". Set " + product.BaseEnvironmentVariable + " to a Git reference or " +
			product.BaselineEnvironmentVariable + "\nto a stored baseline to compare with something else. See `" + product.Name + " guide setup`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			decision, err := c.app.StopHook.Handle(cmd.Context(), s.stdin)
			if err != nil {
				return err
			}
			if decision == nil {
				return nil
			}
			return report.WriteJSON(s.stdout, decision)
		},
	}
	hook.AddCommand(claudeStop)
	return hook
}

func (c *CLI) versionCommand(s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the tool version",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Fprintln(s.stdout, c.app.Tool.Version)
			return nil
		},
	}
}
