package gcal

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

var ErrEventNotFound = errors.New("google calendar event not found")

// IsEventNotFound returns true when a Google Calendar event no longer exists.
func IsEventNotFound(err error) bool {
	return errors.Is(err, ErrEventNotFound)
}

// EventInput represents the input for creating or updating a calendar event
type EventInput struct {
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	Attendees   []string // Email addresses of attendees
}

// EventDetails represents a single Google Calendar event.
type EventDetails struct {
	ID          string
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     *time.Time
	AllDay      bool
	CalendarID  string
	Attendees   []EventAttendee
}

// EventAttendee represents an attendee on a Google Calendar event.
type EventAttendee struct {
	Email       string
	DisplayName string
	Optional    bool
}

func parseGoogleEventTimes(item *calendar.Event, loc *time.Location) (time.Time, time.Time, bool, error) {
	if item == nil || item.Start == nil || item.End == nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("event is missing start or end")
	}

	// All-day events use Date instead of DateTime.
	if item.Start.Date != "" {
		startDate, err := time.ParseInLocation("2006-01-02", item.Start.Date, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("failed to parse all-day start date: %w", err)
		}
		endDate, err := time.ParseInLocation("2006-01-02", item.End.Date, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("failed to parse all-day end date: %w", err)
		}
		return startDate, endDate, true, nil
	}

	if item.Start.DateTime == "" || item.End.DateTime == "" {
		return time.Time{}, time.Time{}, false, fmt.Errorf("event datetime is missing")
	}

	startTime, err := time.Parse(time.RFC3339, item.Start.DateTime)
	if err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("failed to parse start datetime: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, item.End.DateTime)
	if err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("failed to parse end datetime: %w", err)
	}

	return startTime, endTime, false, nil
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

// GetEvent retrieves a single event from Google Calendar.
func (c *Client) GetEvent(calendarID, eventID string) (*EventDetails, error) {
	if c.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}
	if eventID == "" {
		return nil, fmt.Errorf("event id is required")
	}
	if calendarID == "" {
		calendarID = "primary"
	}

	item, err := c.service.Events.Get(calendarID, eventID).Do()
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) && (gErr.Code == http.StatusNotFound || gErr.Code == http.StatusGone) {
			return nil, ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Cancelled means the event was deleted/cancelled on Google Calendar side.
	if item.Status == "cancelled" {
		return nil, ErrEventNotFound
	}

	startTime, endTime, allDay, err := parseGoogleEventTimes(item, time.Now().Location())
	if err != nil {
		return nil, fmt.Errorf("failed to parse event times: %w", err)
	}

	attendees := make([]EventAttendee, 0, len(item.Attendees))
	for _, attendee := range item.Attendees {
		if attendee != nil && attendee.Email != "" {
			attendees = append(attendees, EventAttendee{
				Email:       attendee.Email,
				DisplayName: attendee.DisplayName,
				Optional:    attendee.Optional,
			})
		}
	}

	endCopy := endTime
	return &EventDetails{
		ID:          item.Id,
		Summary:     item.Summary,
		Description: item.Description,
		Location:    item.Location,
		StartTime:   startTime,
		EndTime:     &endCopy,
		AllDay:      allDay,
		CalendarID:  calendarID,
		Attendees:   attendees,
	}, nil
}

// ListEventsInRange returns events in a time window from Google Calendar.
func (c *Client) ListEventsInRange(calendarID string, timeMin, timeMax time.Time) ([]EventDetails, error) {
	if c.service == nil {
		return nil, fmt.Errorf("calendar service not initialized")
	}
	if timeMax.Before(timeMin) {
		return nil, fmt.Errorf("invalid range: time_max is before time_min")
	}
	if calendarID == "" {
		calendarID = "primary"
	}

	var result []EventDetails
	pageToken := ""
	nowLoc := time.Now().Location()

	for {
		call := c.service.Events.List(calendarID).
			TimeMin(timeMin.Format(time.RFC3339)).
			TimeMax(timeMax.Format(time.RFC3339)).
			SingleEvents(true).
			ShowDeleted(false).
			OrderBy("startTime")
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		events, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list events in range: %w", err)
		}

		for _, item := range events.Items {
			if item == nil || item.Status == "cancelled" {
				continue
			}

			startTime, endTime, allDay, parseErr := parseGoogleEventTimes(item, nowLoc)
			if parseErr != nil {
				continue
			}

			attendees := make([]EventAttendee, 0, len(item.Attendees))
			for _, attendee := range item.Attendees {
				if attendee != nil && attendee.Email != "" {
					attendees = append(attendees, EventAttendee{
						Email:       attendee.Email,
						DisplayName: attendee.DisplayName,
						Optional:    attendee.Optional,
					})
				}
			}

			endCopy := endTime
			result = append(result, EventDetails{
				ID:          item.Id,
				Summary:     item.Summary,
				Description: item.Description,
				Location:    item.Location,
				StartTime:   startTime,
				EndTime:     &endCopy,
				AllDay:      allDay,
				CalendarID:  calendarID,
				Attendees:   attendees,
			})
		}

		if events.NextPageToken == "" {
			break
		}
		pageToken = events.NextPageToken
	}

	return result, nil
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

		startTime, endTime, allDay, parseErr := parseGoogleEventTimes(item, now.Location())
		if parseErr != nil {
			// Skip malformed events rather than failing the whole request.
			continue
		}
		event.AllDay = allDay
		event.StartTime = startTime
		event.EndTime = endTime

		result = append(result, event)
	}

	return result, nil
}
