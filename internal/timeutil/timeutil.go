package timeutil

import (
	"fmt"
	"time"
)

var defaultLocation = time.UTC

// ResolveLocation returns the user's location with UTC fallback.
func ResolveLocation(timezone string) (*time.Location, bool) {
	if timezone == "" {
		return defaultLocation, true
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return defaultLocation, true
	}
	return loc, false
}

// ParseDateTime parses a datetime in either RFC3339 (with explicit offset) or local layouts in the provided timezone.
func ParseDateTime(value, timezone string) (time.Time, bool, error) {
	if value == "" {
		return time.Time{}, false, fmt.Errorf("time value is required")
	}

	// If timezone/offset exists, preserve it.
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, false, nil
	}

	loc, fallback := ResolveLocation(timezone)

	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, loc); err == nil {
			return t, fallback, nil
		}
	}

	return time.Time{}, fallback, fmt.Errorf("unable to parse time: %s", value)
}

// ParseDateWithDefaultTime parses a date-only string in the provided timezone at a default clock time.
func ParseDateWithDefaultTime(value, timezone string, defaultHour, defaultMinute int) (time.Time, bool, error) {
	if value == "" {
		return time.Time{}, false, fmt.Errorf("date value is required")
	}

	loc, fallback := ResolveLocation(timezone)
	d, err := time.ParseInLocation("2006-01-02", value, loc)
	if err != nil {
		return time.Time{}, fallback, fmt.Errorf("unable to parse date: %s", value)
	}

	return time.Date(d.Year(), d.Month(), d.Day(), defaultHour, defaultMinute, 0, 0, loc), fallback, nil
}

