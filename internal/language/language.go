// Package language is the single owner of the extension-to-language table.
// Adapters, default include patterns and scope summaries all derive from it.
package language

import (
	"path/filepath"
	"slices"
	"strings"
)

// Unknown is the name reported for a file no language claims.
const Unknown = "Unknown"

// Language is one language with every file extension that denotes it.
type Language struct {
	Name       string
	Extensions []string
}

// The languages this tool understands. An adapter may serve more than one, as
// the JavaScript/TypeScript adapter serves both.
var (
	JavaScript = Language{Name: "JavaScript", Extensions: []string{".js", ".jsx", ".mjs", ".cjs"}}
	TypeScript = Language{Name: "TypeScript", Extensions: []string{".ts", ".tsx", ".mts", ".cts"}}
	Python     = Language{Name: "Python", Extensions: []string{".py", ".pyw"}}
	Java       = Language{Name: "Java", Extensions: []string{".java"}}
	Go         = Language{Name: "Go", Extensions: []string{".go"}}
)

// All returns every language in a stable order.
func All() []Language {
	return []Language{JavaScript, TypeScript, Python, Java, Go}
}

// ExtensionSet builds the lookup an adapter uses for Supports.
func ExtensionSet(languages ...Language) map[string]bool {
	set := make(map[string]bool)
	for _, item := range languages {
		for _, extension := range item.Extensions {
			set[extension] = true
		}
	}
	return set
}

// Names lists the language names of the given languages.
func Names(languages ...Language) []string {
	names := make([]string, 0, len(languages))
	for _, item := range languages {
		names = append(names, item.Name)
	}
	return names
}

// NameFor classifies a file by extension. Classification does not imply that
// an adapter is installed for the language.
func NameFor(filename string) string {
	extension := strings.ToLower(filepath.Ext(filename))
	for _, item := range All() {
		if slices.Contains(item.Extensions, extension) {
			return item.Name
		}
	}
	return Unknown
}

// IncludePatterns returns the default source globs covering every extension,
// so a file an adapter can analyze is never invisible to discovery.
func IncludePatterns() []string {
	var patterns []string
	for _, item := range All() {
		for _, extension := range item.Extensions {
			patterns = append(patterns, "**/*"+extension)
		}
	}
	return patterns
}
