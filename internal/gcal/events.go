package gcal

import (
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
)

// EventInput represents the input for creating or updating a calendar event
type EventInput struct {
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	Attendees   []string // Email addresses of attendees
}

// CreateEvent creates a new event in Google Calendar and returns the event ID
func (c *Client) CreateEvent(calendarID string, input EventInput) (string, error) {
	if c.service == nil {
		return "", fmt.Errorf("calendar service not initialized")
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	// RFC3339 format includes timezone offset, so Google Calendar can infer the timezone
	event := &calendar.Event{
		Summary:     input.Summary,
		Description: input.Description,
		Location:    input.Location,
		Start: &calendar.EventDateTime{
			DateTime: input.StartTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: input.EndTime.Format(time.RFC3339),
		},
	}

	// Add attendees if provided
	if len(input.Attendees) > 0 {
		attendees := make([]*calendar.EventAttendee, len(input.Attendees))
		for i, email := range input.Attendees {
			attendees[i] = &calendar.EventAttendee{Email: email}
		}
		event.Attendees = attendees
	}

	// SendUpdates sends notifications to attendees
	created, err := c.service.Events.Insert(calendarID, event).SendUpdates("all").Do()
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}

	return created.Id, nil
}

// UpdateEvent updates an existing event in Google Calendar
func (c *Client) UpdateEvent(calendarID, eventID string, input EventInput) error {
	if c.service == nil {
		return fmt.Errorf("calendar service not initialized")
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	event := &calendar.Event{
		Summary:     input.Summary,
		Description: input.Description,
		Location:    input.Location,
		Start: &calendar.EventDateTime{
			DateTime: input.StartTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: input.EndTime.Format(time.RFC3339),
		},
	}

	// Add attendees if provided
	if len(input.Attendees) > 0 {
		attendees := make([]*calendar.EventAttendee, len(input.Attendees))
		for i, email := range input.Attendees {
			attendees[i] = &calendar.EventAttendee{Email: email}
		}
		event.Attendees = attendees
	}

	_, err := c.service.Events.Update(calendarID, eventID, event).SendUpdates("all").Do()
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// DeleteEvent deletes an event from Google Calendar
func (c *Client) DeleteEvent(calendarID, eventID string) error {
	if c.service == nil {
		return fmt.Errorf("calendar service not initialized")
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	err := c.service.Events.Delete(calendarID, eventID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}
