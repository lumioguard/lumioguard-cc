// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package core

type Stack[T any] struct {
	data []T
}

func (s *Stack[T]) Push(item T) {
	s.data = append(s.data, item)
}

func (s *Stack[T]) Pop() T {
	l := len(s.data)
	if l == 0 {
		panic("stack is empty")
	}
	item := s.data[l-1]
	var zero T
	s.data[l-1] = zero
	s.data = s.data[:l-1]
	return item
}

func (s *Stack[T]) Peek() T {
	l := len(s.data)
	if l == 0 {
		panic("stack is empty")
	}
	return s.data[l-1]
}

func (s *Stack[T]) Len() int {
	return len(s.data)
}
