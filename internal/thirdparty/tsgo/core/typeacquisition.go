// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import "slices"

type TypeAcquisition struct {
	Enable                              Tristate `json:"enable,omitzero"`
	Include                             []string `json:"include,omitzero"`
	Exclude                             []string `json:"exclude,omitzero"`
	DisableFilenameBasedTypeAcquisition Tristate `json:"disableFilenameBasedTypeAcquisition,omitzero"`
}

func (ta *TypeAcquisition) Equals(other *TypeAcquisition) bool {
	if ta == other {
		return true
	}
	if ta == nil || other == nil {
		return false
	}

	return (ta.Enable == other.Enable &&
		slices.Equal(ta.Include, other.Include) &&
		slices.Equal(ta.Exclude, other.Exclude) &&
		ta.DisableFilenameBasedTypeAcquisition == other.DisableFilenameBasedTypeAcquisition)
}
