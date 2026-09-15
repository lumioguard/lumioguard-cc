// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

// BinarySearchUniqueFunc works like [slices.BinarySearchFunc], but avoids extra
// invocations of the comparison function by assuming that only one element
// in the slice could match the target. Also, unlike [slices.BinarySearchFunc],
// the comparison function is passed the current index of the element being
// compared, instead of the target element.
func BinarySearchUniqueFunc[S ~[]E, E any](x S, cmp func(int, E) int) (int, bool) {
	n := len(x)
	if n == 0 {
		return 0, false
	}
	low, high := 0, n-1
	for low <= high {
		middle := low + ((high - low) >> 1)
		value := cmp(middle, x[middle])
		if value < 0 {
			low = middle + 1
		} else if value > 0 {
			high = middle - 1
		} else {
			return middle, true
		}
	}
	return low, false
}
