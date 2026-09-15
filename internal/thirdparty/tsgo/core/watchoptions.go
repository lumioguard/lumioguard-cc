// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import "time"

type WatchOptions struct {
	Interval        *int               `json:"watchInterval"`
	FileKind        WatchFileKind      `json:"watchFile"`
	DirectoryKind   WatchDirectoryKind `json:"watchDirectory"`
	FallbackPolling PollingKind        `json:"fallbackPolling"`
	SyncWatchDir    Tristate           `json:"synchronousWatchDirectory"`
	ExcludeDir      []string           `json:"excludeDirectories"`
	ExcludeFiles    []string           `json:"excludeFiles"`
}

type WatchFileKind int32

const (
	WatchFileKindNone                         WatchFileKind = 0
	WatchFileKindFixedPollingInterval         WatchFileKind = 1
	WatchFileKindPriorityPollingInterval      WatchFileKind = 2
	WatchFileKindDynamicPriorityPolling       WatchFileKind = 3
	WatchFileKindFixedChunkSizePolling        WatchFileKind = 4
	WatchFileKindUseFsEvents                  WatchFileKind = 5
	WatchFileKindUseFsEventsOnParentDirectory WatchFileKind = 6
)

type WatchDirectoryKind int32

const (
	WatchDirectoryKindNone                   WatchDirectoryKind = 0
	WatchDirectoryKindUseFsEvents            WatchDirectoryKind = 1
	WatchDirectoryKindFixedPollingInterval   WatchDirectoryKind = 2
	WatchDirectoryKindDynamicPriorityPolling WatchDirectoryKind = 3
	WatchDirectoryKindFixedChunkSizePolling  WatchDirectoryKind = 4
)

type PollingKind int32

const (
	PollingKindNone             PollingKind = 0
	PollingKindFixedInterval    PollingKind = 1
	PollingKindPriorityInterval PollingKind = 2
	PollingKindDynamicPriority  PollingKind = 3
	PollingKindFixedChunkSize   PollingKind = 4
)

func (w *WatchOptions) WatchInterval() time.Duration {
	watchInterval := 2000 * time.Millisecond
	if w != nil && w.Interval != nil {
		watchInterval = time.Duration(*w.Interval) * time.Millisecond
	}
	return watchInterval
}
