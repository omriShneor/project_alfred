package gmail

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
)

// DBInterface defines the database operations needed by the Gmail worker
type DBInterface interface {
	IsEmailProcessed(emailID string) (bool, error)
	MarkEmailProcessed(emailID string) error
	GetGmailSettings() (*database.GmailSettings, error)
	UpdateGmailLastPoll() error
	ListEnabledEmailSources() ([]*database.EmailSource, error)
}

// EmailProcessor interface for processing emails
type EmailProcessor interface {
	ProcessEmail(ctx context.Context, email *Email, source *EmailSource) error
}

// Worker handles background email scanning and processing
type Worker struct {
	client       *Client
	scanner      *Scanner
	db           DBInterface
	processor    EmailProcessor
	pollInterval time.Duration
	maxEmails    int64

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// WorkerConfig contains configuration for the email worker
type WorkerConfig struct {
	PollIntervalMinutes int
	MaxEmailsPerPoll    int
}

// NewWorker creates a new Gmail worker
func NewWorker(client *Client, db DBInterface, processor EmailProcessor, config WorkerConfig) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	pollInterval := time.Duration(config.PollIntervalMinutes) * time.Minute
	if pollInterval <= 0 {
		pollInterval = 5 * time.Minute
	}

	maxEmails := int64(config.MaxEmailsPerPoll)
	if maxEmails <= 0 {
		maxEmails = 10
	}

	return &Worker{
		client:       client,
		scanner:      NewScanner(client),
		db:           db,
		processor:    processor,
		pollInterval: pollInterval,
		maxEmails:    maxEmails,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the background email scanning loop
// The worker always starts, but only polls when Gmail is enabled in database settings
func (w *Worker) Start() error {
	if w.client == nil || !w.client.IsAuthenticated() {
		fmt.Println("Gmail worker: client not authenticated, will poll when authenticated")
	}

	fmt.Printf("Gmail worker: starting with %v poll interval (enable/disable via settings)\n", w.pollInterval)

	w.wg.Add(1)
	go w.pollLoop()

	return nil
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop() {
	fmt.Println("Gmail worker: stopping...")
	w.cancel()
	w.wg.Wait()
	fmt.Println("Gmail worker: stopped")
}

// SetClient updates the Gmail client (used when OAuth completes)
func (w *Worker) SetClient(client *Client) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.client = client
	w.scanner = NewScanner(client)
}

// pollLoop runs the polling cycle
func (w *Worker) pollLoop() {
	defer w.wg.Done()

	// Do an initial poll after a short delay
	select {
	case <-w.ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Run first poll immediately
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

// poll performs a single polling cycle
func (w *Worker) poll() {
	w.mu.Lock()
	client := w.client
	scanner := w.scanner
	w.mu.Unlock()

	if client == nil || !client.IsAuthenticated() {
		return
	}

	// Check if Gmail is enabled in settings
	settings, err := w.db.GetGmailSettings()
	if err != nil {
		fmt.Printf("Gmail worker: failed to get settings: %v\n", err)
		return
	}
	if settings == nil || !settings.Enabled {
		return
	}

	// Get enabled sources from database
	dbSources, err := w.db.ListEnabledEmailSources()
	if err != nil {
		fmt.Printf("Gmail worker: failed to get sources: %v\n", err)
		return
	}
	if len(dbSources) == 0 {
		return
	}

	// Convert database sources to gmail sources
	sources := make([]*EmailSource, len(dbSources))
	for i, s := range dbSources {
		sources[i] = &EmailSource{
			ID:         s.ID,
			Type:       EmailSourceType(s.Type),
			Identifier: s.Identifier,
			Name:       s.Name,
			Enabled:    s.Enabled,
			CalendarID: s.CalendarID,
			CreatedAt:  s.CreatedAt,
			UpdatedAt:  s.UpdatedAt,
		}
	}

	// Determine the time range for scanning
	var sinceTime *time.Time
	if settings.LastPollAt != nil {
		// Go back a bit to ensure we don't miss any emails
		t := settings.LastPollAt.Add(-5 * time.Minute)
		sinceTime = &t
	} else {
		// First poll - look at last 24 hours
		t := time.Now().Add(-24 * time.Hour)
		sinceTime = &t
	}

	// Scan for emails
	results, err := scanner.ScanForEmails(sources, sinceTime, w.maxEmails)
	if err != nil {
		fmt.Printf("Gmail worker: failed to scan emails: %v\n", err)
		return
	}

	if len(results) == 0 {
		if err := w.db.UpdateGmailLastPoll(); err != nil {
			fmt.Printf("Gmail worker: failed to update last poll: %v\n", err)
		}
		return
	}

	fmt.Printf("Gmail worker: found %d emails to process\n", len(results))

	// Process each email
	processedCount := 0
	for _, result := range results {
		// Check if already processed
		processed, err := w.db.IsEmailProcessed(result.Email.ID)
		if err != nil {
			fmt.Printf("Gmail worker: failed to check processed status: %v\n", err)
			continue
		}
		if processed {
			continue
		}

		// Process the email
		if w.processor != nil && result.Source != nil {
			if err := w.processor.ProcessEmail(w.ctx, result.Email, result.Source); err != nil {
				fmt.Printf("Gmail worker: failed to process email %s: %v\n", result.Email.ID, err)
				// Continue with other emails
			}
		}

		// Mark as processed
		if err := w.db.MarkEmailProcessed(result.Email.ID); err != nil {
			fmt.Printf("Gmail worker: failed to mark email processed: %v\n", err)
		}
		processedCount++
	}

	fmt.Printf("Gmail worker: processed %d new emails\n", processedCount)

	// Update last poll time
	if err := w.db.UpdateGmailLastPoll(); err != nil {
		fmt.Printf("Gmail worker: failed to update last poll: %v\n", err)
	}
}

// PollNow triggers an immediate poll (for testing or manual trigger)
func (w *Worker) PollNow() {
	go w.poll()
}
