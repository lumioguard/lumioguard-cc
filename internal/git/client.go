// Package git wraps the read-only Git operations the tool needs. Nothing here
// modifies the index or the checkout.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

// Client runs read-only queries with the git executable. Tests substitute
// app.GitReferences one layer up instead.
type Client struct {
	executable string
}

// NewClient creates a Client that shells out to the git executable on PATH.
func NewClient() *Client {
	return &Client{executable: "git"}
}

// run executes git inside dir and returns its standard output.
func (c *Client) run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.executable, append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), detail)
	}
	return stdout.Bytes(), nil
}

func (c *Client) output(ctx context.Context, dir string, args ...string) (string, error) {
	raw, err := c.run(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

// Version returns the installed git version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	return c.output(ctx, ".", "--version")
}

// Inspect describes the Git state of root. An unresolvable base is a required
// error, because the comparison was explicitly requested.
func (c *Client) Inspect(ctx context.Context, root, base string) (domain.GitSnapshot, []domain.Diagnostic) {
	snapshot, err := c.inspect(ctx, root, base)
	if err == nil {
		return snapshot, nil
	}
	unavailable := domain.GitSnapshot{Available: false}
	if base != "" {
		return unavailable, []domain.Diagnostic{domain.RequiredError("git.base_unavailable", "",
			fmt.Sprintf("Git base '%s' could not be resolved: %v", base, err))}
	}
	return unavailable, []domain.Diagnostic{domain.Advisory("git.unavailable", "",
		"Directory is not a Git work tree; full analysis and named baselines remain available.", domain.SeverityInfo)}
}

func (c *Client) inspect(ctx context.Context, root, base string) (domain.GitSnapshot, error) {
	if _, err := c.output(ctx, root, "rev-parse", "--is-inside-work-tree"); err != nil {
		return domain.GitSnapshot{}, err
	}
	commit, err := c.output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return domain.GitSnapshot{}, err
	}
	status, err := c.output(ctx, root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return domain.GitSnapshot{}, err
	}
	dirty := status != ""
	snapshot := domain.GitSnapshot{Available: true, Commit: &commit, Dirty: &dirty}
	if base != "" {
		mergeBase, err := c.output(ctx, root, "merge-base", commit, base)
		if err != nil {
			return domain.GitSnapshot{}, err
		}
		snapshot.BaseCommit = &mergeBase
	}
	return snapshot, nil
}

// MergeBase returns the merge base of HEAD and ref, which isolates the branch's
// own changes even after ref has moved on.
func (c *Client) MergeBase(ctx context.Context, root, ref string) (string, error) {
	head, err := c.output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return c.output(ctx, root, "merge-base", head, ref)
}

// RevParse resolves a reference to a full commit hash.
func (c *Client) RevParse(ctx context.Context, root, ref string) (string, error) {
	return c.output(ctx, root, "rev-parse", "--verify", ref+"^{commit}")
}
