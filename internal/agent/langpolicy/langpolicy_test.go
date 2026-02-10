package langpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectTargetLanguage_StrongScripts(t *testing.T) {
	he := DetectTargetLanguage("ניפגש מחר בשעה חמש לפגישה")
	require.True(t, he.Reliable)
	assert.Equal(t, "he", he.Code)
	assert.Equal(t, "hebrew", he.Script)

	ar := DetectTargetLanguage("سنلتقي غدًا الساعة الخامسة")
	require.True(t, ar.Reliable)
	assert.Equal(t, "ar", ar.Code)
	assert.Equal(t, "arabic", ar.Script)

	ru := DetectTargetLanguage("Встретимся завтра в пять часов")
	require.True(t, ru.Reliable)
	assert.Equal(t, "ru", ru.Code)
	assert.Equal(t, "cyrillic", ru.Script)
}

func TestDetectTargetLanguage_LatinHints(t *testing.T) {
	es := DetectTargetLanguage("mañana tenemos reunión con el equipo")
	require.True(t, es.Reliable)
	assert.Equal(t, "es", es.Code)

	fr := DetectTargetLanguage("demain nous avons une réunion très importante")
	require.True(t, fr.Reliable)
	assert.Equal(t, "fr", fr.Code)

	en := DetectTargetLanguage("please schedule a meeting tomorrow afternoon")
	require.True(t, en.Reliable)
	assert.Equal(t, "en", en.Code)
}

func TestDetectTargetLanguage_LowSignal(t *testing.T) {
	unknown := DetectTargetLanguage("12345 !!! 😊")
	assert.False(t, unknown.Reliable)
	assert.Equal(t, "", unknown.Code)
}

func TestDetectTargetLanguage_DoesNotFlipOnSingleForeignToken(t *testing.T) {
	// A single localized word in an otherwise-English body shouldn't flip the result.
	target := DetectTargetLanguage("You've been invited to a Google Calendar event.\nreunião")
	require.True(t, target.Reliable)
	assert.Equal(t, "en", target.Code)
}

func TestValidateFieldsLanguage_MatchAndMismatch(t *testing.T) {
	target := DetectTargetLanguage("mañana tenemos reunión")
	require.True(t, target.Reliable)
	require.Equal(t, "es", target.Code)

	match := ValidateFieldsLanguage(target, map[string]string{
		"title":       "Reunión del equipo",
		"description": "Mañana revisamos el lanzamiento",
		"location":    "Sala central",
	})
	assert.True(t, match.IsMatch())
	assert.Empty(t, match.Mismatches)

	mismatch := ValidateFieldsLanguage(target, map[string]string{
		"title":       "Team meeting tomorrow",
		"description": "Mañana revisamos el lanzamiento",
	})
	assert.False(t, mismatch.IsMatch())
	require.NotEmpty(t, mismatch.Mismatches)
	assert.Equal(t, "title", mismatch.Mismatches[0].Field)
}

func TestValidateFieldsLanguage_SkipsNeutralFields(t *testing.T) {
	target := DetectTargetLanguage("ניפגש מחר")
	require.True(t, target.Reliable)
	require.Equal(t, "he", target.Code)

	result := ValidateFieldsLanguage(target, map[string]string{
		"title":       "פגישת צוות",
		"description": "",
		"location":    "https://zoom.us/j/123",
		"notes":       "Zoom",
		"id_hint":     "123456",
	})
	assert.True(t, result.IsMatch())
	assert.Empty(t, result.Mismatches)
	assert.GreaterOrEqual(t, result.SkippedFields, 3)
}

func TestBuildLanguageInstructions(t *testing.T) {
	target := TargetLanguage{
		Code:     "he",
		Label:    "Hebrew",
		Script:   "hebrew",
		Reliable: true,
	}
	instruction := BuildLanguageInstruction(target)
	assert.Contains(t, instruction, "Hebrew")
	assert.Contains(t, instruction, "Do not translate proper nouns")

	validation := ValidationResult{
		Mismatches: []FieldMismatch{
			{Field: "title"},
			{Field: "description"},
		},
	}
	correction := BuildCorrectiveRetryInstruction(target, validation)
	assert.Contains(t, correction, "title, description")
	assert.Contains(t, correction, "Hebrew")
}
