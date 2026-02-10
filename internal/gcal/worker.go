package gcal

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
)

// SyncDBInterface defines DB operations needed by the Google Calendar sync worker.
type SyncDBInterface interface {
	GetGCalSettings(userID int64) (*database.GCalSettings, error)
	EnsureGoogleCalendarImportChannel(userID int64) (*database.SourceChannel, error)
	ListSyncedEventsWithGoogleID(userID int64) ([]database.CalendarEvent, error)
	GetEventByGoogleIDForUser(userID int64, googleEventID string) (*database.CalendarEvent, error)
	CreatePendingEvent(event *database.CalendarEvent) (*database.CalendarEvent, error)
	UpdateSyncedEventFromGoogle(id int64, title, description string, startTime time.Time, endTime *time.Time, location string) error
	UpdateEventStatus(id int64, status database.EventStatus) error
	SetEventAttendees(eventID int64, attendees []database.Attendee) error
}

const (
	importLookbackDays  = 30
	importLookaheadDays = 365
	stopWaitTimeout     = 5 * time.Second
)

// Worker periodically syncs Google Calendar changes back to Alfred.
type Worker struct {
	client       *Client
	db           SyncDBInterface
	userID       int64
	pollInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// WorkerConfig contains configuration for the Google Calendar worker.
type WorkerConfig struct {
	UserID              int64
	PollIntervalMinutes int
}

// NewWorker creates a new Google Calendar sync worker.
func NewWorker(client *Client, db SyncDBInterface, config WorkerConfig) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	pollInterval := time.Duration(config.PollIntervalMinutes) * time.Minute
	if pollInterval <= 0 {
		pollInterval = 1 * time.Minute
	}

	return &Worker{
		client:       client,
		db:           db,
		userID:       config.UserID,
		pollInterval: pollInterval,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the periodic Google Calendar sync loop.
func (w *Worker) Start() error {
	if w.client == nil || !w.client.IsAuthenticated() {
		fmt.Println("Google Calendar worker: client not authenticated, will poll when authenticated")
	}

	fmt.Printf("Google Calendar worker: starting with %v poll interval\n", w.pollInterval)

	w.wg.Add(1)
	go w.pollLoop()

	return nil
}

// Stop gracefully shuts down the worker.
func (w *Worker) Stop() {
	fmt.Println("Google Calendar worker: stopping...")
	w.cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("Google Calendar worker: stopped")
	case <-time.After(stopWaitTimeout):
		fmt.Printf("Google Calendar worker: stop timed out after %v; continuing shutdown\n", stopWaitTimeout)
	}
}

// SetClient updates the Google Calendar client (used when auth/scopes change).
func (w *Worker) SetClient(client *Client) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.client = client
}

// PollNow triggers an immediate sync cycle.
func (w *Worker) PollNow() {
	go w.poll()
}

func (w *Worker) pollLoop() {
	defer w.wg.Done()

	// Do an initial poll after a short delay so startup path is not blocked.
	select {
	case <-w.ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.poll()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.poll()
		}
	}
}

func (w *Worker) poll() {
	if w.userID == 0 {
		return
	}

	w.mu.Lock()
	client := w.client
	w.mu.Unlock()

	if client == nil || !client.IsAuthenticated() {
		return
	}

	settings, err := w.db.GetGCalSettings(w.userID)
	if err != nil {
		fmt.Printf("Google Calendar worker: failed to get settings: %v\n", err)
		return
	}
	if settings == nil || !settings.SyncEnabled {
		return
	}

	events, err := w.db.ListSyncedEventsWithGoogleID(w.userID)
	if err != nil {
		fmt.Printf("Google Calendar worker: failed to list synced events: %v\n", err)
		return
	}
	if len(events) == 0 {
		events = []database.CalendarEvent{}
	}

	linkedGoogleIDs := make(map[string]struct{}, len(events))
	for _, event := range events {
		if event.GoogleEventID != nil && *event.GoogleEventID != "" {
			linkedGoogleIDs[*event.GoogleEventID] = struct{}{}
		}
	}

	for _, event := range events {
		if event.GoogleEventID == nil || *event.GoogleEventID == "" {
			continue
		}

		calendarID := event.CalendarID
		if calendarID == "" {
			calendarID = settings.SelectedCalendarID
		}
		if calendarID == "" {
			calendarID = "primary"
		}

		googleEvent, err := client.GetEvent(calendarID, *event.GoogleEventID)
		if err != nil && IsEventNotFound(err) {
			// Fallback to selected calendar and primary before marking deleted.
			fallbackCalendarIDs := []string{settings.SelectedCalendarID, "primary"}
			for _, fallbackID := range fallbackCalendarIDs {
				if fallbackID == "" || fallbackID == calendarID {
					continue
				}

				fallbackEvent, fallbackErr := client.GetEvent(fallbackID, *event.GoogleEventID)
				if fallbackErr == nil {
					googleEvent = fallbackEvent
					err = nil
					break
				}
				if !IsEventNotFound(fallbackErr) {
					err = fallbackErr
					break
				}
			}
		}
		if err != nil {
			if IsEventNotFound(err) {
				if event.Status != database.EventStatusDeleted {
					if updateErr := w.db.UpdateEventStatus(event.ID, database.EventStatusDeleted); updateErr != nil {
						fmt.Printf("Google Calendar worker: failed to mark event %d as deleted: %v\n", event.ID, updateErr)
					}
				}
				continue
			}

			fmt.Printf("Google Calendar worker: failed to fetch google event %s: %v\n", *event.GoogleEventID, err)
			continue
		}

		if shouldUpdateLocalEvent(event, googleEvent) {
			if updateErr := w.db.UpdateSyncedEventFromGoogle(
				event.ID,
				googleEvent.Summary,
				googleEvent.Description,
				googleEvent.StartTime,
				googleEvent.EndTime,
				googleEvent.Location,
			); updateErr != nil {
				fmt.Printf("Google Calendar worker: failed to update event %d from Google: %v\n", event.ID, updateErr)
				continue
			}
		}

		googleAttendees := make([]database.Attendee, 0, len(googleEvent.Attendees))
		for _, attendee := range googleEvent.Attendees {
			email := strings.TrimSpace(attendee.Email)
			if email == "" {
				continue
			}

			googleAttendees = append(googleAttendees, database.Attendee{
				Email:       email,
				DisplayName: strings.TrimSpace(attendee.DisplayName),
				Optional:    attendee.Optional,
			})
		}

		if attendeesChanged(event.Attendees, googleAttendees) {
			if attendeeErr := w.db.SetEventAttendees(event.ID, googleAttendees); attendeeErr != nil {
				fmt.Printf("Google Calendar worker: failed to sync attendees for event %d: %v\n", event.ID, attendeeErr)
			}
		}
	}

	w.importMissingGoogleEvents(client, settings, linkedGoogleIDs)
}

func (w *Worker) importMissingGoogleEvents(client *Client, settings *database.GCalSettings, linkedGoogleIDs map[string]struct{}) {
	if client == nil || settings == nil {
		return
	}

	importChannel, err := w.db.EnsureGoogleCalendarImportChannel(w.userID)
	if err != nil || importChannel == nil {
		fmt.Printf("Google Calendar worker: failed to ensure import channel: %v\n", err)
		return
	}

	targetCalendarID := settings.SelectedCalendarID
	if targetCalendarID == "" {
		targetCalendarID = "primary"
	}

	now := time.Now()
	rangeStart := now.AddDate(0, 0, -importLookbackDays)
	rangeEnd := now.AddDate(0, 0, importLookaheadDays)

	remoteEvents, err := client.ListEventsInRange(targetCalendarID, rangeStart, rangeEnd)
	if err != nil && targetCalendarID != "primary" {
		remoteEvents, err = client.ListEventsInRange("primary", rangeStart, rangeEnd)
		targetCalendarID = "primary"
	}
	if err != nil {
		fmt.Printf("Google Calendar worker: failed to list remote events for import: %v\n", err)
		return
	}

	for _, remoteEvent := range remoteEvents {
		if remoteEvent.ID == "" {
			continue
		}
		if _, exists := linkedGoogleIDs[remoteEvent.ID]; exists {
			continue
		}

		existing, err := w.db.GetEventByGoogleIDForUser(w.userID, remoteEvent.ID)
		if err != nil {
			fmt.Printf("Google Calendar worker: failed to lookup event by google id %s: %v\n", remoteEvent.ID, err)
			continue
		}
		if existing != nil {
			linkedGoogleIDs[remoteEvent.ID] = struct{}{}
			continue
		}

		eventTitle := strings.TrimSpace(remoteEvent.Summary)
		if eventTitle == "" {
			eventTitle = "Untitled event"
		}

		eventIDCopy := remoteEvent.ID
		calendarID := remoteEvent.CalendarID
		if calendarID == "" {
			calendarID = targetCalendarID
		}
		if calendarID == "" {
			calendarID = "primary"
		}

		importedEvent, createErr := w.db.CreatePendingEvent(&database.CalendarEvent{
			UserID:        w.userID,
			ChannelID:     importChannel.ID,
			GoogleEventID: &eventIDCopy,
			CalendarID:    calendarID,
			Title:         eventTitle,
			Description:   remoteEvent.Description,
			StartTime:     remoteEvent.StartTime,
			EndTime:       remoteEvent.EndTime,
			Location:      remoteEvent.Location,
			ActionType:    database.EventActionCreate,
		})
		if createErr != nil {
			fmt.Printf("Google Calendar worker: failed to import new event %s: %v\n", remoteEvent.ID, createErr)
			continue
		}

		if statusErr := w.db.UpdateEventStatus(importedEvent.ID, database.EventStatusSynced); statusErr != nil {
			fmt.Printf("Google Calendar worker: failed to mark imported event %d as synced: %v\n", importedEvent.ID, statusErr)
			continue
		}

		attendees := make([]database.Attendee, 0, len(remoteEvent.Attendees))
		for _, attendee := range remoteEvent.Attendees {
			email := strings.TrimSpace(attendee.Email)
			if email == "" {
				continue
			}

			attendees = append(attendees, database.Attendee{
				Email:       email,
				DisplayName: strings.TrimSpace(attendee.DisplayName),
				Optional:    attendee.Optional,
			})
		}

		if len(attendees) > 0 {
			if attendeeErr := w.db.SetEventAttendees(importedEvent.ID, attendees); attendeeErr != nil {
				fmt.Printf("Google Calendar worker: failed to set attendees for imported event %d: %v\n", importedEvent.ID, attendeeErr)
			}
		}

		linkedGoogleIDs[remoteEvent.ID] = struct{}{}
	}
}

func shouldUpdateLocalEvent(local database.CalendarEvent, remote *EventDetails) bool {
	if remote == nil {
		return false
	}
	if local.Title != remote.Summary {
		return true
	}
	if local.Description != remote.Description {
		return true
	}
	if local.Location != remote.Location {
		return true
	}
	if !local.StartTime.Equal(remote.StartTime) {
		return true
	}
	if local.EndTime == nil && remote.EndTime == nil {
		return false
	}
	if local.EndTime == nil || remote.EndTime == nil {
		return true
	}
	return !local.EndTime.Equal(*remote.EndTime)
}

func attendeesChanged(local, remote []database.Attendee) bool {
	if len(local) != len(remote) {
		return true
	}

	localNormalized := make([]string, 0, len(local))
	for _, attendee := range local {
		localNormalized = append(localNormalized, normalizeAttendee(attendee.Email, attendee.DisplayName, attendee.Optional))
	}

	remoteNormalized := make([]string, 0, len(remote))
	for _, attendee := range remote {
		remoteNormalized = append(remoteNormalized, normalizeAttendee(attendee.Email, attendee.DisplayName, attendee.Optional))
	}

	slices.Sort(localNormalized)
	slices.Sort(remoteNormalized)

	for i := range localNormalized {
		if localNormalized[i] != remoteNormalized[i] {
			return true
		}
	}

	return false
}

func normalizeAttendee(email, displayName string, optional bool) string {
	return strings.ToLower(strings.TrimSpace(email)) + "|" +
		strings.ToLower(strings.TrimSpace(displayName)) + "|" +
		fmt.Sprintf("%t", optional)
}
