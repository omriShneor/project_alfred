package agent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatAPIErrorInsufficientCredits(t *testing.T) {
	body := []byte(`{"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."},"request_id":"req_123"}`)

	err := formatAPIError(400, body)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInsufficientCredits))
	require.Contains(t, err.Error(), "request_id=req_123")
	require.Contains(t, err.Error(), "console.anthropic.com/settings/plans")
}

func TestFormatAPIErrorStructuredGeneric(t *testing.T) {
	body := []byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},"request_id":"req_456"}`)

	err := formatAPIError(401, body)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrInsufficientCredits))
	require.Contains(t, err.Error(), "status 401")
	require.Contains(t, err.Error(), "request_id=req_456")
	require.Contains(t, err.Error(), "authentication_error")
}

func TestFormatAPIErrorUnstructured(t *testing.T) {
	err := formatAPIError(502, []byte("upstream timeout"))
	require.EqualError(t, err, "API error (status 502): upstream timeout")
}
