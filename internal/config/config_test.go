package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

func TestDefaultConfigurationValidates(t *testing.T) {
	parsed, err := Parse([]byte(mustJSON(Default())))
	if err != nil {
		t.Fatalf("default configuration must validate: %v", err)
	}
	if parsed.Metrics.Cyclomatic.Threshold != 12 || parsed.Metrics.DuplicationPercent.MinTokens != 50 {
		t.Fatalf("round trip changed values: %+v", parsed.Metrics)
	}
}

func TestParseRejectsUnknownBoundaryDependencies(t *testing.T) {
	cfg := Default()
	cfg.Architecture.Boundaries = []domain.Boundary{{Name: "domain", Include: []string{"src/domain/**"}, MayDependOn: []string{"missing"}}}
	_, err := Parse([]byte(mustJSON(cfg)))
	if err == nil || !strings.Contains(err.Error(), "unknown boundary") {
		t.Fatalf("expected unknown boundary error, got %v", err)
	}
}

func TestParseRejectsUnknownFieldsAndBadValues(t *testing.T) {
	cases := map[string]string{
		"unknown top-level field": `{"schemaVersion":"1.0","source":{"include":[],"exclude":[]},"metrics":{},"architecture":{},"coverage":{},"extra":1}`,
		"wrong schema version":    `{"schemaVersion":"2.0"}`,
		"not an object":           `[]`,
		"missing lcovFile":        strings.Replace(mustJSON(Default()), `"lcovFile":null,`, ``, 1),
		"negative threshold":      strings.Replace(mustJSON(Default()), `"threshold":12`, `"threshold":-1`, 1),
		"invalid severity":        strings.Replace(mustJSON(Default()), `"severity":"warning"`, `"severity":"fatal"`, 1),
		"fractional minTokens":    strings.Replace(mustJSON(Default()), `"minTokens":50`, `"minTokens":1.5`, 1),
		"invalid glob":            strings.Replace(mustJSON(Default()), `"**/*.js"`, `"src/["`, 1),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(raw)); err == nil {
				t.Fatalf("expected %s to be rejected", name)
			}
		})
	}
}

func TestLoaderReturnsDefaultsWhenMissingAndRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	loader := NewLoader()
	loaded, err := loader.Load(root)
	if err != nil || loaded.Exists {
		t.Fatalf("expected defaults, got exists=%v err=%v", loaded.Exists, err)
	}
	path, err := loader.WriteDefault(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if _, err := loader.WriteDefault(root); !errors.Is(err, ErrExists) {
		t.Fatalf("expected ErrExists, got %v", err)
	}
	loaded, err = loader.Load(root)
	if err != nil || !loaded.Exists {
		t.Fatalf("expected written config to load, got exists=%v err=%v", loaded.Exists, err)
	}
}

func TestLoaderReportsInvalidFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".lumioguard-cc.json"), []byte(`{"schemaVersion":"1.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewLoader().Load(root)
	if err == nil || !strings.Contains(err.Error(), "invalid .lumioguard-cc.json") {
		t.Fatalf("expected invalid configuration error, got %v", err)
	}
}

func mustJSON(cfg domain.Config) string {
	data, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return string(data)
}
