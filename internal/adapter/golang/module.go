package golang

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
)

// moduleFinder maps a source file to its Go import path by reading the
// nearest go.mod above it, no higher than the analyzed root. Each directory
// is read once per process; files are analyzed concurrently.
type moduleFinder struct {
	cache sync.Map // absolute directory -> string (module path, "" when the directory has no go.mod)
}

// locate returns the module of the file, or an empty module when no go.mod
// lies between the file and the root. Without one, every import is external.
func (f *moduleFinder) locate(root, absolutePath string) (adapter.Module, error) {
	directory := filepath.Dir(absolutePath)
	for {
		modulePath, err := f.modulePathIn(directory)
		if err != nil {
			return adapter.Module{}, err
		}
		if modulePath != "" {
			relative, err := filepath.Rel(directory, filepath.Dir(absolutePath))
			if err != nil {
				return adapter.Module{}, err
			}
			importPath := modulePath
			if relative != "." {
				importPath += "/" + filepath.ToSlash(relative)
			}
			return adapter.Module{Package: importPath, Root: modulePath}, nil
		}
		if sameDirectory(directory, root) {
			return adapter.Module{}, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return adapter.Module{}, nil
		}
		directory = parent
	}
}

func (f *moduleFinder) modulePathIn(directory string) (string, error) {
	if cached, ok := f.cache.Load(directory); ok {
		return cached.(string), nil
	}
	modulePath, err := readModulePath(filepath.Join(directory, "go.mod"))
	if err != nil {
		return "", err
	}
	f.cache.Store(directory, modulePath)
	return modulePath, nil
}

// readModulePath returns the module directive of a go.mod file, or "" when
// the file does not exist.
func readModulePath(path string) (string, error) {
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "module") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "module"))
		if rest == line || rest == "" {
			continue
		}
		if comment := strings.Index(rest, "//"); comment >= 0 {
			rest = strings.TrimSpace(rest[:comment])
		}
		return strings.Trim(rest, `"`), scanner.Err()
	}
	return "", scanner.Err()
}

func sameDirectory(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
