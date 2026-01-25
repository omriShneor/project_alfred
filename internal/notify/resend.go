package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/resend/resend-go/v2"
)

// ResendNotifier sends email notifications via Resend API
type ResendNotifier struct {
	client      *resend.Client
	fromAddress string
	appURL      string
}

// NewResendNotifier creates a new Resend email notifier
func NewResendNotifier(apiKey, from, appURL string) *ResendNotifier {
	if apiKey == "" {
		return nil
	}
	return &ResendNotifier{
		client:      resend.NewClient(apiKey),
		fromAddress: from,
		appURL:      appURL,
	}
}

// IsConfigured returns true if the notifier has server-side config
func (r *ResendNotifier) IsConfigured() bool {
	return r.client != nil && r.fromAddress != ""
}

// Send sends an email notification for a pending event to the specified recipient
func (r *ResendNotifier) Send(ctx context.Context, event *database.CalendarEvent, recipient string) error {
	if recipient == "" {
		return fmt.Errorf("no recipient specified")
	}

	subject := fmt.Sprintf("New Event Pending Approval: %s", event.Title)
	html := r.formatEmailHTML(event)

	params := &resend.SendEmailRequest{
		From:    r.fromAddress,
		To:      []string{recipient},
		Subject: subject,
		Html:    html,
	}

	_, err := r.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("resend send failed: %w", err)
	}

	fmt.Printf("Email notification sent to %s for event: %s\n", recipient, event.Title)
	return nil
}

// Name returns the notifier name
func (r *ResendNotifier) Name() string {
	return "resend"
}

// formatEmailHTML creates the HTML email body
func (r *ResendNotifier) formatEmailHTML(event *database.CalendarEvent) string {
	// Format the start time
	startTimeStr := event.StartTime.Format("Monday, January 2, 2006 at 3:04 PM")

	// Format end time if available
	endTimeStr := ""
	if event.EndTime != nil {
		// If same day, just show the time
		if event.StartTime.Format("2006-01-02") == event.EndTime.Format("2006-01-02") {
			endTimeStr = fmt.Sprintf(" - %s", event.EndTime.Format("3:04 PM"))
		} else {
			endTimeStr = fmt.Sprintf(" - %s", event.EndTime.Format("Monday, January 2, 2006 at 3:04 PM"))
		}
	}

	// Build location section
	locationHTML := ""
	if event.Location != "" {
		locationHTML = fmt.Sprintf(`<p style="margin: 8px 0;"><strong>Location:</strong> %s</p>`, event.Location)
	}

	// Build description section
	descriptionHTML := ""
	if event.Description != "" {
		descriptionHTML = fmt.Sprintf(`<p style="margin: 16px 0;">%s</p>`, event.Description)
	}

	// Build reasoning section
	reasoningHTML := ""
	if event.LLMReasoning != "" {
		reasoningHTML = fmt.Sprintf(`<p style="margin: 16px 0; color: #666; font-style: italic;">Claude's reasoning: %s</p>`, event.LLMReasoning)
	}

	// Build channel source
	channelSource := event.ChannelName
	if channelSource == "" {
		channelSource = "Unknown channel"
	}

	// Action type badge
	actionBadge := "New Event"
	actionColor := "#28a745"
	switch event.ActionType {
	case database.EventActionUpdate:
		actionBadge = "Update Event"
		actionColor = "#ffc107"
	case database.EventActionDelete:
		actionBadge = "Delete Event"
		actionColor = "#dc3545"
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
  <div style="background-color: white; border-radius: 8px; padding: 24px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
    <div style="margin-bottom: 16px;">
      <span style="background-color: %s; color: white; padding: 4px 12px; border-radius: 4px; font-size: 12px; font-weight: 600;">%s</span>
    </div>

    <h2 style="margin: 0 0 16px 0; color: #333;">%s</h2>

    <div style="background: #f8f9fa; padding: 16px; border-radius: 8px; margin: 16px 0; border-left: 4px solid #007bff;">
      <p style="margin: 8px 0;"><strong>Date:</strong> %s%s</p>
      %s
      <p style="margin: 8px 0;"><strong>Source:</strong> %s</p>
    </div>

    %s
    %s

    <a href="%s/events" style="display: inline-block; background: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin-top: 16px; font-weight: 500;">
      Review Event
    </a>

    <hr style="margin-top: 32px; border: none; border-top: 1px solid #eee;">
    <p style="color: #999; font-size: 12px; margin-top: 16px;">
      Project Alfred - Calendar Event Assistant<br>
      <span style="color: #ccc;">Sent at %s</span>
    </p>
  </div>
</body>
</html>`,
		actionColor,
		actionBadge,
		event.Title,
		startTimeStr,
		endTimeStr,
		locationHTML,
		channelSource,
		descriptionHTML,
		reasoningHTML,
		r.appURL,
		time.Now().Format("Jan 2, 2006 3:04 PM"),
	)
}
