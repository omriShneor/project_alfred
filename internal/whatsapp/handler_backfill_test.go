package whatsapp

import (
	"sync"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistorySyncBackfillHook_EnabledChannelsOnly(t *testing.T) {
	h := NewHandler(42, nil, false, nil)

	originalDelay := historySyncBackfillDebounce
	historySyncBackfillDebounce = 20 * time.Millisecond
	defer func() { historySyncBackfillDebounce = originalDelay }()

	var mu sync.Mutex
	var called []int64
	h.SetHistorySyncBackfillHook(func(userID int64, channel *database.SourceChannel) {
		require.Equal(t, int64(42), userID)
		mu.Lock()
		called = append(called, channel.ID)
		mu.Unlock()
	})

	h.queueHistorySyncBackfill(map[int64]*database.SourceChannel{
		1: {ID: 1, Enabled: true},
		2: {ID: 2, Enabled: false},
	})

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(called) == 1
	}, time.Second, 10*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int64{1}, called)
}

func TestHistorySyncBackfillHook_DebouncesAcrossChunks(t *testing.T) {
	h := NewHandler(7, nil, false, nil)

	originalDelay := historySyncBackfillDebounce
	historySyncBackfillDebounce = 40 * time.Millisecond
	defer func() { historySyncBackfillDebounce = originalDelay }()

	var mu sync.Mutex
	callCounts := make(map[int64]int)
	h.SetHistorySyncBackfillHook(func(_ int64, channel *database.SourceChannel) {
		mu.Lock()
		callCounts[channel.ID]++
		mu.Unlock()
	})

	// Two quick chunk updates should flush once with both channels.
	h.queueHistorySyncBackfill(map[int64]*database.SourceChannel{
		1: {ID: 1, Enabled: true},
	})
	time.Sleep(10 * time.Millisecond)
	h.queueHistorySyncBackfill(map[int64]*database.SourceChannel{
		2: {ID: 2, Enabled: true},
	})

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(callCounts) == 2
	}, time.Second, 10*time.Millisecond)

	// Give enough time for a potential second flush; counts should remain exactly once.
	time.Sleep(80 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, callCounts[1])
	assert.Equal(t, 1, callCounts[2])
}
