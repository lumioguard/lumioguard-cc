// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

type BuildOptions struct {
	_ noCopy

	Dry               Tristate `json:"dry,omitzero"`
	Force             Tristate `json:"force,omitzero"`
	Verbose           Tristate `json:"verbose,omitzero"`
	Builders          *int     `json:"builders,omitzero"`
	StopBuildOnErrors Tristate `json:"stopBuildOnErrors,omitzero"`

	// CompilerOptions are not parsed here and will be available on ParsedBuildCommandLine

	// Internal fields
	Clean Tristate `json:"clean,omitzero"`
}
