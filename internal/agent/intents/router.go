package intents

import (
	"context"
	"strings"
)

// RoutedIntent is the router decision for which module should run.
type RoutedIntent struct {
	Intent     string
	Confidence float64
	Reasoning  string
}

// Router routes input into an intent.
type Router interface {
	RouteMessages(ctx context.Context, in MessageInput) RoutedIntent
	RouteEmail(ctx context.Context, in EmailInput) RoutedIntent
}

// KeywordRouter is a lightweight deterministic intent router.
type KeywordRouter struct{}

func NewKeywordRouter() *KeywordRouter {
	return &KeywordRouter{}
}

func (r *KeywordRouter) RouteMessages(_ context.Context, in MessageInput) RoutedIntent {
	return routeText(in.NewMessage.MessageText)
}

func (r *KeywordRouter) RouteEmail(_ context.Context, in EmailInput) RoutedIntent {
	text := strings.TrimSpace(in.Email.Subject + "\n" + in.Email.Body)
	return routeText(text)
}

func routeText(text string) RoutedIntent {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return RoutedIntent{Intent: "none", Confidence: 1, Reasoning: "empty input"}
	}

	reminderHints := []string{
		"remind me", "don't forget", "dont forget", "todo", "to do",
		"need to", "must", "remember to", "task",
	}
	eventHints := []string{
		"meeting", "call", "appointment", "schedule", "reschedule",
		"cancel", "lunch", "dinner", "interview", "calendar",
	}

	hasReminder := containsAny(normalized, reminderHints)
	hasEvent := containsAny(normalized, eventHints)

	switch {
	case hasEvent && hasReminder:
		return RoutedIntent{Intent: "both", Confidence: 0.6, Reasoning: "contains event and reminder cues"}
	case hasEvent:
		return RoutedIntent{Intent: "event", Confidence: 0.75, Reasoning: "contains event cues"}
	case hasReminder:
		return RoutedIntent{Intent: "reminder", Confidence: 0.75, Reasoning: "contains reminder cues"}
	default:
		return RoutedIntent{Intent: "none", Confidence: 0.8, Reasoning: "no strong event/reminder cues"}
	}
}

func containsAny(text string, values []string) bool {
	for _, v := range values {
		if strings.Contains(text, v) {
			return true
		}
	}
	return false
}

