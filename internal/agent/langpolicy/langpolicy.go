package langpolicy

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type TargetLanguage struct {
	Code       string
	Label      string
	Script     string
	Confidence float64
	Reliable   bool
}

type FieldMismatch struct {
	Field        string
	DetectedCode string
	DetectedName string
	Reason       string
}

type ValidationResult struct {
	CheckedFields int
	MatchedFields int
	SkippedFields int
	Mismatches    []FieldMismatch
}

func (r ValidationResult) IsMatch() bool {
	return len(r.Mismatches) == 0
}

var (
	tokenPattern = regexp.MustCompile(`[\p{L}\p{M}]+`)
	urlPattern   = regexp.MustCompile(`(?i)(https?://|www\.)`)
	emailPattern = regexp.MustCompile(`(?i)\b[\w.%+\-]+@[\w.\-]+\.[a-z]{2,}\b`)
)

var latinKeywordHints = map[string]map[string]struct{}{
	"en": {
		"tomorrow": {}, "today": {}, "meeting": {}, "please": {}, "remind": {}, "with": {}, "about": {}, "schedule": {},
	},
	"es": {
		"mañana": {}, "hoy": {}, "reunión": {}, "reunion": {}, "recordar": {}, "equipo": {}, "gracias": {}, "por": {},
	},
	"fr": {
		"demain": {}, "aujourd": {}, "réunion": {}, "rappel": {}, "bonjour": {}, "avec": {}, "merci": {}, "pour": {},
	},
	"pt": {
		"amanhã": {}, "hoje": {}, "reunião": {}, "lembrete": {}, "obrigado": {}, "com": {}, "para": {}, "equipe": {},
	},
	"de": {
		"morgen": {}, "heute": {}, "besprechung": {}, "erinnerung": {}, "danke": {}, "mit": {}, "bitte": {}, "termin": {},
	},
	"it": {
		"domani": {}, "oggi": {}, "riunione": {}, "promemoria": {}, "grazie": {}, "con": {}, "per": {}, "incontro": {},
	},
}

func DetectTargetLanguage(text string) TargetLanguage {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return TargetLanguage{Label: "Unknown"}
	}

	scriptCounts, totalLetters := countScripts(trimmed)
	if totalLetters == 0 {
		return TargetLanguage{Label: "Unknown"}
	}

	dominantScript, dominantCount := dominantScript(scriptCounts)
	dominantRatio := float64(dominantCount) / float64(totalLetters)

	// Strong script signal
	if dominantScript == "hebrew" && dominantCount >= 2 && dominantRatio >= 0.35 {
		return buildTarget("he", "hebrew", true, confidenceFromRatio(dominantRatio))
	}
	if dominantScript == "arabic" && dominantCount >= 2 && dominantRatio >= 0.35 {
		return buildTarget("ar", "arabic", true, confidenceFromRatio(dominantRatio))
	}
	if dominantScript == "cyrillic" && dominantCount >= 2 && dominantRatio >= 0.35 {
		// We treat Cyrillic as Russian-family in this first version.
		return buildTarget("ru", "cyrillic", true, confidenceFromRatio(dominantRatio))
	}

	latinCount := scriptCounts["latin"]
	if latinCount == 0 {
		return TargetLanguage{Label: "Unknown"}
	}

	lower := strings.ToLower(trimmed)
	if code, ok := detectLatinBySpecialChars(lower); ok {
		return buildTarget(code, "latin", true, 0.9)
	}

	bestCode, bestScore, secondScore := detectLatinByKeywords(lower)
	if bestScore >= 2 && bestScore >= secondScore+1 {
		confidence := 0.72 + float64(bestScore-secondScore)*0.08
		if confidence > 0.95 {
			confidence = 0.95
		}
		return buildTarget(bestCode, "latin", true, confidence)
	}

	// Default fallback for long Latin text when hints are weak.
	wordCount := len(tokenPattern.FindAllString(lower, -1))
	if latinCount >= 8 && wordCount >= 2 {
		return buildTarget("en", "latin", true, 0.62)
	}

	return TargetLanguage{
		Code:       "",
		Label:      "Unknown",
		Script:     "latin",
		Confidence: 0.45,
		Reliable:   false,
	}
}

func ValidateFieldsLanguage(target TargetLanguage, fields map[string]string) ValidationResult {
	result := ValidationResult{}
	if !target.Reliable || target.Code == "" {
		return result
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := strings.TrimSpace(fields[key])
		if value == "" {
			result.SkippedFields++
			continue
		}
		if isNeutralField(value) {
			result.SkippedFields++
			continue
		}

		result.CheckedFields++
		detected := DetectTargetLanguage(value)
		if !detected.Reliable || detected.Code == "" {
			result.SkippedFields++
			continue
		}

		if isLanguageCompatible(target, detected) {
			result.MatchedFields++
			continue
		}

		reason := fmt.Sprintf("expected %s, got %s", target.Label, detected.Label)
		if target.Script != detected.Script {
			reason = fmt.Sprintf("expected %s script, got %s script", target.Script, detected.Script)
		}
		result.Mismatches = append(result.Mismatches, FieldMismatch{
			Field:        key,
			DetectedCode: detected.Code,
			DetectedName: detected.Label,
			Reason:       reason,
		})
	}

	return result
}

func BuildLanguageInstruction(target TargetLanguage) string {
	if !target.Reliable || target.Code == "" {
		return ""
	}

	return fmt.Sprintf(
		"Generate all user-facing text fields (title, description, and location when applicable) in %s (%s), matching the latest triggering discussion language. Do not translate proper nouns, URLs, email addresses, or quoted literals.",
		target.Label,
		target.Code,
	)
}

func BuildCorrectiveRetryInstruction(target TargetLanguage, validation ValidationResult) string {
	if !target.Reliable || target.Code == "" {
		return ""
	}

	mismatchFields := make([]string, 0, len(validation.Mismatches))
	for _, mismatch := range validation.Mismatches {
		mismatchFields = append(mismatchFields, mismatch.Field)
	}

	fieldText := "the user-facing text fields"
	if len(mismatchFields) > 0 {
		fieldText = strings.Join(mismatchFields, ", ")
	}

	return fmt.Sprintf(
		"Your previous output language did not match. Re-run and return %s in %s (%s). Keep proper nouns, URLs, email addresses, and quoted literals unchanged.",
		fieldText,
		target.Label,
		target.Code,
	)
}

func buildTarget(code, script string, reliable bool, confidence float64) TargetLanguage {
	label := languageLabel(code)
	return TargetLanguage{
		Code:       code,
		Label:      label,
		Script:     script,
		Confidence: confidence,
		Reliable:   reliable,
	}
}

func languageLabel(code string) string {
	switch code {
	case "he":
		return "Hebrew"
	case "ar":
		return "Arabic"
	case "ru":
		return "Russian"
	case "es":
		return "Spanish"
	case "fr":
		return "French"
	case "pt":
		return "Portuguese"
	case "de":
		return "German"
	case "it":
		return "Italian"
	case "en":
		return "English"
	default:
		return "Unknown"
	}
}

func confidenceFromRatio(ratio float64) float64 {
	confidence := 0.7 + ratio*0.28
	if confidence > 0.98 {
		confidence = 0.98
	}
	return confidence
}

func dominantScript(counts map[string]int) (string, int) {
	bestScript := ""
	bestCount := 0
	for _, script := range []string{"hebrew", "arabic", "cyrillic", "latin"} {
		count := counts[script]
		if count > bestCount {
			bestCount = count
			bestScript = script
		}
	}
	return bestScript, bestCount
}

func countScripts(text string) (map[string]int, int) {
	counts := map[string]int{
		"hebrew":   0,
		"arabic":   0,
		"cyrillic": 0,
		"latin":    0,
	}

	totalLetters := 0
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		totalLetters++

		switch {
		case unicode.Is(unicode.Hebrew, r):
			counts["hebrew"]++
		case unicode.Is(unicode.Arabic, r):
			counts["arabic"]++
		case unicode.Is(unicode.Cyrillic, r):
			counts["cyrillic"]++
		case unicode.Is(unicode.Latin, r):
			counts["latin"]++
		}
	}

	return counts, totalLetters
}

func detectLatinBySpecialChars(text string) (string, bool) {
	// Special characters are a strong hint, but a single token in a different
	// language (e.g., a localized footer in an otherwise-English email invite)
	// should not flip the entire detection.
	//
	// We count occurrences and require a small threshold before accepting.
	counts := map[string]int{
		"pt": 0,
		"de": 0,
		"fr": 0,
		"it": 0,
		"es": 0,
	}

	for _, r := range text {
		switch r {
		// Portuguese
		case 'ã', 'õ':
			counts["pt"]++
		// German
		case 'ä', 'ö', 'ü', 'ß':
			counts["de"]++
		// French
		// Note: we intentionally exclude 'ü' here since it's ambiguous and also a strong
		// German hint; keyword detection can disambiguate if needed.
		case 'à', 'â', 'ç', 'è', 'ê', 'ë', 'î', 'ï', 'ô', 'û', 'ù', 'ÿ', 'œ', 'æ':
			counts["fr"]++
		// Italian
		case 'ì', 'ò':
			counts["it"]++
		// Spanish
		case 'ñ', '¿', '¡', 'á', 'í', 'ó', 'ú':
			counts["es"]++
		}
	}

	bestCode := ""
	bestCount := 0
	secondCount := 0
	for _, code := range []string{"pt", "de", "fr", "it", "es"} {
		c := counts[code]
		if c > bestCount {
			secondCount = bestCount
			bestCount = c
			bestCode = code
			continue
		}
		if c > secondCount {
			secondCount = c
		}
	}

	// Require more than a single special character to prevent false positives.
	if bestCount >= 2 && bestCount >= secondCount+1 {
		return bestCode, true
	}
	return "", false
}

func detectLatinByKeywords(text string) (bestCode string, bestScore int, secondScore int) {
	tokens := tokenPattern.FindAllString(strings.ToLower(text), -1)
	if len(tokens) == 0 {
		return "", 0, 0
	}

	scores := make(map[string]int, len(latinKeywordHints))
	for _, token := range tokens {
		for langCode, hints := range latinKeywordHints {
			if _, ok := hints[token]; ok {
				scores[langCode]++
			}
		}
	}

	for langCode, score := range scores {
		if score > bestScore {
			secondScore = bestScore
			bestScore = score
			bestCode = langCode
			continue
		}
		if score > secondScore {
			secondScore = score
		}
	}

	return bestCode, bestScore, secondScore
}

func isNeutralField(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}

	if emailPattern.MatchString(trimmed) || urlPattern.MatchString(trimmed) {
		return true
	}

	letters := countLetterRunes(trimmed)
	if letters == 0 {
		return true
	}

	// Short single-token values are often names/brands (e.g., Zoom, WeWork) and should not be forced.
	tokens := tokenPattern.FindAllString(trimmed, -1)
	if len(tokens) <= 1 && letters <= 8 {
		return true
	}

	return false
}

func countLetterRunes(text string) int {
	count := 0
	for _, r := range text {
		if unicode.IsLetter(r) {
			count++
		}
	}
	return count
}

func isLanguageCompatible(target TargetLanguage, detected TargetLanguage) bool {
	if target.Script != "latin" {
		return target.Script == detected.Script
	}

	if detected.Script != "latin" {
		return false
	}

	if target.Code == "" || detected.Code == "" {
		return true
	}
	if target.Code == detected.Code {
		return true
	}

	// Avoid false positives for weakly detected Latin variants.
	return detected.Confidence < 0.8
}
