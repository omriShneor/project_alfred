package testutil

import (
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
)

// ChannelBuilder builds test channels
type ChannelBuilder struct {
	sourceType  source.SourceType
	channelType source.ChannelType
	identifier  string
	name        string
	enabled     bool
}

// NewChannelBuilder creates a new channel builder with defaults
func NewChannelBuilder() *ChannelBuilder {
	return &ChannelBuilder{
		sourceType:  source.SourceTypeWhatsApp,
		channelType: source.ChannelTypeSender,
		identifier:  "test@s.whatsapp.net",
		name:        "Test Contact",
		enabled:     true,
	}
}

// WithSourceType sets the source type
func (b *ChannelBuilder) WithSourceType(st source.SourceType) *ChannelBuilder {
	b.sourceType = st
	return b
}

// WhatsApp sets source type to WhatsApp
func (b *ChannelBuilder) WhatsApp() *ChannelBuilder {
	b.sourceType = source.SourceTypeWhatsApp
	b.channelType = source.ChannelTypeSender
	b.identifier = "test@s.whatsapp.net"
	return b
}

// Telegram sets source type to Telegram
func (b *ChannelBuilder) Telegram() *ChannelBuilder {
	b.sourceType = source.SourceTypeTelegram
	b.channelType = source.ChannelTypeSender
	b.identifier = "tg_user_123"
	return b
}

// Gmail sets source type to Gmail
func (b *ChannelBuilder) Gmail() *ChannelBuilder {
	b.sourceType = source.SourceTypeGmail
	b.channelType = source.ChannelTypeSender
	b.identifier = "sender@example.com"
	return b
}

// GmailDomain sets source type to Gmail domain
func (b *ChannelBuilder) GmailDomain() *ChannelBuilder {
	b.sourceType = source.SourceTypeGmail
	b.channelType = source.ChannelTypeDomain
	b.identifier = "example.com"
	return b
}

// GmailCategory sets source type to Gmail category
func (b *ChannelBuilder) GmailCategory() *ChannelBuilder {
	b.sourceType = source.SourceTypeGmail
	b.channelType = source.ChannelTypeCategory
	b.identifier = "CATEGORY_PRIMARY"
	return b
}

// WithChannelType sets the channel type
func (b *ChannelBuilder) WithChannelType(ct source.ChannelType) *ChannelBuilder {
	b.channelType = ct
	return b
}

// WithIdentifier sets the identifier
func (b *ChannelBuilder) WithIdentifier(id string) *ChannelBuilder {
	b.identifier = id
	return b
}

// WithName sets the name
func (b *ChannelBuilder) WithName(name string) *ChannelBuilder {
	b.name = name
	return b
}

// Enabled sets the channel as enabled
func (b *ChannelBuilder) Enabled() *ChannelBuilder {
	b.enabled = true
	return b
}

// Disabled sets the channel as disabled
func (b *ChannelBuilder) Disabled() *ChannelBuilder {
	b.enabled = false
	return b
}

// Build creates the channel in the database
func (b *ChannelBuilder) Build(db *database.DB) (*database.SourceChannel, error) {
	channel, err := db.CreateSourceChannel(b.sourceType, b.channelType, b.identifier, b.name)
	if err != nil {
		return nil, err
	}

	if !b.enabled {
		if err := db.UpdateSourceChannel(channel.ID, channel.Name, false); err != nil {
			return nil, err
		}
		channel.Enabled = false
	}

	return channel, nil
}

// MustBuild creates the channel or panics
func (b *ChannelBuilder) MustBuild(db *database.DB) *database.SourceChannel {
	channel, err := b.Build(db)
	if err != nil {
		panic(fmt.Sprintf("failed to build channel: %v", err))
	}
	return channel
}

// EventBuilder builds test events
type EventBuilder struct {
	channelID   int64
	calendarID  string
	title       string
	description string
	startTime   time.Time
	endTime     *time.Time
	location    string
	status      database.EventStatus
	actionType  database.EventActionType
	reasoning   string
}

// NewEventBuilder creates a new event builder with defaults
func NewEventBuilder(channelID int64) *EventBuilder {
	now := time.Now().Truncate(time.Second)
	endTime := now.Add(time.Hour)
	return &EventBuilder{
		channelID:  channelID,
		calendarID: "primary",
		title:      "Test Event",
		startTime:  now,
		endTime:    &endTime,
		status:     database.EventStatusPending,
		actionType: database.EventActionCreate,
		reasoning:  "Test event created by builder",
	}
}

// WithTitle sets the title
func (b *EventBuilder) WithTitle(title string) *EventBuilder {
	b.title = title
	return b
}

// WithDescription sets the description
func (b *EventBuilder) WithDescription(desc string) *EventBuilder {
	b.description = desc
	return b
}

// WithLocation sets the location
func (b *EventBuilder) WithLocation(loc string) *EventBuilder {
	b.location = loc
	return b
}

// WithStartTime sets the start time
func (b *EventBuilder) WithStartTime(t time.Time) *EventBuilder {
	b.startTime = t
	return b
}

// WithEndTime sets the end time
func (b *EventBuilder) WithEndTime(t time.Time) *EventBuilder {
	b.endTime = &t
	return b
}

// WithCalendarID sets the calendar ID
func (b *EventBuilder) WithCalendarID(id string) *EventBuilder {
	b.calendarID = id
	return b
}

// WithReasoning sets the LLM reasoning
func (b *EventBuilder) WithReasoning(reasoning string) *EventBuilder {
	b.reasoning = reasoning
	return b
}

// Pending sets status to pending
func (b *EventBuilder) Pending() *EventBuilder {
	b.status = database.EventStatusPending
	return b
}

// Confirmed sets status to confirmed
func (b *EventBuilder) Confirmed() *EventBuilder {
	b.status = database.EventStatusConfirmed
	return b
}

// Synced sets status to synced
func (b *EventBuilder) Synced() *EventBuilder {
	b.status = database.EventStatusSynced
	return b
}

// Rejected sets status to rejected
func (b *EventBuilder) Rejected() *EventBuilder {
	b.status = database.EventStatusRejected
	return b
}

// CreateAction sets action type to create
func (b *EventBuilder) CreateAction() *EventBuilder {
	b.actionType = database.EventActionCreate
	return b
}

// UpdateAction sets action type to update
func (b *EventBuilder) UpdateAction() *EventBuilder {
	b.actionType = database.EventActionUpdate
	return b
}

// DeleteAction sets action type to delete
func (b *EventBuilder) DeleteAction() *EventBuilder {
	b.actionType = database.EventActionDelete
	return b
}

// Build creates the event in the database
func (b *EventBuilder) Build(db *database.DB) (*database.CalendarEvent, error) {
	event := &database.CalendarEvent{
		ChannelID:    b.channelID,
		CalendarID:   b.calendarID,
		Title:        b.title,
		Description:  b.description,
		StartTime:    b.startTime,
		EndTime:      b.endTime,
		Location:     b.location,
		Status:       b.status,
		ActionType:   b.actionType,
		LLMReasoning: b.reasoning,
	}

	created, err := db.CreatePendingEvent(event)
	if err != nil {
		return nil, err
	}

	// If status is not pending, update it
	if b.status != database.EventStatusPending {
		if err := db.UpdateEventStatus(created.ID, b.status); err != nil {
			return nil, err
		}
		created.Status = b.status
	}

	return created, nil
}

// MustBuild creates the event or panics
func (b *EventBuilder) MustBuild(db *database.DB) *database.CalendarEvent {
	event, err := b.Build(db)
	if err != nil {
		panic(fmt.Sprintf("failed to build event: %v", err))
	}
	return event
}

// MessageBuilder builds test messages
type MessageBuilder struct {
	sourceType source.SourceType
	channelID  int64
	senderID   string
	senderName string
	text       string
	subject    string
	timestamp  time.Time
}

// NewMessageBuilder creates a new message builder with defaults
func NewMessageBuilder(channelID int64) *MessageBuilder {
	return &MessageBuilder{
		sourceType: source.SourceTypeWhatsApp,
		channelID:  channelID,
		senderID:   "sender@s.whatsapp.net",
		senderName: "Test Sender",
		text:       "Let's meet tomorrow at 2pm for lunch",
		timestamp:  time.Now().Truncate(time.Second),
	}
}

// WithSourceType sets the source type
func (b *MessageBuilder) WithSourceType(st source.SourceType) *MessageBuilder {
	b.sourceType = st
	return b
}

// WhatsApp sets source type to WhatsApp
func (b *MessageBuilder) WhatsApp() *MessageBuilder {
	b.sourceType = source.SourceTypeWhatsApp
	return b
}

// Telegram sets source type to Telegram
func (b *MessageBuilder) Telegram() *MessageBuilder {
	b.sourceType = source.SourceTypeTelegram
	return b
}

// Gmail sets source type to Gmail
func (b *MessageBuilder) Gmail() *MessageBuilder {
	b.sourceType = source.SourceTypeGmail
	return b
}

// WithSenderID sets the sender ID
func (b *MessageBuilder) WithSenderID(id string) *MessageBuilder {
	b.senderID = id
	return b
}

// WithSenderName sets the sender name
func (b *MessageBuilder) WithSenderName(name string) *MessageBuilder {
	b.senderName = name
	return b
}

// WithText sets the message text
func (b *MessageBuilder) WithText(text string) *MessageBuilder {
	b.text = text
	return b
}

// WithSubject sets the email subject (for Gmail)
func (b *MessageBuilder) WithSubject(subject string) *MessageBuilder {
	b.subject = subject
	return b
}

// WithTimestamp sets the timestamp
func (b *MessageBuilder) WithTimestamp(t time.Time) *MessageBuilder {
	b.timestamp = t
	return b
}

// Build stores the message in the database
func (b *MessageBuilder) Build(db *database.DB) (*database.SourceMessage, error) {
	return db.StoreSourceMessage(
		b.sourceType,
		b.channelID,
		b.senderID,
		b.senderName,
		b.text,
		b.subject,
		b.timestamp,
	)
}

// MustBuild stores the message or panics
func (b *MessageBuilder) MustBuild(db *database.DB) *database.SourceMessage {
	msg, err := b.Build(db)
	if err != nil {
		panic(fmt.Sprintf("failed to build message: %v", err))
	}
	return msg
}

// TestEventMessages contains sample messages for testing event detection
var TestEventMessages = []string{
	"Let's meet tomorrow at 2pm for lunch at the Italian restaurant",
	"Can we schedule a call for next Monday at 10am?",
	"Don't forget the team meeting on Friday at 3:30pm",
	"Doctor appointment on March 15th at 9:00 AM",
	"Birthday party at John's house this Saturday at 7pm",
	"Project deadline is next Wednesday, let's sync at 4pm",
	"Coffee catch up on Thursday morning, 8:30am works for me",
	"Interview scheduled for tomorrow at 2:30 PM in conference room B",
}

// TestNonEventMessages contains sample messages that should NOT trigger events
var TestNonEventMessages = []string{
	"How are you doing?",
	"Thanks for the help yesterday!",
	"Did you see the game last night?",
	"I'll send you the files soon",
	"Happy birthday!",
	"Great work on the presentation",
}

// ReminderBuilder builds test reminders
type ReminderBuilder struct {
	channelID    int64
	calendarID   string
	title        string
	description  string
	dueDate      time.Time
	reminderTime *time.Time
	priority     database.ReminderPriority
	status       database.ReminderStatus
	actionType   database.ReminderActionType
	reasoning    string
	source       string
}

// NewReminderBuilder creates a new reminder builder with defaults
func NewReminderBuilder(channelID int64) *ReminderBuilder {
	dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	return &ReminderBuilder{
		channelID:  channelID,
		calendarID: "primary",
		title:      "Test Reminder",
		dueDate:    dueDate,
		priority:   database.ReminderPriorityNormal,
		status:     database.ReminderStatusPending,
		actionType: database.ReminderActionCreate,
		reasoning:  "Test reminder created by builder",
		source:     "whatsapp",
	}
}

// WithTitle sets the title
func (b *ReminderBuilder) WithTitle(title string) *ReminderBuilder {
	b.title = title
	return b
}

// WithDescription sets the description
func (b *ReminderBuilder) WithDescription(desc string) *ReminderBuilder {
	b.description = desc
	return b
}

// WithDueDate sets the due date
func (b *ReminderBuilder) WithDueDate(t time.Time) *ReminderBuilder {
	b.dueDate = t
	return b
}

// WithReminderTime sets the reminder time
func (b *ReminderBuilder) WithReminderTime(t time.Time) *ReminderBuilder {
	b.reminderTime = &t
	return b
}

// WithCalendarID sets the calendar ID
func (b *ReminderBuilder) WithCalendarID(id string) *ReminderBuilder {
	b.calendarID = id
	return b
}

// WithReasoning sets the LLM reasoning
func (b *ReminderBuilder) WithReasoning(reasoning string) *ReminderBuilder {
	b.reasoning = reasoning
	return b
}

// WithSource sets the source
func (b *ReminderBuilder) WithSource(source string) *ReminderBuilder {
	b.source = source
	return b
}

// LowPriority sets priority to low
func (b *ReminderBuilder) LowPriority() *ReminderBuilder {
	b.priority = database.ReminderPriorityLow
	return b
}

// NormalPriority sets priority to normal
func (b *ReminderBuilder) NormalPriority() *ReminderBuilder {
	b.priority = database.ReminderPriorityNormal
	return b
}

// HighPriority sets priority to high
func (b *ReminderBuilder) HighPriority() *ReminderBuilder {
	b.priority = database.ReminderPriorityHigh
	return b
}

// Pending sets status to pending
func (b *ReminderBuilder) Pending() *ReminderBuilder {
	b.status = database.ReminderStatusPending
	return b
}

// Confirmed sets status to confirmed
func (b *ReminderBuilder) Confirmed() *ReminderBuilder {
	b.status = database.ReminderStatusConfirmed
	return b
}

// Synced sets status to synced
func (b *ReminderBuilder) Synced() *ReminderBuilder {
	b.status = database.ReminderStatusSynced
	return b
}

// Rejected sets status to rejected
func (b *ReminderBuilder) Rejected() *ReminderBuilder {
	b.status = database.ReminderStatusRejected
	return b
}

// Completed sets status to completed
func (b *ReminderBuilder) Completed() *ReminderBuilder {
	b.status = database.ReminderStatusCompleted
	return b
}

// Dismissed sets status to dismissed
func (b *ReminderBuilder) Dismissed() *ReminderBuilder {
	b.status = database.ReminderStatusDismissed
	return b
}

// CreateAction sets action type to create
func (b *ReminderBuilder) CreateAction() *ReminderBuilder {
	b.actionType = database.ReminderActionCreate
	return b
}

// UpdateAction sets action type to update
func (b *ReminderBuilder) UpdateAction() *ReminderBuilder {
	b.actionType = database.ReminderActionUpdate
	return b
}

// DeleteAction sets action type to delete
func (b *ReminderBuilder) DeleteAction() *ReminderBuilder {
	b.actionType = database.ReminderActionDelete
	return b
}

// Build creates the reminder in the database
func (b *ReminderBuilder) Build(db *database.DB) (*database.Reminder, error) {
	reminder := &database.Reminder{
		ChannelID:    b.channelID,
		CalendarID:   b.calendarID,
		Title:        b.title,
		Description:  b.description,
		DueDate:      b.dueDate,
		ReminderTime: b.reminderTime,
		Priority:     b.priority,
		Status:       b.status,
		ActionType:   b.actionType,
		LLMReasoning: b.reasoning,
		Source:       b.source,
	}

	created, err := db.CreatePendingReminder(reminder)
	if err != nil {
		return nil, err
	}

	// If status is not pending, update it
	if b.status != database.ReminderStatusPending {
		if err := db.UpdateReminderStatus(created.ID, b.status); err != nil {
			return nil, err
		}
		created.Status = b.status
	}

	return created, nil
}

// MustBuild creates the reminder or panics
func (b *ReminderBuilder) MustBuild(db *database.DB) *database.Reminder {
	reminder, err := b.Build(db)
	if err != nil {
		panic(fmt.Sprintf("failed to build reminder: %v", err))
	}
	return reminder
}

// TestReminderMessages contains sample messages for testing reminder detection
var TestReminderMessages = []string{
	"Remind me to call mom tomorrow",
	"Don't forget to submit the report by Friday",
	"Remember to buy groceries after work",
	"I need to pay the electricity bill next week",
	"Remind me to book the flight for next month",
	"Don't forget the dentist appointment on the 15th",
}
