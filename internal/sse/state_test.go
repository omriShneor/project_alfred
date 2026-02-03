package sse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		state := NewState()

		assert.Equal(t, "checking", state.WhatsAppStatus)
		assert.Equal(t, "pending", state.TelegramStatus)
		assert.Equal(t, "checking", state.GCalStatus)
		assert.False(t, state.Complete)
	})

	t.Run("set whatsapp status", func(t *testing.T) {
		state := NewState()

		state.SetWhatsAppStatus("connected")
		assert.Equal(t, "connected", state.WhatsAppStatus)
	})

	t.Run("set telegram status", func(t *testing.T) {
		state := NewState()

		state.SetTelegramStatus("connected")
		assert.Equal(t, "connected", state.TelegramStatus)
	})

	t.Run("set gcal status", func(t *testing.T) {
		state := NewState()

		state.SetGCalStatus("connected")
		assert.Equal(t, "connected", state.GCalStatus)
	})

	t.Run("set qr code", func(t *testing.T) {
		state := NewState()

		state.SetQR("data:image/png;base64,...")
		assert.Equal(t, "data:image/png;base64,...", state.CurrentQR)
		assert.Equal(t, "waiting", state.WhatsAppStatus)
	})

	t.Run("set errors", func(t *testing.T) {
		state := NewState()

		state.SetWhatsAppError("connection failed")
		assert.Equal(t, "error", state.WhatsAppStatus)
		assert.Equal(t, "connection failed", state.WhatsAppError)

		state.SetTelegramError("auth failed")
		assert.Equal(t, "error", state.TelegramStatus)
		assert.Equal(t, "auth failed", state.TelegramError)

		state.SetGCalError("token expired")
		assert.Equal(t, "error", state.GCalStatus)
		assert.Equal(t, "token expired", state.GCalError)
	})

	t.Run("error clears when status changes", func(t *testing.T) {
		state := NewState()

		state.SetWhatsAppError("error message")
		assert.Equal(t, "error message", state.WhatsAppError)

		state.SetWhatsAppStatus("connected")
		assert.Equal(t, "", state.WhatsAppError)
	})

	t.Run("get status response", func(t *testing.T) {
		state := NewState()
		state.SetWhatsAppStatus("connected")
		state.SetGCalStatus("connected")
		state.SetGCalConfigured(true)

		status := state.GetStatus()

		assert.Equal(t, "connected", status.WhatsApp.Status)
		assert.Equal(t, "connected", status.GCal.Status)
		assert.True(t, status.GCal.Configured)
	})

	t.Run("subscribe and receive updates", func(t *testing.T) {
		state := NewState()
		ch := state.Subscribe()

		go func() {
			time.Sleep(10 * time.Millisecond)
			state.SetWhatsAppStatus("connected")
		}()

		select {
		case update := <-ch:
			assert.Equal(t, "whatsapp_status", update.Type)
			assert.Equal(t, "connected", update.Data)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for update")
		}

		state.Unsubscribe(ch)
	})

	t.Run("mark complete", func(t *testing.T) {
		state := NewState()
		ch := state.Subscribe()

		state.MarkComplete()

		assert.True(t, state.IsComplete())

		// Should receive complete update
		select {
		case update := <-ch:
			assert.Equal(t, "complete", update.Type)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for complete update")
		}

		state.Unsubscribe(ch)
	})

	t.Run("auto complete when all connected", func(t *testing.T) {
		state := NewState()
		ch := state.Subscribe()

		state.SetWhatsAppStatus("connected")
		state.SetGCalStatus("connected")

		assert.True(t, state.IsComplete())

		// Drain channel
		time.Sleep(20 * time.Millisecond)
		state.Unsubscribe(ch)
	})
}

func TestStateManager(t *testing.T) {
	t.Run("get state creates new state if not exists", func(t *testing.T) {
		manager := NewStateManager()

		state1 := manager.GetState(1)
		require.NotNil(t, state1)

		// Getting same user should return same state
		state1Again := manager.GetState(1)
		assert.Equal(t, state1, state1Again)
	})

	t.Run("different users have different states", func(t *testing.T) {
		manager := NewStateManager()

		state1 := manager.GetState(1)
		state2 := manager.GetState(2)

		// Different states
		assert.NotEqual(t, state1, state2)

		// Modifications to one don't affect the other
		state1.SetWhatsAppStatus("connected")
		assert.Equal(t, "connected", state1.WhatsAppStatus)
		assert.Equal(t, "checking", state2.WhatsAppStatus)
	})

	t.Run("remove state", func(t *testing.T) {
		manager := NewStateManager()

		state1 := manager.GetState(1)
		state1.SetWhatsAppStatus("connected")

		manager.RemoveState(1)

		// Getting state again should create a fresh one
		state1New := manager.GetState(1)
		assert.Equal(t, "checking", state1New.WhatsAppStatus) // Fresh state
	})

	t.Run("broadcast to user", func(t *testing.T) {
		manager := NewStateManager()

		state1 := manager.GetState(1)
		state2 := manager.GetState(2)

		ch1 := state1.Subscribe()
		ch2 := state2.Subscribe()

		// Broadcast to user 1 only
		manager.BroadcastToUser(1, Update{Type: "test", Data: "hello"})

		// User 1 should receive
		select {
		case update := <-ch1:
			assert.Equal(t, "test", update.Type)
			assert.Equal(t, "hello", update.Data)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("user 1 didn't receive update")
		}

		// User 2 should not receive
		select {
		case <-ch2:
			t.Fatal("user 2 received update meant for user 1")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}

		state1.Unsubscribe(ch1)
		state2.Unsubscribe(ch2)
	})

	t.Run("get all states", func(t *testing.T) {
		manager := NewStateManager()

		manager.GetState(1)
		manager.GetState(2)
		manager.GetState(3)

		allStates := manager.GetAllStates()
		assert.Len(t, allStates, 3)
		assert.Contains(t, allStates, int64(1))
		assert.Contains(t, allStates, int64(2))
		assert.Contains(t, allStates, int64(3))
	})

	t.Run("concurrent access", func(t *testing.T) {
		manager := NewStateManager()

		done := make(chan bool, 10)

		// Multiple goroutines accessing the manager
		for i := 0; i < 10; i++ {
			go func(userID int64) {
				state := manager.GetState(userID)
				state.SetWhatsAppStatus("connected")
				manager.BroadcastToUser(userID, Update{Type: "test", Data: "data"})
				done <- true
			}(int64(i))
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		allStates := manager.GetAllStates()
		assert.Len(t, allStates, 10)
	})
}
