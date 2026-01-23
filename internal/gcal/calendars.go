package gcal

import (
	"fmt"
)

// CalendarInfo represents a Google Calendar
type CalendarInfo struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Primary     bool   `json:"primary"`
	AccessRole  string `json:"access_role"`
}

// ListCalendars returns all calendars the user has access to
func (c *Client) ListCalendars() ([]CalendarInfo, error) {
	if c.service == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	list, err := c.service.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	var calendars []CalendarInfo
	for _, item := range list.Items {
		calendars = append(calendars, CalendarInfo{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			Primary:     item.Primary,
			AccessRole:  item.AccessRole,
		})
	}

	return calendars, nil
}
