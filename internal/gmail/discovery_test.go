package gmail

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"User+tag@gmail.com", "user@gmail.com"},
		{"user+tag@googlemail.com", "user@googlemail.com"},
		{"user@example.com", "user@example.com"},
		{" USER@EXAMPLE.COM ", "user@example.com"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, normalizeEmail(tt.in))
	}
}

func TestRecencyMultiplier(t *testing.T) {
	now := time.Now()
	assert.Equal(t, 1.5, recencyMultiplier(now.Add(-10*24*time.Hour)))
	assert.Equal(t, 1.0, recencyMultiplier(now.Add(-60*24*time.Hour)))
	assert.Equal(t, 0.7, recencyMultiplier(now.Add(-200*24*time.Hour)))
	assert.Equal(t, 0.4, recencyMultiplier(now.Add(-800*24*time.Hour)))
}

func TestParseMessageTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ms := now.UnixMilli()

	parsed := parseMessageTime(ms, "")
	assert.WithinDuration(t, now, parsed, time.Second)

	dateHeader := now.Format(time.RFC1123Z)
	parsed = parseMessageTime(0, dateHeader)
	assert.WithinDuration(t, now, parsed, time.Second)
}

func TestSplitAddressList(t *testing.T) {
	list := splitAddressList("A <a@example.com>, b@example.com , \"C\" <c@example.com>")
	assert.Equal(t, []string{"A <a@example.com>", "b@example.com", "\"C\" <c@example.com>"}, list)
}
