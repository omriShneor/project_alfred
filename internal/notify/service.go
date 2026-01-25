package notify

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/database"
)

// Service orchestrates notifications based on user preferences
type Service struct {
	db            *database.DB
	emailNotifier Notifier
	// Future: pushNotifier, smsNotifier, webhookNotifier
}

// NewService creates a notification service
func NewService(db *database.DB, emailNotifier Notifier) *Service {
	return &Service{
		db:            db,
		emailNotifier: emailNotifier,
	}
}

// NotifyPendingEvent sends notifications for a new pending event
// based on user preferences. Errors are logged but don't fail the operation.
func (s *Service) NotifyPendingEvent(ctx context.Context, event *database.CalendarEvent) {
	fmt.Printf("Notification: Processing event %d (%s)\n", event.ID, event.Title)

	prefs, err := s.db.GetUserNotificationPrefs()
	if err != nil {
		fmt.Printf("Notification: Failed to get prefs: %v\n", err)
		return
	}

	fmt.Printf("Notification: Prefs loaded - email_enabled=%v, email_address=%q\n",
		prefs.EmailEnabled, prefs.EmailAddress)

	// Email notification
	if prefs.EmailEnabled && prefs.EmailAddress != "" {
		if s.emailNotifier != nil && s.emailNotifier.IsConfigured() {
			fmt.Printf("Notification: Sending email to %s\n", prefs.EmailAddress)
			if err := s.emailNotifier.Send(ctx, event, prefs.EmailAddress); err != nil {
				fmt.Printf("Notification: Email failed: %v\n", err)
			} else {
				fmt.Printf("Notification: Email sent successfully\n")
			}
		} else {
			fmt.Println("Notification: Email enabled but server not configured (no API key)")
		}
	} else {
		fmt.Println("Notification: Email not enabled or no address configured")
	}

	// Future: Push notification
	// if prefs.PushEnabled && prefs.PushToken != "" && s.pushNotifier != nil { ... }

	// Future: SMS notification
	// if prefs.SMSEnabled && prefs.SMSPhone != "" && s.smsNotifier != nil { ... }

	// Future: Webhook notification
	// if prefs.WebhookEnabled && prefs.WebhookURL != "" && s.webhookNotifier != nil { ... }
}

// IsEmailAvailable returns true if email notifications can be used
func (s *Service) IsEmailAvailable() bool {
	return s.emailNotifier != nil && s.emailNotifier.IsConfigured()
}
