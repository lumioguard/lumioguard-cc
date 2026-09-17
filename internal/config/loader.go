package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/product"
)

// ErrExists is returned when a configuration file would be overwritten.
var ErrExists = errors.New("configuration file already exists")

// Loaded is the result of reading a repository's configuration.
type Loaded struct {
	Config domain.Config
	Path   string
	Exists bool
}

// Loader reads and writes the configuration file of a repository.
type Loader struct{}

// NewLoader creates a Loader.
func NewLoader() *Loader {
	return &Loader{}
}

// Path returns the configuration path for a repository root.
func (l *Loader) Path(root string) string {
	return filepath.Join(root, product.ConfigFileName)
}

// Load reads the configuration or returns the defaults when no file exists.
// An invalid file is an error: it is never silently replaced by defaults.
func (l *Loader) Load(root string) (Loaded, error) {
	path := l.Path(root)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Loaded{Config: Default(), Path: path, Exists: false}, nil
	}
	if err != nil {
		return Loaded{}, fmt.Errorf("read %s: %w", product.ConfigFileName, err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return Loaded{}, fmt.Errorf("invalid %s: %w", product.ConfigFileName, err)
	}
	return Loaded{Config: cfg, Path: path, Exists: true}, nil
}

// WriteDefault creates the default configuration file without overwriting an
// existing one. It returns ErrExists when the file is already present.
func (l *Loader) WriteDefault(root string) (string, error) {
	path := l.Path(root)
	data, err := json.MarshalIndent(Default(), "", "  ")
	if err != nil {
		return path, fmt.Errorf("encode default configuration: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return path, ErrExists
	}
	if err != nil {
		return path, fmt.Errorf("create %s: %w", product.ConfigFileName, err)
	}
	if err := writeAndClose(file, append(data, '\n')); err != nil {
		return path, fmt.Errorf("write %s: %w", product.ConfigFileName, err)
	}
	return path, nil
}

// writeAndClose reports a failed close too, because that is where a write can be lost.
func writeAndClose(file *os.File, data []byte) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
