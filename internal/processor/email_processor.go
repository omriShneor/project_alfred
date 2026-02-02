package processor

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

// EmailProcessor processes emails for calendar event and reminder detection
type EmailProcessor struct {
	db               *database.DB
	analyzer         agent.Analyzer
	reminderAnalyzer agent.ReminderAnalyzer
	notifyService    *notify.Service
	eventCreator     *EventCreator
	reminderCreator  *ReminderCreator
}

// NewEmailProcessor creates a new email processor
func NewEmailProcessor(db *database.DB, analyzer agent.Analyzer, reminderAnalyzer agent.ReminderAnalyzer, notifyService *notify.Service) *EmailProcessor {
	return &EmailProcessor{
		db:               db,
		analyzer:         analyzer,
		reminderAnalyzer: reminderAnalyzer,
		notifyService:    notifyService,
		eventCreator:     NewEventCreator(db, notifyService),
		reminderCreator:  NewReminderCreator(db, notifyService),
	}
}

// ProcessEmail processes a single email for event and reminder detection
func (p *EmailProcessor) ProcessEmail(ctx context.Context, email *gmail.Email, emailSource *gmail.EmailSource, thread *gmail.Thread) error {
	threadLen := 0
	if thread != nil {
		threadLen = len(thread.Messages)
	}
	fmt.Printf("Processing email: %s (from: %s, thread: %d messages)\n", truncate(email.Subject, 50), email.From, threadLen)

	// Clean the email body
	body := gmail.CleanEmailBody(email.Body)

	// Build email content with thread context (shared between analyzers)
	emailContent := agent.EmailContent{
		Subject: email.Subject,
		From:    email.From,
		To:      email.To,
		Date:    email.Date,
		Body:    body,
	}

	// Add thread history (excluding the latest message which is the email being analyzed)
	if thread != nil && len(thread.Messages) > 1 {
		for _, msg := range thread.Messages[:len(thread.Messages)-1] {
			emailContent.ThreadHistory = append(emailContent.ThreadHistory, agent.EmailThreadMessage{
				From:    msg.From,
				Date:    msg.Date,
				Subject: msg.Subject,
				Body:    msg.Body,
			})
		}
	}

	// Run event analyzer (independent goroutine - fire and forget)
	if p.analyzer != nil && p.analyzer.IsConfigured() {
		go func() {
			analysis, err := p.analyzer.AnalyzeEmail(ctx, emailContent)
			if err != nil {
				fmt.Printf("Email event analysis error: %v\n", err)
				return
			}

			fmt.Printf("Email event analysis: action=%s, has_event=%v, confidence=%.2f\n",
				analysis.Action, analysis.HasEvent, analysis.Confidence)

			if analysis.HasEvent && analysis.Action != "none" {
				if err := p.createPendingEventFromEmail(emailSource, analysis); err != nil {
					fmt.Printf("Failed to create pending event from email: %v\n", err)
				}
			}
		}()
	}

	// Run reminder analyzer (independent goroutine - fire and forget)
	if p.reminderAnalyzer != nil && p.reminderAnalyzer.IsConfigured() {
		go func() {
			analysis, err := p.reminderAnalyzer.AnalyzeEmail(ctx, emailContent)
			if err != nil {
				fmt.Printf("Email reminder analysis error: %v\n", err)
				return
			}

			fmt.Printf("Email reminder analysis: action=%s, has_reminder=%v, confidence=%.2f\n",
				analysis.Action, analysis.HasReminder, analysis.Confidence)

			if analysis.HasReminder && analysis.Action != "none" {
				if err := p.createPendingReminderFromEmail(emailSource, analysis); err != nil {
					fmt.Printf("Failed to create pending reminder from email: %v\n", err)
				}
			}
		}()
	}

	return nil
}

// createPendingEventFromEmail creates a pending event from email analysis
func (p *EmailProcessor) createPendingEventFromEmail(emailSource *gmail.EmailSource, analysis *agent.EventAnalysis) error {
	// Get or create a placeholder channel for email sources
	emailChannel, err := p.getOrCreateEmailChannel(emailSource)
	if err != nil {
		return fmt.Errorf("failed to get email channel: %w", err)
	}

	emailSourceID := emailSource.ID
	params := EventCreationParams{
		ChannelID:     emailChannel.ID,
		SourceType:    source.SourceTypeGmail,
		EmailSourceID: &emailSourceID,
		Analysis:      analysis,
	}

	_, err = p.eventCreator.CreateEventFromAnalysis(context.Background(), params)
	return err
}

// createPendingReminderFromEmail creates a pending reminder from email analysis
func (p *EmailProcessor) createPendingReminderFromEmail(emailSource *gmail.EmailSource, analysis *agent.ReminderAnalysis) error {
	// Get or create a placeholder channel for email sources
	emailChannel, err := p.getOrCreateEmailChannel(emailSource)
	if err != nil {
		return fmt.Errorf("failed to get email channel: %w", err)
	}

	emailSourceID := emailSource.ID
	params := ReminderCreationParams{
		ChannelID:     emailChannel.ID,
		SourceType:    source.SourceTypeGmail,
		EmailSourceID: &emailSourceID,
		Analysis:      analysis,
	}

	_, err = p.reminderCreator.CreateReminderFromAnalysis(context.Background(), params)
	return err
}

// getOrCreateEmailChannel gets or creates a placeholder channel for email-sourced events
// This is needed because calendar_events has a foreign key to channels
func (p *EmailProcessor) getOrCreateEmailChannel(source *gmail.EmailSource) (*database.Channel, error) {
	// Use a channel identifier based on the email source
	identifier := fmt.Sprintf("email:%s:%s", source.Type, source.Identifier)

	// Check if channel exists
	channel, err := p.db.GetChannelByIdentifier(identifier)
	if err != nil {
		return nil, err
	}
	if channel != nil {
		return channel, nil
	}

	// Create new channel for this email source
	return p.db.CreateChannel(database.ChannelTypeSender, identifier, fmt.Sprintf("Email: %s", source.Name))
}
