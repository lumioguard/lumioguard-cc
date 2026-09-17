// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import "github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/tspath"

type ProjectReference struct {
	// Path is a normalized path on disk.
	Path string `json:"path"`
	// OriginalPath is the path as it was originally written.
	OriginalPath string `json:"originalPath"`
	// Circular indicates that this reference is intended to form a circularity.
	Circular bool `json:"circular"`
}

func ResolveProjectReferencePath(ref *ProjectReference) string {
	return ResolveConfigFileNameOfProjectReference(ref.Path)
}

func ResolveConfigFileNameOfProjectReference(path string) string {
	if tspath.FileExtensionIs(path, tspath.ExtensionJson) {
		return path
	}
	return tspath.CombinePaths(path, "tsconfig.json")
}
