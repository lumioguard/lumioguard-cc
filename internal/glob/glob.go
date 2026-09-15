// Package glob matches repository-relative paths against the doublestar glob
// patterns used by source selection and architecture boundaries.
package glob

import "github.com/bmatcuk/doublestar/v4"

// MatchesAny reports whether the repository-relative path matches at least one
// pattern. "**" spans any number of directories, including none.
func MatchesAny(relative string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, err := doublestar.Match(pattern, relative); err == nil && matched {
			return true
		}
	}
	return false
}

// MatchesDirectory reports whether a directory matches a pattern, trying "dir"
// and "dir/" so "**/dist/**" prunes the directory itself.
func MatchesDirectory(relative string, patterns []string) bool {
	return MatchesAny(relative, patterns) || MatchesAny(relative+"/", patterns)
}

// Valid reports whether the pattern is syntactically valid.
func Valid(pattern string) bool {
	return doublestar.ValidatePattern(pattern)
}
