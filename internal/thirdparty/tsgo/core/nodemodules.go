// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import (
	"maps"
	"sync"
)

// require('module').builtinModules.filter(x => !x.match(/^(?:_|node:)/))
var UnprefixedNodeCoreModules = map[string]bool{
	"assert":              true,
	"assert/strict":       true,
	"async_hooks":         true,
	"buffer":              true,
	"child_process":       true,
	"cluster":             true,
	"console":             true,
	"constants":           true,
	"crypto":              true,
	"dgram":               true,
	"diagnostics_channel": true,
	"dns":                 true,
	"dns/promises":        true,
	"domain":              true,
	"events":              true,
	"fs":                  true,
	"fs/promises":         true,
	"http":                true,
	"http2":               true,
	"https":               true,
	"inspector":           true,
	"inspector/promises":  true,
	"module":              true,
	"net":                 true,
	"os":                  true,
	"path":                true,
	"path/posix":          true,
	"path/win32":          true,
	"perf_hooks":          true,
	"process":             true,
	"punycode":            true,
	"querystring":         true,
	"readline":            true,
	"readline/promises":   true,
	"repl":                true,
	"stream":              true,
	"stream/consumers":    true,
	"stream/promises":     true,
	"stream/web":          true,
	"string_decoder":      true,
	"sys":                 true,
	"timers":              true,
	"timers/promises":     true,
	"tls":                 true,
	"trace_events":        true,
	"tty":                 true,
	"url":                 true,
	"util":                true,
	"util/types":          true,
	"v8":                  true,
	"vm":                  true,
	"wasi":                true,
	"worker_threads":      true,
	"zlib":                true,
}

// require('module').builtinModules.filter(x => x.startsWith('node:'))
var ExclusivelyPrefixedNodeCoreModules = map[string]bool{
	"node:quic":           true,
	"node:sea":            true,
	"node:sqlite":         true,
	"node:test":           true,
	"node:test/reporters": true,
}

var NodeCoreModules = sync.OnceValue(func() map[string]bool {
	nodeCoreModules := make(map[string]bool, len(UnprefixedNodeCoreModules)*2+len(ExclusivelyPrefixedNodeCoreModules))
	for unprefixed := range UnprefixedNodeCoreModules {
		nodeCoreModules[unprefixed] = true
		nodeCoreModules["node:"+unprefixed] = true
	}
	maps.Copy(nodeCoreModules, ExclusivelyPrefixedNodeCoreModules)
	return nodeCoreModules
})

func NonRelativeModuleNameForTypingCache(moduleName string) string {
	if NodeCoreModules()[moduleName] {
		return "node"
	}
	return moduleName
}
