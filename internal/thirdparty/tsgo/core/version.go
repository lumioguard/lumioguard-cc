// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import (
	"strings"
)

// This is a var so it can be overridden by ldflags.
var version = "7.1.0-dev"

func Version() string {
	return version
}

var versionMajorMinor = func() string {
	seenMajor := false
	i := strings.IndexFunc(version, func(r rune) bool {
		if r == '.' {
			if seenMajor {
				return true
			}
			seenMajor = true
		}
		return false
	})
	if i == -1 {
		panic("invalid version string: " + version)
	}
	return version[:i]
}()

func VersionMajorMinor() string {
	return versionMajorMinor
}
