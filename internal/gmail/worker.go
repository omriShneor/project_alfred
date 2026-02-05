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
	GetGmailSettings(userID int64) (*database.GmailSettings, error)
	UpdateGmailLastPoll(userID int64) error
	ListEnabledEmailSources(userID int64) ([]*database.EmailSource, error)
	// Top contacts caching
	GetTopContacts(userID int64, limit int) ([]database.TopContact, error)
	ReplaceTopContacts(userID int64, contacts []database.TopContact) error
	GetTopContactsComputedAt(userID int64) (*time.Time, error)
	SetTopContactsComputedAt(userID int64, t time.Time) error
}

// EmailProcessor interface for processing emails
type EmailProcessor interface {
	ProcessEmail(ctx context.Context, email *Email, source *EmailSource, thread *Thread) error
}

// Worker handles background email scanning and processing
type Worker struct {
	client       *Client
	scanner      *Scanner
	db           DBInterface
	processor    EmailProcessor
	userID       int64 // User this worker is processing for
	pollInterval time.Duration
	maxEmails    int64

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// WorkerConfig contains configuration for the email worker
type WorkerConfig struct {
	UserID              int64
	PollIntervalMinutes int
	MaxEmailsPerPoll    int
}

// NewWorker creates a new Gmail worker for a specific user
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
		userID:       config.UserID,
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

	// Run first poll and check if top contacts need refreshing (every 3 days)
	w.poll()
	w.RefreshTopIfNeeded()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.poll()
			w.RefreshTopIfNeeded()
		}
	}
}

// poll performs a single polling cycle
func (w *Worker) poll() {
	// Skip if no valid user - services not started yet
	if w.userID == 0 {
		return
	}

	w.mu.Lock()
	client := w.client
	scanner := w.scanner
	w.mu.Unlock()

	if client == nil || !client.IsAuthenticated() {
		return
	}

	// Check if Gmail is enabled in settings
	settings, err := w.db.GetGmailSettings(w.userID)
	if err != nil {
		fmt.Printf("Gmail worker: failed to get settings: %v\n", err)
		return
	}
	if settings == nil || !settings.Enabled {
		return
	}

	// Get enabled sources from database
	dbSources, err := w.db.ListEnabledEmailSources(w.userID)
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
		if err := w.db.UpdateGmailLastPoll(w.userID); err != nil {
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

		// Fetch thread context if available
		var thread *Thread
		if result.Email.ThreadID != "" {
			thread, err = client.GetThread(result.Email.ThreadID, 10)
			if err != nil {
				fmt.Printf("Gmail worker: warning - failed to get thread %s: %v\n", result.Email.ThreadID, err)
				// Continue without thread context (graceful degradation)
			}
		}

		// Process the email with thread context
		if w.processor != nil && result.Source != nil {
			if err := w.processor.ProcessEmail(w.ctx, result.Email, result.Source, thread); err != nil {
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
	if err := w.db.UpdateGmailLastPoll(w.userID); err != nil {
		fmt.Printf("Gmail worker: failed to update last poll: %v\n", err)
	}
}

// PollNow triggers an immediate poll (for testing or manual trigger)
func (w *Worker) PollNow() {
	go w.poll()
}

// maybeRefreshTopContacts checks if top contacts need refreshing (every 3 days)
func (w *Worker) RefreshTopIfNeeded() {
	lastComputed, err := w.db.GetTopContactsComputedAt(w.userID)
	if err != nil {
		fmt.Printf("Gmail worker: failed to get top contacts computed at: %v\n", err)
		return
	}

	// Refresh if never computed or older than 24 hours
	needsRefresh := lastComputed == nil || time.Since(*lastComputed) > 24*time.Hour

	if needsRefresh {
		go w.RefreshTopContacts()
	}
}

// RefreshTopContacts fetches and caches top contacts
func (w *Worker) RefreshTopContacts() {
	w.mu.Lock()
	client := w.client
	w.mu.Unlock()

	if client == nil {
		fmt.Println("Gmail worker: cannot refresh top contacts - client is nil")
		return
	}
	if !client.IsAuthenticated() {
		fmt.Println("Gmail worker: cannot refresh top contacts - client not authenticated")
		return
	}

	fmt.Println("Gmail worker: refreshing top contacts...")

	contacts, err := client.DiscoverTopContacts(8)
	if err != nil {
		fmt.Printf("Gmail worker: failed to discover top contacts: %v\n", err)
		return
	}

	// Convert to database type
	dbContacts := make([]database.TopContact, len(contacts))
	for i, c := range contacts {
		dbContacts[i] = database.TopContact{
			Email:      c.Email,
			Name:       c.Name,
			EmailCount: c.EmailCount,
		}
	}

	if err := w.db.ReplaceTopContacts(w.userID, dbContacts); err != nil {
		fmt.Printf("Gmail worker: failed to replace top contacts: %v\n", err)
		return
	}

	if err := w.db.SetTopContactsComputedAt(w.userID, time.Now()); err != nil {
		fmt.Printf("Gmail worker: failed to set top contacts computed at: %v\n", err)
		return
	}

	fmt.Printf("Gmail worker: cached %d top contacts\n", len(dbContacts))
}
