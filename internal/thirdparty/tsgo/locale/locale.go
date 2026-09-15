// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package locale

import (
	"context"

	"golang.org/x/text/language"
)

type contextKey int

type Locale language.Tag

var Default Locale

func (l Locale) String() string {
	if l == Default {
		return ""
	}
	return language.Tag(l).String()
}

func WithLocale(ctx context.Context, locale Locale) context.Context {
	return context.WithValue(ctx, contextKey(0), locale)
}

func FromContext(ctx context.Context) Locale {
	locale, _ := ctx.Value(contextKey(0)).(Locale)
	return locale
}

func Parse(localeStr string) (locale Locale, ok bool) {
	// Parse gracefully fails.
	tag, err := language.Parse(localeStr)
	return Locale(tag), err == nil
}
