// Package cli defines the command-line interface. Commands parse arguments,
// call one application service, render the result and map the exit code.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lumioguard/lumioguard-cc/internal/app"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/product"
	"github.com/lumioguard/lumioguard-cc/internal/report"
)

// exitError carries an exit code without a message (the output was already
// written) or with a message printed to stderr.
type exitError struct {
	code    int
	message string
}

func (e *exitError) Error() string {
	return e.message
}

// CLI builds and executes the command tree.
type CLI struct {
	app *app.Application
}

// New creates a CLI over an Application.
func New(application *app.Application) *CLI {
	return &CLI{app: application}
}

// session holds per-invocation global options and streams.
type session struct {
	root   string
	format report.Format
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// Execute runs the CLI with the given arguments and returns the exit code.
func (c *CLI) Execute(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	s := &session{stdin: stdin, stdout: stdout, stderr: stderr}
	root := c.rootCommand(s)
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return domain.ExitPassed
	}
	var exit *exitError
	if errors.As(err, &exit) {
		if exit.message != "" {
			fmt.Fprintln(stderr, exit.message)
		}
		return exit.code
	}
	fmt.Fprintf(stderr, "%s: %v\n", product.Name, err)
	return domain.ExitIncomplete
}

func (c *CLI) rootCommand(s *session) *cobra.Command {
	var rootFlag, formatFlag string
	root := &cobra.Command{
		Use:     product.Name,
		Short:   product.DisplayName + ": " + product.Tagline,
		Version: c.app.Tool.Version,
		Long: product.DisplayName + " (CC stands for Crap Cleaner) sits in a coding agent's loop and checks every\n" +
			"change for complex functions, copied blocks and dependency tangles in JavaScript, TypeScript,\n" +
			"Python, Java and Go. Compared with an earlier version, it fails only on what the change made\n" +
			"new or worse, so the code stays small and clean. It runs locally and never calls an AI model.\n\n" +
			"Start here:\n" +
			"  " + product.Name + " init                  create " + product.ConfigFileName + "\n" +
			"  " + product.Name + " check                 measure the whole project\n" +
			"  " + product.Name + " check --base HEAD     show only what uncommitted work made worse\n" +
			"  " + product.Name + " guide                 step-by-step guides for people and coding agents\n\n" +
			"Exit codes:\n" +
			"  0  analysis completed and nothing blocking is new or worse\n" +
			"  1  a blocking finding is new or worse\n" +
			"  2  invalid input or configuration, incomplete analysis, or internal failure",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			format, err := report.ParseFormat(formatFlag)
			if err != nil {
				return err
			}
			if format == report.FormatSARIF && cmd.Name() != "check" {
				return fmt.Errorf("--format sarif is only supported by check")
			}
			s.format = format
			if rootFlag == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("determine working directory: %w", err)
				}
				rootFlag = cwd
			}
			absolute, err := filepath.Abs(rootFlag)
			if err != nil {
				return fmt.Errorf("resolve --root: %w", err)
			}
			s.root = absolute
			return nil
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.PersistentFlags().StringVar(&rootFlag, "root", "", "repository root to analyze (default: current directory)")
	root.PersistentFlags().StringVar(&formatFlag, "format", string(report.FormatHuman), "output format: human, json, or sarif (check only)")
	root.AddCommand(
		c.initCommand(s),
		c.doctorCommand(s),
		c.checkCommand(s),
		c.worklistCommand(s),
		c.baselineCommand(s),
		c.explainCommand(s),
		c.guideCommand(s),
		c.hookCommand(s),
		c.versionCommand(s),
	)
	return root
}
