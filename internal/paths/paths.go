// Package paths normalises file system paths into the repository-relative,
// forward-slash form used everywhere in reports and baselines.
package paths

import (
	"path"
	"path/filepath"
	"strings"
)

// Relative returns target relative to root using forward slashes. When target
// cannot be expressed relative to root, its slash-normalised form is returned.
func Relative(root, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(relative)
}

// Resolve joins a possibly relative path with root, keeping absolute paths as they are.
func Resolve(root, target string) string {
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Join(root, filepath.FromSlash(target))
}

// Extension returns the lower-cased file extension including the dot.
func Extension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// Dir returns the directory of a repository-relative path, with the repository
// root spelled as the empty string rather than path.Dir's ".".
func Dir(relative string) string {
	directory := path.Dir(relative)
	if directory == "." {
		return ""
	}
	return directory
}
