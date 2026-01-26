package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
)

const expoPushURL = "https://exp.host/--/api/v2/push/send"

// ExpoPushNotifier sends push notifications via Expo Push Notification Service
type ExpoPushNotifier struct {
	httpClient *http.Client
}

// NewExpoPushNotifier creates a new Expo push notifier
func NewExpoPushNotifier() *ExpoPushNotifier {
	return &ExpoPushNotifier{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name
func (e *ExpoPushNotifier) Name() string {
	return "expo_push"
}

// IsConfigured returns true - Expo push doesn't require server-side credentials
func (e *ExpoPushNotifier) IsConfigured() bool {
	return true
}

// expoPushMessage represents the message format for Expo Push API
type expoPushMessage struct {
	To       string                 `json:"to"`
	Title    string                 `json:"title"`
	Body     string                 `json:"body"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Sound    string                 `json:"sound,omitempty"`
	Badge    int                    `json:"badge,omitempty"`
	Priority string                 `json:"priority,omitempty"`
}

// Send sends a push notification for a pending event
func (e *ExpoPushNotifier) Send(ctx context.Context, event *database.CalendarEvent, recipient string) error {
	if recipient == "" {
		return fmt.Errorf("no push token specified")
	}

	// Determine title based on action type
	title := "New Event Detected"
	switch event.ActionType {
	case database.EventActionUpdate:
		title = "Event Update Detected"
	case database.EventActionDelete:
		title = "Event Deletion Detected"
	}

	// Format the body with event title and date
	body := event.Title
	if !event.StartTime.IsZero() {
		body = fmt.Sprintf("%s - %s", event.Title, event.StartTime.Format("Mon, Jan 2 at 3:04 PM"))
	}

	message := expoPushMessage{
		To:    recipient,
		Title: title,
		Body:  body,
		Sound: "default",
		Data: map[string]interface{}{
			"eventId":    event.ID,
			"actionType": string(event.ActionType),
			"screen":     "Events",
		},
		Priority: "high",
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal push message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", expoPushURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expo push API returned status %d", resp.StatusCode)
	}

	fmt.Printf("Push notification sent to %s for event: %s\n", recipient, event.Title)
	return nil
}
