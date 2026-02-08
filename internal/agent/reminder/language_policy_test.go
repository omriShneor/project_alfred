package reminder

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
			MessageText: "mañana recuerda llamar a mamá",
		},
	}
	newMessage := database.MessageRecord{
		Timestamp:   time.Date(2026, 2, 8, 10, 1, 0, 0, time.UTC),
		SenderName:  "Bob",
		MessageText: "no olvides enviar el reporte",
	}

	prompt := buildUserPrompt(
		history,
		newMessage,
		nil,
		"Generate fields in Spanish.",
		"Retry in Spanish only.",
	)

	assert.Contains(t, prompt, "## Output Language Requirement")
	assert.Contains(t, prompt, "Generate fields in Spanish.")
	assert.Contains(t, prompt, "## Correction Required")
	assert.Contains(t, prompt, "Retry in Spanish only.")
}

func TestBuildEmailPrompt_IncludesLanguageInstructions(t *testing.T) {
	email := agent.EmailContent{
		From:    "alice@example.com",
		To:      "me@example.com",
		Date:    "2026-02-08",
		Subject: "תזכורת",
		Body:    "תזכורת להגיש את הדוח מחר בבוקר",
	}

	prompt := buildEmailPrompt(email, "Generate fields in Hebrew.", "Retry in Hebrew only.")
	assert.Contains(t, prompt, "## Output Language Requirement")
	assert.Contains(t, prompt, "Generate fields in Hebrew.")
	assert.Contains(t, prompt, "## Correction Required")
	assert.Contains(t, prompt, "Retry in Hebrew only.")
}

func TestShouldRetryReminderForLanguage(t *testing.T) {
	target := langpolicy.TargetLanguage{
		Code:     "es",
		Label:    "Spanish",
		Script:   "latin",
		Reliable: true,
	}

	t.Run("create mismatch triggers retry", func(t *testing.T) {
		analysis := &agent.ReminderAnalysis{
			Action: "create",
			Reminder: &agent.ReminderData{
				Title:       "please submit the report tomorrow",
				Description: "schedule this task with the team",
			},
		}

		retry, validation := shouldRetryReminderForLanguage(target, analysis)
		assert.True(t, retry)
		require.NotEmpty(t, validation.Mismatches)
	})

	t.Run("update match does not retry", func(t *testing.T) {
		analysis := &agent.ReminderAnalysis{
			Action: "update",
			Reminder: &agent.ReminderData{
				Title:       "Enviar reporte",
				Description: "Mañana por la mañana",
			},
		}

		retry, validation := shouldRetryReminderForLanguage(target, analysis)
		assert.False(t, retry)
		assert.True(t, validation.IsMatch())
	})

	t.Run("none action does not retry", func(t *testing.T) {
		analysis := &agent.ReminderAnalysis{
			Action:   "none",
			Reminder: &agent.ReminderData{},
		}
		retry, _ := shouldRetryReminderForLanguage(target, analysis)
		assert.False(t, retry)
	})
}
