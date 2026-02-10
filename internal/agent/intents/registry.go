package intents

import (
	"fmt"
	"sort"
	"sync"
)

// Registry stores pluggable intent modules by name.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]IntentModule
}

func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]IntentModule)}
}

func (r *Registry) Register(module IntentModule) error {
	if module == nil {
		return fmt.Errorf("module is nil")
	}

	name := module.IntentName()
	if name == "" {
		return fmt.Errorf("module intent name is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("intent module already registered: %s", name)
	}
	r.modules[name] = module
	return nil
}

func (r *Registry) Get(intent string) (IntentModule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	module, ok := r.modules[intent]
	return module, ok
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

