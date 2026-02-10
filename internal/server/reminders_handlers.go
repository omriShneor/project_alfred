package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
	"github.com/omriShneor/project_alfred/internal/timeutil"
)

// handleListReminders returns reminders with optional status and channel_id filters
func (s *Server) handleListReminders(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	statusFilter := r.URL.Query().Get("status")
	channelIDStr := r.URL.Query().Get("channel_id")

	var status *database.ReminderStatus
	if statusFilter != "" {
		st := database.ReminderStatus(statusFilter)
		status = &st
	}

	var channelID *int64
	if channelIDStr != "" {
		id, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err == nil {
			channelID = &id
		}
	}

	reminders, err := s.db.ListReminders(userID, status, channelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, reminders)
}

// handleCreateReminder creates a manual reminder/todo for the authenticated user.
func (s *Server) handleCreateReminder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Title        string  `json:"title"`
		Description  string  `json:"description"`
		Location     string  `json:"location"`
		DueDate      *string `json:"due_date"`
		ReminderTime *string `json:"reminder_time"`
		Priority     string  `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	var dueDate *time.Time
	timezone := s.getUserTimezone(userID)
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		parsed, err := parseReminderDateTime(strings.TrimSpace(*req.DueDate), timezone)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid due_date format")
			return
		}
		dueDate = &parsed
	}

	var reminderTime *time.Time
	if req.ReminderTime != nil && strings.TrimSpace(*req.ReminderTime) != "" {
		parsed, err := parseReminderDateTime(strings.TrimSpace(*req.ReminderTime), timezone)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid reminder_time format")
			return
		}
		reminderTime = &parsed
	}

	priority, err := parseReminderPriority(req.Priority)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	manualChannel, err := s.db.EnsureManualReminderChannel(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to setup manual reminders channel: %v", err))
		return
	}

	calendarID, err := s.db.GetSelectedCalendarID(userID)
	if err != nil || calendarID == "" {
		calendarID = "primary"
	}

	reminder := &database.Reminder{
		UserID:       userID,
		ChannelID:    manualChannel.ID,
		CalendarID:   calendarID,
		Title:        title,
		Description:  strings.TrimSpace(req.Description),
		Location:     strings.TrimSpace(req.Location),
		DueDate:      dueDate,
		ReminderTime: reminderTime,
		Priority:     priority,
		ActionType:   database.ReminderActionCreate,
		LLMReasoning: "manual reminder created by user",
		Source:       "manual",
	}

	created, err := s.db.CreatePendingReminder(reminder)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create reminder: %v", err))
		return
	}

	// Keep newly created reminders pending so all newly created items follow the
	// same review flow regardless of creation path.
	respondJSON(w, http.StatusCreated, created)
}

// handleGetReminder returns a single reminder by ID
func (s *Server) handleGetReminder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil || reminder.UserID != userID {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	// Include message context if available
	response := map[string]any{
		"reminder": reminder,
	}

	if reminder.OriginalMsgID != nil {
		msg, err := s.db.GetMessageByID(*reminder.OriginalMsgID)
		if err == nil {
			response["trigger_message"] = msg
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// handleUpdateReminder updates a pending reminder's details
func (s *Server) handleUpdateReminder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil || reminder.UserID != userID {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	if reminder.Status != database.ReminderStatusPending {
		respondError(w, http.StatusBadRequest, "can only update pending reminders")
		return
	}

	var req struct {
		Title        *string `json:"title"`
		Description  *string `json:"description"`
		Location     *string `json:"location"`
		DueDate      *string `json:"due_date"`
		ReminderTime *string `json:"reminder_time"`
		Priority     *string `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	title := reminder.Title
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
		if title == "" {
			respondError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
	}

	description := reminder.Description
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}

	location := reminder.Location
	if req.Location != nil {
		location = strings.TrimSpace(*req.Location)
	}

	timezone := s.getUserTimezone(userID)

	dueDate := reminder.DueDate
	if req.DueDate != nil {
		dueDateText := strings.TrimSpace(*req.DueDate)
		if dueDateText == "" {
			dueDate = nil
		} else {
			parsed, err := parseReminderDateTime(dueDateText, timezone)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid due_date format")
				return
			}
			dueDate = &parsed
		}
	}

	reminderTime := reminder.ReminderTime
	if req.ReminderTime != nil {
		reminderTimeText := strings.TrimSpace(*req.ReminderTime)
		if reminderTimeText == "" {
			reminderTime = nil
		} else {
			parsed, err := parseReminderDateTime(reminderTimeText, timezone)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid reminder_time format")
				return
			}
			reminderTime = &parsed
		}
	}

	priority := reminder.Priority
	if req.Priority != nil && strings.TrimSpace(*req.Priority) != "" {
		parsedPriority, err := parseReminderPriority(*req.Priority)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		priority = parsedPriority
	}

	if err := s.db.UpdatePendingReminder(id, title, description, location, dueDate, reminderTime, priority); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update reminder: %v", err))
		return
	}

	updated, _ := s.db.GetReminderByID(id)
	respondJSON(w, http.StatusOK, updated)
}

// handleConfirmReminder confirms a pending reminder, optionally syncing to Google Calendar
func (s *Server) handleConfirmReminder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil || reminder.UserID != userID {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	if reminder.Status != database.ReminderStatusPending {
		respondError(w, http.StatusBadRequest, "reminder is not pending")
		return
	}

	// Check if sync is enabled and Google Calendar is connected
	gcalSettings, _ := s.db.GetGCalSettings(userID)
	userGCalClient := s.getGCalClientForUser(userID)
	shouldSync := gcalSettings != nil && gcalSettings.SyncEnabled && userGCalClient != nil && userGCalClient.IsAuthenticated()

	// If not syncing to Google Calendar, just confirm the reminder locally
	if !shouldSync {
		var newStatus database.ReminderStatus
		if reminder.ActionType == database.ReminderActionDelete {
			newStatus = database.ReminderStatusDismissed
		} else {
			newStatus = database.ReminderStatusConfirmed
		}

		if err := s.db.UpdateReminderStatus(id, newStatus); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm reminder: %v", err))
			return
		}

		updatedReminder, _ := s.db.GetReminderByID(id)
		respondJSON(w, http.StatusOK, updatedReminder)
		return
	}

	// Sync to Google Calendar as a reminder event
	switch reminder.ActionType {
	case database.ReminderActionCreate:
		if reminder.DueDate == nil {
			// Cannot sync without a timestamp; keep reminder locally.
			if err := s.db.UpdateReminderStatus(id, database.ReminderStatusConfirmed); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm reminder: %v", err))
				return
			}
			break
		}

		endTime := reminder.DueDate.Add(30 * time.Minute)
		googleEventID, err := userGCalClient.CreateEvent(reminder.CalendarID, gcal.EventInput{
			Summary:     "[Reminder] " + reminder.Title,
			Description: reminder.Description,
			Location:    reminder.Location,
			StartTime:   *reminder.DueDate,
			EndTime:     endTime,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create calendar reminder: %v", err))
			return
		}

		// Update database with Google event ID
		if err := s.db.UpdateReminderGoogleID(id, googleEventID); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update reminder: %v", err))
			return
		}

	case database.ReminderActionUpdate:
		// If no Google event ID or due date, just confirm locally
		if reminder.GoogleEventID == nil || reminder.DueDate == nil {
			if err := s.db.UpdateReminderStatus(id, database.ReminderStatusConfirmed); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm reminder: %v", err))
				return
			}
			break
		}

		endTime := reminder.DueDate.Add(30 * time.Minute)
		err = userGCalClient.UpdateEvent(reminder.CalendarID, *reminder.GoogleEventID, gcal.EventInput{
			Summary:     "[Reminder] " + reminder.Title,
			Description: reminder.Description,
			Location:    reminder.Location,
			StartTime:   *reminder.DueDate,
			EndTime:     endTime,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update calendar reminder: %v", err))
			return
		}
		if err := s.db.UpdateReminderStatus(id, database.ReminderStatusSynced); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update reminder status: %v", err))
			return
		}

	case database.ReminderActionDelete:
		// If we have a Google event ID, delete from calendar
		if reminder.GoogleEventID != nil {
			if err := userGCalClient.DeleteEvent(reminder.CalendarID, *reminder.GoogleEventID); err != nil {
				fmt.Printf("Warning: failed to delete calendar reminder: %v\n", err)
			}
		}
		if err := s.db.UpdateReminderStatus(id, database.ReminderStatusDismissed); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to dismiss reminder: %v", err))
			return
		}
	}

	updatedReminder, _ := s.db.GetReminderByID(id)
	respondJSON(w, http.StatusOK, updatedReminder)
}

// handleRejectReminder rejects a pending reminder
func (s *Server) handleRejectReminder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil || reminder.UserID != userID {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	if reminder.Status != database.ReminderStatusPending {
		respondError(w, http.StatusBadRequest, "reminder is not pending")
		return
	}

	if err := s.db.UpdateReminderStatus(id, database.ReminderStatusRejected); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to reject reminder: %v", err))
		return
	}

	updatedReminder, _ := s.db.GetReminderByID(id)
	respondJSON(w, http.StatusOK, updatedReminder)
}

// handleCompleteReminder marks a confirmed/synced reminder as completed
func (s *Server) handleCompleteReminder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil || reminder.UserID != userID {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	// Can complete confirmed or synced reminders
	if reminder.Status != database.ReminderStatusConfirmed && reminder.Status != database.ReminderStatusSynced {
		respondError(w, http.StatusBadRequest, "can only complete confirmed or synced reminders")
		return
	}

	if err := s.db.UpdateReminderStatus(id, database.ReminderStatusCompleted); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to complete reminder: %v", err))
		return
	}

	updatedReminder, _ := s.db.GetReminderByID(id)
	respondJSON(w, http.StatusOK, updatedReminder)
}

// handleDismissReminder dismisses a reminder (user no longer wants to be reminded)
func (s *Server) handleDismissReminder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil || reminder.UserID != userID {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	// Can dismiss pending, confirmed, or synced reminders
	if reminder.Status == database.ReminderStatusCompleted || reminder.Status == database.ReminderStatusRejected || reminder.Status == database.ReminderStatusDismissed {
		respondError(w, http.StatusBadRequest, "reminder is already in a final state")
		return
	}

	// If synced to Google Calendar, delete the event
	userGCalClient := s.getGCalClientForUser(userID)
	if reminder.GoogleEventID != nil && userGCalClient != nil && userGCalClient.IsAuthenticated() {
		if err := userGCalClient.DeleteEvent(reminder.CalendarID, *reminder.GoogleEventID); err != nil {
			fmt.Printf("Warning: failed to delete calendar reminder: %v\n", err)
		}
	}

	if err := s.db.UpdateReminderStatus(id, database.ReminderStatusDismissed); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to dismiss reminder: %v", err))
		return
	}

	updatedReminder, _ := s.db.GetReminderByID(id)
	respondJSON(w, http.StatusOK, updatedReminder)
}

func parseReminderDateTime(s, timezone string) (time.Time, error) {
	if t, _, err := parseEventTime(s, timezone); err == nil {
		return t, nil
	}

	// Accept date-only values and default them to 09:00 in user timezone.
	if t, _, err := timeutil.ParseDateWithDefaultTime(s, timezone, 9, 0); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unrecognized datetime format")
}

func parseReminderPriority(raw string) (database.ReminderPriority, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return database.ReminderPriorityNormal, nil
	}

	switch database.ReminderPriority(normalized) {
	case database.ReminderPriorityLow, database.ReminderPriorityNormal, database.ReminderPriorityHigh:
		return database.ReminderPriority(normalized), nil
	default:
		return "", fmt.Errorf("invalid priority: must be one of low, normal, high")
	}
}
