package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// ChangedLines maps lines added since the merge base of HEAD and base, plus all
// lines of untracked files. Failure is a required diagnostic.
func (c *Client) ChangedLines(ctx context.Context, root, base string) (analysis.ChangedLines, string, []domain.Diagnostic) {
	lines, baseCommit, err := c.changedLines(ctx, root, base)
	if err != nil {
		return analysis.ChangedLines{}, "", []domain.Diagnostic{domain.RequiredError("git.diff_failed", "",
			fmt.Sprintf("Could not calculate changed lines: %v", err))}
	}
	return lines, baseCommit, nil
}

func (c *Client) changedLines(ctx context.Context, root, base string) (analysis.ChangedLines, string, error) {
	baseCommit, err := c.MergeBase(ctx, root, base)
	if err != nil {
		return nil, "", err
	}
	diff, err := c.run(ctx, root, "-c", "core.quotepath=false", "diff", "--unified=0", "--no-color", "--no-ext-diff", baseCommit, "--")
	if err != nil {
		return nil, "", err
	}
	lines := parseAddedLines(string(diff))
	untracked, err := c.output(ctx, root, "-c", "core.quotepath=false", "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, "", err
	}
	for _, name := range strings.Split(untracked, "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		data, err := os.ReadFile(paths.Resolve(root, name))
		if err != nil {
			// Untracked entries that cannot be read as files are outside changed source-line coverage.
			continue
		}
		lines[filepath.ToSlash(name)] = allLines(string(data))
	}
	return lines, baseCommit, nil
}

// parseAddedLines extracts the "+" side line ranges of a unified diff with
// zero context lines. Deleted files reset the current target.
func parseAddedLines(diff string) analysis.ChangedLines {
	result := analysis.ChangedLines{}
	current := ""
	for raw := range strings.Lines(diff) {
		line := strings.TrimRight(raw, "\r\n")
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			current = line[len("+++ b/"):]
			if _, ok := result[current]; !ok {
				result[current] = analysis.LineSet{}
			}
		case strings.HasPrefix(line, "+++ "):
			current = ""
		case strings.HasPrefix(line, "@@"):
			// Guarding on the prefix keeps the regexp off the added and
			// removed content lines, which are almost the whole diff.
			match := hunkHeader.FindStringSubmatch(line)
			if match == nil || current == "" {
				continue
			}
			start, _ := strconv.Atoi(match[1])
			count := 1
			if match[2] != "" {
				count, _ = strconv.Atoi(match[2])
			}
			for added := start; added < start+count; added++ {
				result[current].Add(added)
			}
		}
	}
	return result
}

// allLines marks every line of an untracked file as changed, counting
// separators instead of splitting the text.
func allLines(text string) analysis.LineSet {
	count := strings.Count(text, "\n") + 1
	set := make(analysis.LineSet, count)
	for line := 1; line <= count; line++ {
		set.Add(line)
	}
	return set
}
