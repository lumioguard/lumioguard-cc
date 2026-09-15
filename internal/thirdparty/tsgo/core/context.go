// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import (
	"context"
)

type key int

const (
	requestIDKey key = iota
	checkerLifetimeKey
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

type CheckerLifetime int

const (
	CheckerLifetimeTemporary CheckerLifetime = iota
	CheckerLifetimeDiagnostics
	CheckerLifetimeAPI
)

func WithCheckerLifetime(ctx context.Context, lifetime CheckerLifetime) context.Context {
	return context.WithValue(ctx, checkerLifetimeKey, lifetime)
}

func GetCheckerLifetime(ctx context.Context) CheckerLifetime {
	if lifetime, ok := ctx.Value(checkerLifetimeKey).(CheckerLifetime); ok {
		return lifetime
	}
	return CheckerLifetimeTemporary
}
