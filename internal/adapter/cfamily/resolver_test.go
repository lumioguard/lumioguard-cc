package cfamily

import (
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
)

func TestResolveIncludes(t *testing.T) {
	source := &adapter.SourceFile{RelativePath: "src/net/socket.c"}
	var files []*adapter.SourceFile
	for _, path := range []string{
		"src/net/socket.c", "src/net/socket.h", "src/util/log.h", "include/project/api.h",
		"lib/a/config.h", "lib/b/config.h", "string.h", "src/net/readme.py",
	} {
		files = append(files, &adapter.SourceFile{RelativePath: path})
	}
	index := adapter.NewModuleIndex(files)
	resolver := includeResolver{}
	cases := map[string]adapter.Resolution{
		`"socket.h"`:        adapter.Internal("src/net/socket.h"),
		`"../util/log.h"`:   adapter.Internal("src/util/log.h"),
		`"util/log.h"`:      adapter.Internal("src/util/log.h"),
		`"src/util/log.h"`:  adapter.Internal("src/util/log.h"),
		`<project/api.h>`:   adapter.Internal("include/project/api.h"),
		`"log.h"`:           adapter.Internal("src/util/log.h"),
		`"config.h"`:        adapter.NotResolved(),
		`<string.h>`:        adapter.External(),
		`<openssl/ssl.h>`:   adapter.External(),
		`"generated.h"`:     adapter.External(),
		`"readme.py"`:       adapter.External(),
		`"socket.c"`:        adapter.External(),
		`"..\\util\\log.h"`: adapter.Internal("src/util/log.h"),
	}
	for specifier, want := range cases {
		got := resolver.Resolve(source, adapter.Import{Specifier: specifier, Kind: adapter.ImportStatic}, index)
		if got != want {
			t.Errorf("Resolve(%s) = %+v, want %+v", specifier, got, want)
		}
	}
}
