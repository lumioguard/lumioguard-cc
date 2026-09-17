// Command third-party-audit checks copied upstreams against vuln.go.dev, which govulncheck cannot
// do because copied code carries this module's path. Exit 0 clean, 1 advisories, 2 not run.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lumioguard/lumioguard-cc/internal/thirdparty"
)

const databaseURL = "https://vuln.go.dev"

// moduleIndex is one entry of /index/modules.json.
type moduleIndex struct {
	Path  string `json:"path"`
	Vulns []struct {
		ID       string `json:"id"`
		Modified string `json:"modified"`
		Fixed    string `json:"fixed"`
	} `json:"vulns"`
}

// osvEntry is the subset of the OSV record this command reports.
type osvEntry struct {
	ID       string   `json:"id"`
	Summary  string   `json:"summary"`
	Aliases  []string `json:"aliases"`
	Affected []struct {
		Package struct {
			Name string `json:"name"`
		} `json:"package"`
		Ranges []struct {
			Events []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
}

func main() {
	timeout := flag.Duration("timeout", 60*time.Second, "network timeout")
	flag.Parse()
	code, err := run(*timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "third-party-audit:", err)
		os.Exit(2)
	}
	os.Exit(code)
}

func run(timeout time.Duration) (int, error) {
	components := thirdparty.All()
	fmt.Printf("Auditing %d copied or generated upstream components against %s\n\n", len(components), databaseURL)

	var withModule []thirdparty.Component
	for _, component := range components {
		if component.Module == "" {
			fmt.Printf("%-28s %s\n", component.Path, "not a Go module; pinned by commit "+short(component.Version))
			fmt.Printf("%-28s %s\n\n", "", "no vulnerability database covers it; review the upstream when refreshing the pin")
			continue
		}
		withModule = append(withModule, component)
	}
	if len(withModule) == 0 {
		return 0, nil
	}

	client := &http.Client{Timeout: timeout}
	index, err := fetchIndex(client)
	if err != nil {
		return 0, err
	}

	findings := 0
	for _, component := range withModule {
		entry, listed := index[component.Module]
		fmt.Printf("%-28s %s@%s\n", component.Path, component.Module, short(component.Version))
		if !listed {
			fmt.Printf("%-28s no advisories recorded for this module\n\n", "")
			continue
		}
		ids := make([]string, 0, len(entry.Vulns))
		for _, vuln := range entry.Vulns {
			ids = append(ids, vuln.ID)
		}
		sort.Strings(ids)
		for _, id := range ids {
			advisory, err := fetchAdvisory(client, id)
			if err != nil {
				return 0, err
			}
			findings++
			fmt.Printf("%-28s %s: %s\n", "", advisory.ID, summary(advisory))
			fmt.Printf("%-28s   affected: %s\n", "", ranges(advisory, component.Module))
			fmt.Printf("%-28s   details:  %s/ID/%s.json\n", "", databaseURL, advisory.ID)
		}
		fmt.Printf("%-28s pinned version is %s; compare it with the ranges above\n\n", "", component.Version)
	}

	if findings == 0 {
		fmt.Println("No advisories found for any copied upstream.")
		return 0, nil
	}
	fmt.Printf("%d advisory record(s) affect a copied upstream. Review each against the pinned version.\n", findings)
	fmt.Println("A pseudo-version pins an untagged commit, so decide by commit date rather than by semantic version order.")
	return 1, nil
}

func fetchIndex(client *http.Client) (map[string]moduleIndex, error) {
	var entries []moduleIndex
	if err := fetchJSON(client, databaseURL+"/index/modules.json", &entries); err != nil {
		return nil, err
	}
	index := make(map[string]moduleIndex, len(entries))
	for _, entry := range entries {
		index[entry.Path] = entry
	}
	return index, nil
}

func fetchAdvisory(client *http.Client, id string) (osvEntry, error) {
	var advisory osvEntry
	err := fetchJSON(client, fmt.Sprintf("%s/ID/%s.json", databaseURL, id), &advisory)
	return advisory, err
}

func fetchJSON(client *http.Client, url string, out any) error {
	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: HTTP %d", url, response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}
	return nil
}

func summary(advisory osvEntry) string {
	text := advisory.Summary
	if text == "" {
		text = strings.Join(advisory.Aliases, ", ")
	}
	if text == "" {
		text = "no summary published"
	}
	return text
}

func ranges(advisory osvEntry, module string) string {
	var parts []string
	for _, affected := range advisory.Affected {
		if affected.Package.Name != module && !strings.HasPrefix(affected.Package.Name, module+"/") {
			continue
		}
		for _, r := range affected.Ranges {
			for _, event := range r.Events {
				switch {
				case event.Introduced != "":
					parts = append(parts, "introduced "+event.Introduced)
				case event.Fixed != "":
					parts = append(parts, "fixed "+event.Fixed)
				}
			}
		}
	}
	if len(parts) == 0 {
		return "see the advisory"
	}
	return strings.Join(parts, ", ")
}

func short(version string) string {
	if len(version) > 24 {
		return version[:24] + "..."
	}
	return version
}
