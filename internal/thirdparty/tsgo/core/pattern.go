// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import "strings"

type Pattern struct {
	Text      string
	StarIndex int // -1 for exact match
}

func TryParsePattern(pattern string) Pattern {
	starIndex := strings.Index(pattern, "*")
	if starIndex == -1 || !strings.Contains(pattern[starIndex+1:], "*") {
		return Pattern{Text: pattern, StarIndex: starIndex}
	}
	return Pattern{}
}

func (p *Pattern) IsValid() bool {
	return p.StarIndex == -1 || p.StarIndex < len(p.Text)
}

func (p *Pattern) Matches(candidate string) bool {
	if p.StarIndex == -1 {
		return p.Text == candidate
	}
	return len(candidate) >= len(p.Text)-1 &&
		strings.HasPrefix(candidate, p.Text[:p.StarIndex]) &&
		strings.HasSuffix(candidate, p.Text[p.StarIndex+1:])
}

func (p *Pattern) MatchedText(candidate string) string {
	if !p.Matches(candidate) {
		panic("candidate does not match pattern")
	}
	if p.StarIndex == -1 {
		return ""
	}
	return candidate[p.StarIndex : len(candidate)-len(p.Text)+p.StarIndex+1]
}

func FindBestPatternMatch[T any](values []T, getPattern func(v T) Pattern, candidate string) T {
	var bestPattern T
	longestMatchPrefixLength := -1
	for _, value := range values {
		pattern := getPattern(value)
		if (pattern.StarIndex == -1 || pattern.StarIndex > longestMatchPrefixLength) && pattern.Matches(candidate) {
			bestPattern = value
			longestMatchPrefixLength = pattern.StarIndex
		}
	}
	return bestPattern
}
