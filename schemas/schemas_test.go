// Package schemas verifies that produced documents conform to the versioned
// JSON schemas shipped with the tool.
package schemas

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/lumioguard/lumioguard-cc/internal/app"
	"github.com/lumioguard/lumioguard-cc/internal/baseline"
	"github.com/lumioguard/lumioguard-cc/internal/compose"
	"github.com/lumioguard/lumioguard-cc/internal/config"
)

const base = "https://raw.githubusercontent.com/lumioguard/lumioguard-cc/main/schemas/"

func compiler(t *testing.T) *jsonschema.Compiler {
	t.Helper()
	c := jsonschema.NewCompiler()
	for _, name := range []string{"config.schema.json", "report.schema.json", "baseline.schema.json"} {
		file, err := os.Open(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		document, err := jsonschema.UnmarshalJSON(file)
		_ = file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.AddResource(base+name, document); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// roundTrip converts a Go value into the generic form the validator expects.
func roundTrip(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatal(err)
	}
	return generic
}

func TestDefaultConfigurationMatchesSchema(t *testing.T) {
	schema, err := compiler(t).Compile(base + "config.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(roundTrip(t, config.Default())); err != nil {
		t.Fatalf("default configuration violates schema: %v", err)
	}
}

func TestProducedReportAndBaselineMatchSchemas(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.ts"), []byte("import { x } from './x';\nexport const answer = (a: number) => a > 1 ? x : 42;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "x.ts"), []byte("export const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	application := compose.NewApplication()
	report, err := application.Check.Run(context.Background(), app.CheckRequest{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	c := compiler(t)
	reportSchema, err := c.Compile(base + "report.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := reportSchema.Validate(roundTrip(t, report)); err != nil {
		t.Fatalf("report violates schema: %v", err)
	}
	baselineSchema, err := c.Compile(base + "baseline.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	entry := baseline.NewStore("0.1.0").FromReport("initial", report)
	if err := baselineSchema.Validate(roundTrip(t, entry)); err != nil {
		t.Fatalf("baseline violates schema: %v", err)
	}
}
