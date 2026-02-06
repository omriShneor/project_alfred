package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

// ReminderCreationParams contains parameters for creating a reminder from analysis
type ReminderCreationParams struct {
	// User info
	UserID int64

	// Channel info
	ChannelID  int64
	CalendarID string // If empty, will be looked up from settings

	// Source tracking
	SourceType    source.SourceType
	EmailSourceID *int64 // Only for gmail sources
	MessageID     *int64 // Reference to triggering message

	// From reminder analysis
	Analysis *agent.ReminderAnalysis
}

// ReminderCreator handles shared reminder creation logic
type ReminderCreator struct {
	db            *database.DB
	notifyService *notify.Service
}

// NewReminderCreator creates a new ReminderCreator
func NewReminderCreator(db *database.DB, notifyService *notify.Service) *ReminderCreator {
	return &ReminderCreator{
		db:            db,
		notifyService: notifyService,
	}
}

// CreateReminderFromAnalysis creates or updates a pending reminder from the analysis
func (rc *ReminderCreator) CreateReminderFromAnalysis(ctx context.Context, params ReminderCreationParams) (*database.Reminder, error) {
	if params.Analysis == nil {
		return nil, fmt.Errorf("analysis is nil")
	}

	// Handle different actions
	switch params.Analysis.Action {
	case "create":
		return rc.createReminder(ctx, params)
	case "update":
		return rc.updateReminder(ctx, params)
	case "delete":
		return rc.deleteReminder(ctx, params)
	case "none":
		return nil, nil // No action needed
	default:
		return nil, fmt.Errorf("unknown action type: %s", params.Analysis.Action)
	}
}

// createReminder creates a new pending reminder
func (rc *ReminderCreator) createReminder(_ context.Context, params ReminderCreationParams) (*database.Reminder, error) {
	if params.Analysis.Reminder == nil {
		return nil, fmt.Errorf("analysis has no reminder data")
	}

	reminderData := params.Analysis.Reminder

	// Parse due date
	dueDate, err := parseReminderTime(reminderData.DueDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse due date: %w", err)
	}

	// Parse reminder time (optional, defaults to due date)
	var reminderTime *time.Time
	if reminderData.ReminderTime != "" {
		rt, err := parseReminderTime(reminderData.ReminderTime)
		if err == nil {
			reminderTime = &rt
		}
	}

	// Map priority
	priority := mapReminderPriority(reminderData.Priority)

	// Get calendar ID if not provided
	calendarID := params.CalendarID
	if calendarID == "" {
		calendarID, _ = rc.db.GetSelectedCalendarID(params.UserID)
	}

	reminder := &database.Reminder{
		ChannelID:    params.ChannelID,
		CalendarID:   calendarID,
		Title:        reminderData.Title,
		Description:  reminderData.Description,
		DueDate:      dueDate,
		ReminderTime: reminderTime,
		Priority:     priority,
		ActionType:   database.ReminderActionCreate,
		OriginalMsgID: params.MessageID,
		LLMReasoning: params.Analysis.Reasoning,
		Source:       string(params.SourceType),
	}

	created, err := rc.db.CreatePendingReminder(reminder)
	if err != nil {
		return nil, fmt.Errorf("failed to save reminder: %w", err)
	}

	// Update email source ID if applicable
	if params.EmailSourceID != nil {
		_, _ = rc.db.Exec(`UPDATE reminders SET email_source_id = ? WHERE id = ?`,
			*params.EmailSourceID, created.ID)
	}

	fmt.Printf("Created pending reminder: %s (ID: %d, Due: %s, Priority: %s, Source: %s)\n",
		created.Title, created.ID, created.DueDate.Format("2006-01-02 15:04"), created.Priority, params.SourceType)

	// Send notification (non-blocking, don't fail reminder creation)
	if rc.notifyService != nil {
		go rc.notifyService.NotifyPendingReminder(context.Background(), created)
	}

	return created, nil
}

// updateReminder updates an existing pending reminder
func (rc *ReminderCreator) updateReminder(_ context.Context, params ReminderCreationParams) (*database.Reminder, error) {
	if params.Analysis.Reminder == nil {
		return nil, fmt.Errorf("analysis has no reminder data for update")
	}

	reminderData := params.Analysis.Reminder

	// Find the existing reminder by alfred_reminder_id
	if reminderData.AlfredReminderRef == 0 {
		return nil, fmt.Errorf("update action requires alfred_reminder_id")
	}

	existing, err := rc.db.GetReminderByID(reminderData.AlfredReminderRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing reminder %d: %w", reminderData.AlfredReminderRef, err)
	}

	// Only update pending reminders
	if existing.Status != database.ReminderStatusPending {
		return nil, fmt.Errorf("cannot update reminder with status %s", existing.Status)
	}

	// Build updates - only change fields that were provided
	title := existing.Title
	if reminderData.Title != "" {
		title = reminderData.Title
	}

	description := existing.Description
	if reminderData.Description != "" {
		description = reminderData.Description
	}

	dueDate := existing.DueDate
	if reminderData.DueDate != "" {
		parsed, err := parseReminderTime(reminderData.DueDate)
		if err == nil {
			dueDate = parsed
		}
	}

	var reminderTime *time.Time
	if reminderData.ReminderTime != "" {
		rt, err := parseReminderTime(reminderData.ReminderTime)
		if err == nil {
			reminderTime = &rt
		}
	} else {
		reminderTime = existing.ReminderTime
	}

	priority := existing.Priority
	if reminderData.Priority != "" {
		priority = mapReminderPriority(reminderData.Priority)
	}

	// Perform the update
	if err := rc.db.UpdatePendingReminder(
		existing.ID,
		title,
		description,
		dueDate,
		reminderTime,
		priority,
	); err != nil {
		return nil, fmt.Errorf("failed to update pending reminder: %w", err)
	}

	fmt.Printf("Updated pending reminder: %s (ID: %d)\n", title, existing.ID)

	// Return the updated reminder
	updated, _ := rc.db.GetReminderByID(existing.ID)
	if updated != nil {
		return updated, nil
	}
	return existing, nil
}

// deleteReminder cancels/rejects an existing pending reminder
func (rc *ReminderCreator) deleteReminder(_ context.Context, params ReminderCreationParams) (*database.Reminder, error) {
	if params.Analysis.Reminder == nil {
		return nil, fmt.Errorf("analysis has no reminder data for delete")
	}

	reminderData := params.Analysis.Reminder

	// Find the existing reminder by alfred_reminder_id
	if reminderData.AlfredReminderRef == 0 {
		return nil, fmt.Errorf("delete action requires alfred_reminder_id")
	}

	existing, err := rc.db.GetReminderByID(reminderData.AlfredReminderRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing reminder %d: %w", reminderData.AlfredReminderRef, err)
	}

	// Only delete/reject pending reminders
	if existing.Status != database.ReminderStatusPending {
		return nil, fmt.Errorf("cannot delete reminder with status %s", existing.Status)
	}

	// Mark as rejected
	if err := rc.db.UpdateReminderStatus(existing.ID, database.ReminderStatusRejected); err != nil {
		return nil, fmt.Errorf("failed to reject pending reminder: %w", err)
	}

	fmt.Printf("Rejected pending reminder: %s (ID: %d) - user cancelled\n",
		existing.Title, existing.ID)

	return existing, nil
}

// parseReminderTime parses a time string in various formats
func parseReminderTime(timeStr string) (time.Time, error) {
	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try ISO 8601 without timezone
	t, err = time.Parse("2006-01-02T15:04:05", timeStr)
	if err == nil {
		return t, nil
	}

	// Try date only
	t, err = time.Parse("2006-01-02", timeStr)
	if err == nil {
		// Default to 9 AM if only date provided
		return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// mapReminderPriority maps a string priority to ReminderPriority
func mapReminderPriority(priority string) database.ReminderPriority {
	switch priority {
	case "low":
		return database.ReminderPriorityLow
	case "high":
		return database.ReminderPriorityHigh
	default:
		return database.ReminderPriorityNormal
	}
}
