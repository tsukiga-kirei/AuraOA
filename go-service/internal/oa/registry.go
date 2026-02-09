package oa

import (
	"fmt"
	"sync"
)

// DefaultRegistry is the global adapter registry.
var DefaultRegistry = NewRegistry()

// Registry manages OA adapters by type and version.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]OAAdapter // key: "type:version"
}

func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]OAAdapter),
	}
}

func (r *Registry) GetAdapter(oaType string, version string) (OAAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := oaType + ":" + version
	adapter, ok := r.adapters[key]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for %s", key)
	}
	return adapter, nil
}

func (r *Registry) RegisterAdapter(oaType string, version string, adapter OAAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := oaType + ":" + version
	r.adapters[key] = adapter
	return nil
}
