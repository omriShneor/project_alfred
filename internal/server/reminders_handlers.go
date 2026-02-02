package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gcal"
)

// handleListReminders returns reminders with optional status and channel_id filters
func (s *Server) handleListReminders(w http.ResponseWriter, r *http.Request) {
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

	reminders, err := s.db.ListReminders(status, channelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, reminders)
}

// handleGetReminder returns a single reminder by ID
func (s *Server) handleGetReminder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil {
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	if reminder.Status != database.ReminderStatusPending {
		respondError(w, http.StatusBadRequest, "can only update pending reminders")
		return
	}

	var req struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		DueDate      string `json:"due_date"`
		ReminderTime string `json:"reminder_time"`
		Priority     string `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Use existing values if not provided
	title := reminder.Title
	if req.Title != "" {
		title = req.Title
	}

	description := reminder.Description
	if req.Description != "" {
		description = req.Description
	}

	dueDate := reminder.DueDate
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			parsed, err = time.Parse("2006-01-02T15:04:05", req.DueDate)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid due_date format")
				return
			}
		}
		dueDate = parsed
	}

	var reminderTime *time.Time
	if req.ReminderTime != "" {
		parsed, err := time.Parse(time.RFC3339, req.ReminderTime)
		if err != nil {
			parsed, err = time.Parse("2006-01-02T15:04:05", req.ReminderTime)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid reminder_time format")
				return
			}
		}
		reminderTime = &parsed
	} else {
		reminderTime = reminder.ReminderTime
	}

	priority := reminder.Priority
	if req.Priority != "" {
		priority = database.ReminderPriority(req.Priority)
	}

	if err := s.db.UpdatePendingReminder(id, title, description, dueDate, reminderTime, priority); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update reminder: %v", err))
		return
	}

	updated, _ := s.db.GetReminderByID(id)
	respondJSON(w, http.StatusOK, updated)
}

// handleConfirmReminder confirms a pending reminder, optionally syncing to Google Calendar
func (s *Server) handleConfirmReminder(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	if reminder.Status != database.ReminderStatusPending {
		respondError(w, http.StatusBadRequest, "reminder is not pending")
		return
	}

	// Check if sync is enabled and Google Calendar is connected
	gcalSettings, _ := s.db.GetGCalSettings()
	shouldSync := gcalSettings != nil && gcalSettings.SyncEnabled && s.gcalClient != nil && s.gcalClient.IsAuthenticated()

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
		// Create reminder event in Google Calendar
		// Use a 30-minute event block at the due date
		endTime := reminder.DueDate.Add(30 * time.Minute)

		googleEventID, err := s.gcalClient.CreateEvent(reminder.CalendarID, gcal.EventInput{
			Summary:     "[Reminder] " + reminder.Title,
			Description: reminder.Description,
			StartTime:   reminder.DueDate,
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
		// If no Google event ID, just confirm locally
		if reminder.GoogleEventID == nil {
			if err := s.db.UpdateReminderStatus(id, database.ReminderStatusConfirmed); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm reminder: %v", err))
				return
			}
		} else {
			endTime := reminder.DueDate.Add(30 * time.Minute)
			err = s.gcalClient.UpdateEvent(reminder.CalendarID, *reminder.GoogleEventID, gcal.EventInput{
				Summary:     "[Reminder] " + reminder.Title,
				Description: reminder.Description,
				StartTime:   reminder.DueDate,
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
		}

	case database.ReminderActionDelete:
		// If we have a Google event ID, delete from calendar
		if reminder.GoogleEventID != nil {
			if err := s.gcalClient.DeleteEvent(reminder.CalendarID, *reminder.GoogleEventID); err != nil {
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil {
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil {
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	reminder, err := s.db.GetReminderByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "reminder not found")
		return
	}

	// Can dismiss pending, confirmed, or synced reminders
	if reminder.Status == database.ReminderStatusCompleted || reminder.Status == database.ReminderStatusRejected || reminder.Status == database.ReminderStatusDismissed {
		respondError(w, http.StatusBadRequest, "reminder is already in a final state")
		return
	}

	// If synced to Google Calendar, delete the event
	if reminder.GoogleEventID != nil && s.gcalClient != nil && s.gcalClient.IsAuthenticated() {
		if err := s.gcalClient.DeleteEvent(reminder.CalendarID, *reminder.GoogleEventID); err != nil {
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
