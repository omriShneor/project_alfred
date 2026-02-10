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

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	statusFilter := r.URL.Query().Get("status")
	channelIDStr := r.URL.Query().Get("channel_id")

	var status *database.EventStatus
	if statusFilter != "" {
		s := database.EventStatus(statusFilter)
		status = &s
	}

	var channelID *int64
	if channelIDStr != "" {
		id, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err == nil {
			channelID = &id
		}
	}

	events, err := s.db.ListEvents(userID, status, channelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, events)
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
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

	event, err := s.db.GetEventByID(id)
	if err != nil || event.UserID != userID {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	// Include message context if available
	response := map[string]interface{}{
		"event": event,
	}

	if event.OriginalMsgID != nil {
		msg, err := s.db.GetMessageByID(*event.OriginalMsgID)
		if err == nil {
			response["trigger_message"] = msg
		}
	}

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleConfirmEvent(w http.ResponseWriter, r *http.Request) {
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

	event, err := s.db.GetEventByID(id)
	if err != nil || event.UserID != userID {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	if event.Status != database.EventStatusPending {
		respondError(w, http.StatusBadRequest, "event is not pending")
		return
	}

	// Check if sync is enabled and Google Calendar is connected
	gcalSettings, _ := s.db.GetGCalSettings(userID)
	userGCalClient := s.getGCalClientForUser(userID)
	shouldSync := gcalSettings != nil && gcalSettings.SyncEnabled && userGCalClient != nil && userGCalClient.IsAuthenticated()

	// If not syncing to Google Calendar, just confirm the event locally
	if !shouldSync {
		var newStatus database.EventStatus
		if event.ActionType == database.EventActionDelete {
			newStatus = database.EventStatusDeleted
		} else {
			newStatus = database.EventStatusConfirmed
		}

		if err := s.db.UpdateEventStatus(id, newStatus); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm event: %v", err))
			return
		}

		updatedEvent, _ := s.db.GetEventByID(id)
		respondJSON(w, http.StatusOK, updatedEvent)
		return
	}

	// Sync to Google Calendar
	var googleEventID string
	switch event.ActionType {
	case database.EventActionCreate:
		// Create event in Google Calendar
		endTime := event.StartTime.Add(1 * time.Hour) // 1 hour default
		if event.EndTime != nil {
			endTime = *event.EndTime
		}

		// Extract attendee emails
		attendeeEmails := make([]string, len(event.Attendees))
		for i, a := range event.Attendees {
			attendeeEmails[i] = a.Email
		}

		googleEventID, err = userGCalClient.CreateEvent(event.CalendarID, gcal.EventInput{
			Summary:     event.Title,
			Description: event.Description,
			Location:    event.Location,
			StartTime:   event.StartTime,
			EndTime:     endTime,
			Attendees:   attendeeEmails,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create calendar event: %v", err))
			return
		}

		// Update database with Google event ID
		if err := s.db.UpdateEventGoogleID(id, googleEventID); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update event: %v", err))
			return
		}

	case database.EventActionUpdate:
		// If no Google event ID, just confirm locally (event was created before sync was enabled)
		if event.GoogleEventID == nil {
			if err := s.db.UpdateEventStatus(id, database.EventStatusConfirmed); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to confirm event: %v", err))
				return
			}
			break
		}

		endTime := event.StartTime.Add(1 * time.Hour)
		if event.EndTime != nil {
			endTime = *event.EndTime
		}

		// Extract attendee emails
		updateAttendeeEmails := make([]string, len(event.Attendees))
		for i, a := range event.Attendees {
			updateAttendeeEmails[i] = a.Email
		}

		err = userGCalClient.UpdateEvent(event.CalendarID, *event.GoogleEventID, gcal.EventInput{
			Summary:     event.Title,
			Description: event.Description,
			Location:    event.Location,
			StartTime:   event.StartTime,
			EndTime:     endTime,
			Attendees:   updateAttendeeEmails,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update calendar event: %v", err))
			return
		}

		if err := s.db.UpdateEventStatus(id, database.EventStatusSynced); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update event status: %v", err))
			return
		}

	case database.EventActionDelete:
		// If no Google event ID, just mark as deleted locally
		if event.GoogleEventID == nil {
			if err := s.db.UpdateEventStatus(id, database.EventStatusDeleted); err != nil {
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete event: %v", err))
				return
			}
			break
		}

		err = userGCalClient.DeleteEvent(event.CalendarID, *event.GoogleEventID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete calendar event: %v", err))
			return
		}

		if err := s.db.UpdateEventStatus(id, database.EventStatusDeleted); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update event status: %v", err))
			return
		}
	}

	// Get updated event
	updatedEvent, _ := s.db.GetEventByID(id)
	respondJSON(w, http.StatusOK, updatedEvent)
}

func (s *Server) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
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

	event, err := s.db.GetEventByID(id)
	if err != nil || event.UserID != userID {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	if event.Status != database.EventStatusPending {
		respondError(w, http.StatusBadRequest, "can only edit pending events")
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		StartTime   string  `json:"start_time"`
		EndTime     *string `json:"end_time"`
		Location    string  `json:"location"`
		Attendees   []struct {
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
		} `json:"attendees"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	if req.StartTime == "" {
		respondError(w, http.StatusBadRequest, "start_time is required")
		return
	}

	timezone := s.getUserTimezone(userID)

	// Parse start time
	startTime, _, err := parseEventTime(req.StartTime, timezone)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid start_time format: %v", err))
		return
	}

	// Parse end time if provided
	var endTime *time.Time
	if req.EndTime != nil && *req.EndTime != "" {
		et, _, err := parseEventTime(*req.EndTime, timezone)
		if err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid end_time format: %v", err))
			return
		}
		endTime = &et
	}

	if err := s.db.UpdatePendingEvent(id, req.Title, req.Description, startTime, endTime, req.Location); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update attendees
	attendees := make([]database.Attendee, len(req.Attendees))
	for i, a := range req.Attendees {
		attendees[i] = database.Attendee{
			Email:       a.Email,
			DisplayName: a.DisplayName,
		}
	}
	if err := s.db.SetEventAttendees(id, attendees); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update attendees: %v", err))
		return
	}

	updatedEvent, _ := s.db.GetEventByID(id)
	respondJSON(w, http.StatusOK, updatedEvent)
}

func (s *Server) handleRejectEvent(w http.ResponseWriter, r *http.Request) {
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

	event, err := s.db.GetEventByID(id)
	if err != nil || event.UserID != userID {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	if event.Status != database.EventStatusPending {
		respondError(w, http.StatusBadRequest, "event is not pending")
		return
	}

	if err := s.db.UpdateEventStatus(id, database.EventStatusRejected); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (s *Server) handleGetChannelHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	channelID, err := strconv.ParseInt(r.PathValue("channelId"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	channel, err := s.db.GetSourceChannelByID(userID, channelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if channel == nil {
		respondError(w, http.StatusNotFound, "channel not found")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 25
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := s.db.GetMessageHistory(channelID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, messages)
}
