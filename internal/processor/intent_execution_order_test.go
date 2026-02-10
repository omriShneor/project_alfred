package processor

import (
	"context"
	"testing"

	"github.com/omriShneor/project_alfred/internal/agent/intents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubIntentModule struct {
	name string
}

func (m *stubIntentModule) IntentName() string { return m.name }

func (m *stubIntentModule) AnalyzeMessages(ctx context.Context, in intents.MessageInput) (*intents.ModuleOutput, error) {
	return nil, nil
}

func (m *stubIntentModule) AnalyzeEmail(ctx context.Context, in intents.EmailInput) (*intents.ModuleOutput, error) {
	return nil, nil
}

func (m *stubIntentModule) Validate(ctx context.Context, out *intents.ModuleOutput) error {
	return nil
}

func (m *stubIntentModule) Persist(ctx context.Context, out *intents.ModuleOutput, persister intents.Persister) error {
	return nil
}

func TestResolveIntentExecutionOrder(t *testing.T) {
	registry := intents.NewRegistry()
	require.NoError(t, registry.Register(&stubIntentModule{name: "event"}))
	require.NoError(t, registry.Register(&stubIntentModule{name: "reminder"}))

	t.Run("none route runs all registered", func(t *testing.T) {
		order, unknown := resolveIntentExecutionOrder(registry, intents.RoutedIntent{Intent: "none"})
		assert.False(t, unknown)
		assert.Equal(t, []string{"event", "reminder"}, order)
	})

	t.Run("known route is prioritized but still runs others", func(t *testing.T) {
		order, unknown := resolveIntentExecutionOrder(registry, intents.RoutedIntent{Intent: "reminder"})
		assert.False(t, unknown)
		assert.Equal(t, []string{"reminder", "event"}, order)
	})

	t.Run("unknown route hard stops", func(t *testing.T) {
		order, unknown := resolveIntentExecutionOrder(registry, intents.RoutedIntent{Intent: "task"})
		assert.True(t, unknown)
		assert.Equal(t, []string{"task"}, order)
	})
}

func TestResolveIntentExecutionOrder_EmptyRegistry(t *testing.T) {
	registry := intents.NewRegistry()
	order, unknown := resolveIntentExecutionOrder(registry, intents.RoutedIntent{Intent: "event"})
	assert.False(t, unknown)
	assert.Nil(t, order)
}

// Keep compile-time guard near tests where the stub is defined.
var _ intents.IntentModule = (*stubIntentModule)(nil)
