package adapter

// Registry holds the installed language adapters in priority order.
type Registry struct {
	adapters []LanguageAdapter
}

// NewRegistry creates a registry. The first adapter that supports a file wins.
func NewRegistry(adapters ...LanguageAdapter) *Registry {
	return &Registry{adapters: append([]LanguageAdapter(nil), adapters...)}
}

// Find returns the adapter responsible for a file, if any.
func (r *Registry) Find(filename string) (LanguageAdapter, bool) {
	for _, candidate := range r.adapters {
		if candidate.Supports(filename) {
			return candidate, true
		}
	}
	return nil, false
}

// All returns the registered adapters in priority order.
func (r *Registry) All() []LanguageAdapter {
	return append([]LanguageAdapter(nil), r.adapters...)
}
