package processor

import (
	"strings"

	"github.com/omriShneor/project_alfred/internal/agent/intents"
)

// resolveIntentExecutionOrder builds the module run order for a routed intent.
// Routing is advisory for known intents: we prefer the routed module first, then
// run any other registered modules to maximize recall.
func resolveIntentExecutionOrder(registry *intents.Registry, route intents.RoutedIntent) (order []string, unknownRoutedIntent bool) {
	if registry == nil {
		return nil, false
	}

	registered := registry.List()
	if len(registered) == 0 {
		return nil, false
	}

	intent := strings.TrimSpace(route.Intent)
	switch intent {
	case "", "none", "both":
		return registered, false
	}

	if _, ok := registry.Get(intent); !ok {
		// Unknown/unregistered routed intent must hard-stop to safe no_action.
		return []string{intent}, true
	}

	// Put routed intent first, then run remaining registered modules.
	order = make([]string, 0, len(registered))
	order = append(order, intent)
	for _, name := range registered {
		if name == intent {
			continue
		}
		order = append(order, name)
	}
	return order, false
}
