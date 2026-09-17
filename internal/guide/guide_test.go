package guide

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/config"
)

func TestEveryTopicHasExactlyOneFile(t *testing.T) {
	files, err := fs.Glob(files, "topics/*.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(Topics()) {
		t.Fatalf("%d topic files for %d topics: %v", len(files), len(Topics()), files)
	}
	for _, listed := range Topics() {
		topic, ok := Lookup(listed.Name, "9.9.9")
		if !ok || !strings.HasPrefix(topic.Text, "# ") {
			t.Fatalf("topic %q is missing or has no title", listed.Name)
		}
		if strings.Contains(topic.Text, "{{") {
			t.Fatalf("topic %q has an unfilled placeholder", listed.Name)
		}
	}
	if _, ok := Lookup("nope", "1"); ok {
		t.Fatal("unknown topic must not be found")
	}
}

// The configuration guide shows the defaults; they must match what init writes.
func TestConfigGuideShowsTheRealDefaults(t *testing.T) {
	topic, _ := Lookup("config", "1")
	block := regexp.MustCompile("(?s)```json\n(.*?)```").FindStringSubmatch(topic.Text)
	if block == nil {
		t.Fatal("config guide has no JSON block")
	}
	var shown, actual any
	if err := json.Unmarshal([]byte(block[1]), &shown); err != nil {
		t.Fatalf("config guide JSON is invalid: %v", err)
	}
	data, err := json.Marshal(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(shown, actual) {
		t.Fatalf("config guide defaults differ from config.Default():\nshown:  %v\nactual: %v", shown, actual)
	}
}
