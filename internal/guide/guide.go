// Package guide holds the task guides printed by `lumioguard-cc guide`. They ship
// inside the binary, so the instructions always match the installed version.
package guide

import (
	"embed"
	"strings"
)

//go:embed topics/*.md
var files embed.FS

// Topic is one task guide. Text is empty in listings.
type Topic struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Text    string `json:"text,omitempty"`
}

// topics lists the guides in reading order; each has topics/<name>.md.
var topics = []Topic{
	{Name: "setup", Summary: "Set up a project: configuration, what blocks, agent hooks and CI"},
	{Name: "check", Summary: "Check a code change before finishing and fix what it made worse"},
	{Name: "cleanup", Summary: "Reduce existing debt in small refactors that keep behaviour"},
	{Name: "report", Summary: "Read the JSON report and find what a change caused"},
	{Name: "config", Summary: "Configuration fields, exclusions, boundaries and coverage"},
}

// Topics lists every guide in reading order, without the text.
func Topics() []Topic {
	return append([]Topic(nil), topics...)
}

// Lookup returns the named guide with {{version}} replaced by version.
func Lookup(name, version string) (Topic, bool) {
	for _, topic := range topics {
		if topic.Name != name {
			continue
		}
		data, err := files.ReadFile("topics/" + name + ".md")
		if err != nil {
			return Topic{}, false
		}
		topic.Text = strings.ReplaceAll(string(data), "{{version}}", version)
		return topic, true
	}
	return Topic{}, false
}
