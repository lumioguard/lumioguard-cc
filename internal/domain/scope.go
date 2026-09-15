package domain

import (
	"fmt"
	"strings"
)

// Scope locates a measurement or finding. Key is the stable identity used for
// baseline matching; it never depends on line numbers alone.
type Scope struct {
	Kind    ScopeKind `json:"kind"`
	File    string    `json:"file,omitempty"`
	Symbol  string    `json:"symbol,omitempty"`
	Line    int       `json:"line,omitempty"`
	EndLine int       `json:"endLine,omitempty"`
	Key     string    `json:"key"`
}

// RepositoryScope is the whole analyzed repository.
func RepositoryScope() Scope {
	return Scope{Kind: ScopeRepository, Key: "repository"}
}

// FileScope points at one repository-relative file.
func FileScope(file string) Scope {
	return Scope{Kind: ScopeFile, File: file, Key: file}
}

// FunctionScope points at a named function inside a file.
func FunctionScope(file, symbol string, line, endLine int) Scope {
	return Scope{
		Kind:    ScopeFunction,
		File:    file,
		Symbol:  symbol,
		Line:    line,
		EndLine: endLine,
		Key:     fmt.Sprintf("%s::%s", file, symbol),
	}
}

// CycleScope identifies a dependency cycle by its sorted member list.
func CycleScope(key string) Scope {
	return Scope{Kind: ScopeCycle, Key: key}
}

// DependencyScope identifies one source-to-target dependency edge.
func DependencyScope(file string, line int, key string) Scope {
	return Scope{Kind: ScopeDependency, File: file, Line: line, Key: key}
}

// Location renders "file:line", "file" or the key for display.
func (s Scope) Location() string {
	if s.File == "" {
		return s.Key
	}
	if s.Line > 0 {
		return fmt.Sprintf("%s:%d", s.File, s.Line)
	}
	return s.File
}

// WithFile returns a copy of the scope moved to another file, rewriting the
// key so baseline matching follows a detected rename.
func (s Scope) WithFile(file string) Scope {
	if s.File == "" || s.File == file {
		return s
	}
	moved := s
	if s.Key == s.File {
		moved.Key = file
	} else {
		moved.Key = strings.Replace(s.Key, s.File+"::", file+"::", 1)
	}
	moved.File = file
	return moved
}
