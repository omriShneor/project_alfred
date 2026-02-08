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

// TodayEvent represents a calendar event for today's schedule display
type TodayEvent struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	AllDay      bool      `json:"all_day"`
	CalendarID  string    `json:"calendar_id"`
}

// ListTodayEvents returns events for the current local day for the specified calendar
func (c *Client) ListTodayEvents(calendarID string) ([]TodayEvent, error) {
	if c.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	// Get start/end of current day in local time.
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	events, err := c.service.Events.List(calendarID).
		TimeMin(startOfDay.Format(time.RFC3339)).
		TimeMax(endOfDay.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list today's events: %w", err)
	}

	result := make([]TodayEvent, 0, len(events.Items))
	for _, item := range events.Items {
		event := TodayEvent{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			Location:    item.Location,
			CalendarID:  calendarID,
		}

		// Handle all-day events (date only, no time)
		if item.Start.Date != "" {
			event.AllDay = true
			startDate, _ := time.ParseInLocation("2006-01-02", item.Start.Date, now.Location())
			endDate, _ := time.ParseInLocation("2006-01-02", item.End.Date, now.Location())
			event.StartTime = startDate
			event.EndTime = endDate
		} else {
			// Regular timed event
			startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			endTime, _ := time.Parse(time.RFC3339, item.End.DateTime)
			event.StartTime = startTime
			event.EndTime = endTime
		}

		result = append(result, event)
	}

	return result, nil
}
