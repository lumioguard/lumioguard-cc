// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

import "context"

type Semaphore interface {
	Acquire() (release func())
	TryAcquire(ctx context.Context) (release func(), acquired bool)
}

var _ Semaphore = UnlimitedSemaphore{}

type UnlimitedSemaphore struct{}

func (s UnlimitedSemaphore) Acquire() (release func()) {
	return func() {}
}

func (s UnlimitedSemaphore) TryAcquire(ctx context.Context) (release func(), acquired bool) {
	return func() {}, true
}

var _ Semaphore = (*LimitedSemaphore)(nil)

type LimitedSemaphore struct {
	ch      chan struct{}
	release func()
}

func NewLimitedSemaphore(maxConcurrency int) *LimitedSemaphore {
	if maxConcurrency <= 0 {
		panic("maxConcurrency must be positive")
	}
	s := &LimitedSemaphore{
		ch: make(chan struct{}, maxConcurrency),
	}
	s.release = func() { <-s.ch }
	return s
}

func (s *LimitedSemaphore) Acquire() (release func()) {
	s.ch <- struct{}{}
	return s.release
}

func (s *LimitedSemaphore) TryAcquire(ctx context.Context) (release func(), acquired bool) {
	select {
	case s.ch <- struct{}{}:
		return s.release, true
	case <-ctx.Done():
		return func() {}, false
	}
}
