package gmail

import (
	"fmt"
	"strings"
	"time"
)

// EmailSourceType represents the type of email source
type EmailSourceType string

const (
	SourceTypeCategory EmailSourceType = "category"
	SourceTypeSender   EmailSourceType = "sender"
	SourceTypeDomain   EmailSourceType = "domain"
)

// EmailSource represents a tracked email source
type EmailSource struct {
	ID         int64           `json:"id"`
	Type       EmailSourceType `json:"type"`
	Identifier string          `json:"identifier"` // e.g., "CATEGORY_PRIMARY", "john@example.com", "microsoft.com"
	Name       string          `json:"name"`       // Display name
	Enabled    bool            `json:"enabled"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// ScanResult represents an email that was scanned and may contain events
type ScanResult struct {
	Email      *Email
	Source     *EmailSource
	ShouldSkip bool
	SkipReason string
}

// Scanner scans emails from tracked sources
type Scanner struct {
	client *Client
}

// NewScanner creates a new email scanner
func NewScanner(client *Client) *Scanner {
	return &Scanner{client: client}
}

// BuildQueryForSource builds a Gmail search query for a source
func BuildQueryForSource(source *EmailSource) string {
	switch source.Type {
	case SourceTypeCategory:
		// Convert CATEGORY_PRIMARY to "category:primary"
		category := strings.ToLower(strings.TrimPrefix(source.Identifier, "CATEGORY_"))
		return fmt.Sprintf("category:%s", category)
	case SourceTypeSender:
		return fmt.Sprintf("from:%s", source.Identifier)
	case SourceTypeDomain:
		// For domains, we need to use the from: operator with wildcard
		domain := strings.TrimPrefix(source.Identifier, "@")
		return fmt.Sprintf("from:@%s", domain)
	default:
		return ""
	}
}

// BuildCombinedQuery builds a Gmail query for multiple sources
// Returns empty string if no valid sources
func BuildCombinedQuery(sources []*EmailSource, sinceTime *time.Time) string {
	var queries []string

	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		q := BuildQueryForSource(source)
		if q != "" {
			queries = append(queries, fmt.Sprintf("(%s)", q))
		}
	}

	if len(queries) == 0 {
		return ""
	}

	combined := strings.Join(queries, " OR ")

	// Add coarse date filter if provided. Gmail's after: operator is date-only,
	// so we still apply an exact timestamp cutoff after fetching each message.
	if sinceTime != nil {
		dateStr := sinceTime.Format("2006/01/02")
		combined = fmt.Sprintf("(%s) after:%s", combined, dateStr)
	}

	return combined
}

// ScanForEmails retrieves emails matching the tracked sources
func (s *Scanner) ScanForEmails(sources []*EmailSource, sinceTime *time.Time, maxResults int64) ([]*ScanResult, error) {
	if s.client == nil || !s.client.IsAuthenticated() {
		return nil, fmt.Errorf("Gmail client not authenticated")
	}

	query := BuildCombinedQuery(sources, sinceTime)
	if query == "" {
		return nil, nil // No enabled sources
	}

	messages, err := s.client.ListMessages(query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	var results []*ScanResult
	for _, msg := range messages {
		email, err := s.client.GetMessage(msg.Id)
		if err != nil {
			fmt.Printf("Warning: failed to get message %s: %v\n", msg.Id, err)
			continue
		}
		if !passesSinceTimeFilter(email, sinceTime) {
			continue
		}

		// Determine which source this email matches
		source := s.matchSource(email, sources)

		results = append(results, &ScanResult{
			Email:  email,
			Source: source,
		})
	}

	return results, nil
}

// ScanSourceEmails retrieves emails for a specific source
func (s *Scanner) ScanSourceEmails(source *EmailSource, sinceTime *time.Time, maxResults int64) ([]*ScanResult, error) {
	if s.client == nil || !s.client.IsAuthenticated() {
		return nil, fmt.Errorf("Gmail client not authenticated")
	}

	query := BuildQueryForSource(source)
	if query == "" {
		return nil, fmt.Errorf("invalid source configuration")
	}

	// Add coarse date filter if provided. Gmail's after: operator is date-only,
	// so we still apply an exact timestamp cutoff after fetching each message.
	if sinceTime != nil {
		dateStr := sinceTime.Format("2006/01/02")
		query = fmt.Sprintf("%s after:%s", query, dateStr)
	}

	messages, err := s.client.ListMessages(query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	var results []*ScanResult
	for _, msg := range messages {
		email, err := s.client.GetMessage(msg.Id)
		if err != nil {
			fmt.Printf("Warning: failed to get message %s: %v\n", msg.Id, err)
			continue
		}
		if !passesSinceTimeFilter(email, sinceTime) {
			continue
		}

		results = append(results, &ScanResult{
			Email:  email,
			Source: source,
		})
	}

	return results, nil
}

// matchSource finds which source an email matches
func (s *Scanner) matchSource(email *Email, sources []*EmailSource) *EmailSource {
	senderEmail := strings.ToLower(ExtractSenderEmail(email.From))
	senderDomain := ExtractDomain(senderEmail)

	for _, source := range sources {
		if !source.Enabled {
			continue
		}

		switch source.Type {
		case SourceTypeCategory:
			// Check if email has the category label
			categoryLabel := source.Identifier
			for _, label := range email.Labels {
				if label == categoryLabel {
					return source
				}
			}
		case SourceTypeSender:
			if strings.EqualFold(senderEmail, source.Identifier) {
				return source
			}
		case SourceTypeDomain:
			domain := strings.TrimPrefix(source.Identifier, "@")
			if strings.EqualFold(senderDomain, domain) {
				return source
			}
		}
	}

	return nil
}

// passesSinceTimeFilter applies an exact timestamp cutoff after message fetch.
func passesSinceTimeFilter(email *Email, sinceTime *time.Time) bool {
	if sinceTime == nil || email == nil {
		return true
	}
	if email.ReceivedAt.IsZero() {
		// If timestamp is unavailable, keep behavior permissive.
		return true
	}
	return email.ReceivedAt.After(*sinceTime)
}
