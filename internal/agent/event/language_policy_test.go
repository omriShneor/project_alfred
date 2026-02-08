package event

import (
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/agent/langpolicy"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUserPrompt_IncludesLanguageInstructions(t *testing.T) {
	history := []database.MessageRecord{
		{
			Timestamp:   time.Date(2026, 2, 8, 10, 0, 0, 0, time.UTC),
			SenderName:  "Alice",
			MessageText: "ניפגש מחר",
		},
	}
	newMessage := database.MessageRecord{
		Timestamp:   time.Date(2026, 2, 8, 10, 1, 0, 0, time.UTC),
		SenderName:  "Bob",
		MessageText: "בשעה חמש במשרד",
	}

	prompt := buildUserPrompt(
		history,
		newMessage,
		nil,
		"Generate fields in Hebrew.",
		"Retry in Hebrew only.",
	)

	assert.Contains(t, prompt, "## Output Language Requirement")
	assert.Contains(t, prompt, "Generate fields in Hebrew.")
	assert.Contains(t, prompt, "## Correction Required")
	assert.Contains(t, prompt, "Retry in Hebrew only.")
}

func TestBuildEmailPrompt_IncludesLanguageInstructions(t *testing.T) {
	email := agent.EmailContent{
		From:    "alice@example.com",
		To:      "me@example.com",
		Date:    "2026-02-08",
		Subject: "Reunión",
		Body:    "mañana tenemos reunión con el equipo",
	}

	prompt := buildEmailPrompt(email, "Generate fields in Spanish.", "Retry in Spanish only.")
	assert.Contains(t, prompt, "## Output Language Requirement")
	assert.Contains(t, prompt, "Generate fields in Spanish.")
	assert.Contains(t, prompt, "## Correction Required")
	assert.Contains(t, prompt, "Retry in Spanish only.")
}

func TestShouldRetryEventForLanguage(t *testing.T) {
	target := langpolicy.TargetLanguage{
		Code:     "he",
		Label:    "Hebrew",
		Script:   "hebrew",
		Reliable: true,
	}

	t.Run("create mismatch triggers retry", func(t *testing.T) {
		analysis := &agent.EventAnalysis{
			Action: "create",
			Event: &agent.EventData{
				Title:       "Team meeting tomorrow",
				Description: "Weekly sync",
				Location:    "Tel Aviv Office",
			},
		}

		retry, validation := shouldRetryEventForLanguage(target, analysis)
		assert.True(t, retry)
		require.NotEmpty(t, validation.Mismatches)
	})

	t.Run("create match does not retry", func(t *testing.T) {
		analysis := &agent.EventAnalysis{
			Action: "create",
			Event: &agent.EventData{
				Title:       "פגישת צוות",
				Description: "מחר נבדוק את ההשקה",
				Location:    "משרד תל אביב",
			},
		}

		retry, validation := shouldRetryEventForLanguage(target, analysis)
		assert.False(t, retry)
		assert.True(t, validation.IsMatch())
	})

	t.Run("delete action does not retry", func(t *testing.T) {
		analysis := &agent.EventAnalysis{
			Action: "delete",
			Event:  &agent.EventData{AlfredEventRef: 12},
		}
		retry, _ := shouldRetryEventForLanguage(target, analysis)
		assert.False(t, retry)
	})

	t.Run("unreliable target does not retry", func(t *testing.T) {
		analysis := &agent.EventAnalysis{
			Action: "create",
			Event:  &agent.EventData{Title: "Team sync"},
		}
		retry, _ := shouldRetryEventForLanguage(langpolicy.TargetLanguage{}, analysis)
		assert.False(t, retry)
	})
}
