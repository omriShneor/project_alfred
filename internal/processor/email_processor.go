package processor

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/agent/intents"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/gmail"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

// EmailProcessor processes emails for calendar event and reminder detection
type EmailProcessor struct {
	db               *database.DB
	eventAnalyzer    agent.EventAnalyzer
	reminderAnalyzer agent.ReminderAnalyzer
	notifyService    *notify.Service
	eventCreator     *EventCreator
	reminderCreator  *ReminderCreator
	intentRegistry   *intents.Registry
	intentRouter     intents.Router
}

// NewEmailProcessor creates a new email processor
func NewEmailProcessor(db *database.DB, eventAnalyzer agent.EventAnalyzer, reminderAnalyzer agent.ReminderAnalyzer, notifyService *notify.Service) *EmailProcessor {
	registry := intents.NewRegistry()
	if eventAnalyzer != nil && eventAnalyzer.IsConfigured() {
		_ = registry.Register(&intents.EventModule{Analyzer: eventAnalyzer})
	}
	if reminderAnalyzer != nil && reminderAnalyzer.IsConfigured() {
		_ = registry.Register(&intents.ReminderModule{Analyzer: reminderAnalyzer})
	}

	return &EmailProcessor{
		db:               db,
		eventAnalyzer:    eventAnalyzer,
		reminderAnalyzer: reminderAnalyzer,
		notifyService:    notifyService,
		eventCreator:     NewEventCreator(db, notifyService),
		reminderCreator:  NewReminderCreator(db, notifyService),
		intentRegistry:   registry,
		intentRouter:     intents.NewKeywordRouter(),
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

	if err := p.routeAnalyzeAndPersistEmail(ctx, emailSource, intents.EmailInput{Email: emailContent}); err != nil {
		fmt.Printf("Email intent orchestration error: %v\n", err)
	}

	return nil
}

type emailIntentPersister struct {
	p           *EmailProcessor
	emailSource *gmail.EmailSource
}

func (ep *emailIntentPersister) PersistEvent(ctx context.Context, analysis *agent.EventAnalysis) error {
	return ep.p.createPendingEventFromEmail(ep.emailSource, analysis)
}

func (ep *emailIntentPersister) PersistReminder(ctx context.Context, analysis *agent.ReminderAnalysis) error {
	return ep.p.createPendingReminderFromEmail(ep.emailSource, analysis)
}

func (p *EmailProcessor) routeAnalyzeAndPersistEmail(ctx context.Context, emailSource *gmail.EmailSource, input intents.EmailInput) error {
	if p.intentRegistry == nil || p.intentRouter == nil {
		return nil
	}

	route := p.intentRouter.RouteEmail(ctx, input)
	fmt.Printf("Email intent route: intent=%s confidence=%.2f reason=%s\n", route.Intent, route.Confidence, truncate(route.Reasoning, 80))

	intentOrder, unknownRoutedIntent := resolveIntentExecutionOrder(p.intentRegistry, route)
	if len(intentOrder) == 0 {
		return nil
	}

	var firstErr error
	for _, intentName := range intentOrder {
		if err := p.runEmailIntentModule(ctx, intentName, emailSource, input); err != nil {
			fmt.Printf("Email intent module %s error: %v\n", intentName, err)
			if firstErr == nil {
				firstErr = err
			}
		}
		if unknownRoutedIntent {
			break
		}
	}
	return firstErr
}

func (p *EmailProcessor) runEmailIntentModule(ctx context.Context, intentName string, emailSource *gmail.EmailSource, input intents.EmailInput) error {
	emailChannel, userID, channelErr := p.getOrCreateEmailChannel(emailSource)
	channelID := int64(0)
	if emailChannel != nil {
		channelID = emailChannel.ID
	}

	module, ok := p.intentRegistry.Get(intentName)
	if !ok {
		fmt.Printf("Unknown email intent '%s' -> no_action\n", intentName)
		if channelErr == nil && channelID != 0 {
			_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
				UserID:     userID,
				ChannelID:  channelID,
				SourceType: string(source.SourceTypeGmail),
				Intent:     intentName,
				Status:     "unknown_intent",
				Reasoning:  "intent module not registered",
			})
		}
		return nil
	}

	output, err := module.AnalyzeEmail(ctx, input)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}

	if err := module.Validate(ctx, output); err != nil {
		fmt.Printf("Email intent validation failed intent=%s: %v\n", intentName, err)
		if channelErr == nil && channelID != 0 {
			_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
				UserID:     userID,
				ChannelID:  channelID,
				SourceType: string(source.SourceTypeGmail),
				Intent:     intentName,
				Action:     output.Action,
				Confidence: output.Confidence,
				Reasoning:  output.Reasoning,
				Status:     "validation_failed",
			})
		}
		return nil
	}
	if output.Confidence < minPersistConfidence {
		fmt.Printf("Skipping low-confidence email intent=%s confidence=%.2f\n", intentName, output.Confidence)
		if channelErr == nil && channelID != 0 {
			_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
				UserID:     userID,
				ChannelID:  channelID,
				SourceType: string(source.SourceTypeGmail),
				Intent:     intentName,
				Action:     output.Action,
				Confidence: output.Confidence,
				Reasoning:  output.Reasoning,
				Status:     "skipped_low_confidence",
			})
		}
		return nil
	}

	err = module.Persist(ctx, output, &emailIntentPersister{p: p, emailSource: emailSource})
	status := "persisted"
	if err != nil {
		status = "persist_error"
	}
	if channelErr == nil && channelID != 0 {
		_ = p.db.CreateAnalysisTrace(database.AnalysisTrace{
			UserID:     userID,
			ChannelID:  channelID,
			SourceType: string(source.SourceTypeGmail),
			Intent:     intentName,
			Action:     output.Action,
			Confidence: output.Confidence,
			Reasoning:  output.Reasoning,
			Status:     status,
		})
	}
	return err
}

// createPendingEventFromEmail creates a pending event from email analysis
func (p *EmailProcessor) createPendingEventFromEmail(emailSource *gmail.EmailSource, analysis *agent.EventAnalysis) error {
	// Get or create a placeholder channel for email sources
	emailChannel, userID, err := p.getOrCreateEmailChannel(emailSource)
	if err != nil {
		return fmt.Errorf("failed to get email channel: %w", err)
	}

	emailSourceID := emailSource.ID
	params := EventCreationParams{
		UserID:        userID,
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
	emailChannel, userID, err := p.getOrCreateEmailChannel(emailSource)
	if err != nil {
		return fmt.Errorf("failed to get email channel: %w", err)
	}

	emailSourceID := emailSource.ID
	params := ReminderCreationParams{
		UserID:        userID,
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
func (p *EmailProcessor) getOrCreateEmailChannel(emailSource *gmail.EmailSource) (*database.SourceChannel, int64, error) {
	dbSource, err := p.db.GetEmailSourceByID(emailSource.ID)
	if err != nil {
		return nil, 0, err
	}
	if dbSource == nil {
		return nil, 0, fmt.Errorf("email source not found: %d", emailSource.ID)
	}

	// Use a channel identifier based on the email source
	identifier := fmt.Sprintf("email:%s:%s", emailSource.Type, emailSource.Identifier)

	// Check if channel exists
	channel, err := p.db.GetSourceChannelByIdentifier(dbSource.UserID, source.SourceTypeGmail, identifier)
	if err != nil {
		return nil, 0, err
	}
	if channel != nil {
		return channel, dbSource.UserID, nil
	}

	// Create new channel for this email source
	channel, err = p.db.CreateSourceChannel(
		dbSource.UserID,
		source.SourceTypeGmail,
		source.ChannelTypeSender,
		identifier,
		fmt.Sprintf("Email: %s", emailSource.Name),
	)
	if err != nil {
		return nil, 0, err
	}

	return channel, dbSource.UserID, nil
}
