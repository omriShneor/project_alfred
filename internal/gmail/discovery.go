package gmail

import (
	"fmt"
	"sort"
	"strings"
)

// GmailCategory represents a Gmail category
type GmailCategory struct {
	ID          string `json:"id"`          // e.g., "CATEGORY_PRIMARY"
	Name        string `json:"name"`        // e.g., "Primary"
	Description string `json:"description"` // Human-readable description
	EmailCount  int    `json:"email_count"` // Approximate count of emails
}

// EmailSender represents a frequent email sender
type EmailSender struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	EmailCount int    `json:"email_count"`
}

// EmailDomain represents a frequent email domain
type EmailDomain struct {
	Domain     string `json:"domain"` // e.g., "microsoft.com"
	EmailCount int    `json:"email_count"`
}

// Standard Gmail categories
var GmailCategories = []GmailCategory{
	{ID: "CATEGORY_PRIMARY", Name: "Primary", Description: "Important personal emails"},
	{ID: "CATEGORY_SOCIAL", Name: "Social", Description: "Social network notifications"},
	{ID: "CATEGORY_PROMOTIONS", Name: "Promotions", Description: "Marketing and promotional emails"},
	{ID: "CATEGORY_UPDATES", Name: "Updates", Description: "Receipts, confirmations, statements"},
	{ID: "CATEGORY_FORUMS", Name: "Forums", Description: "Mailing lists and forums"},
}

// DiscoverCategories returns Gmail categories with email counts
func (c *Client) DiscoverCategories() ([]GmailCategory, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	categories := make([]GmailCategory, len(GmailCategories))
	copy(categories, GmailCategories)

	for i := range categories {
		// Query for emails in each category
		query := fmt.Sprintf("category:%s", strings.ToLower(strings.TrimPrefix(categories[i].ID, "CATEGORY_")))
		messages, err := c.ListMessages(query, 1)
		if err != nil {
			// If we can't count, just continue
			continue
		}
		// Use the list count as an approximation
		// Note: This is a rough estimate since we're only getting the first page
		if len(messages) > 0 {
			categories[i].EmailCount = len(messages)
		}
	}

	return categories, nil
}

// DiscoverSenders finds frequent email senders from recent emails
func (c *Client) DiscoverSenders(limit int) ([]EmailSender, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	// Get recent emails from inbox
	messages, err := c.ListMessages("in:inbox", 500)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Count senders
	senderCounts := make(map[string]*EmailSender)
	for _, msg := range messages {
		email, err := c.GetMessage(msg.Id)
		if err != nil {
			continue
		}

		senderEmail := ExtractSenderEmail(email.From)
		senderName := ExtractSenderName(email.From)

		if sender, exists := senderCounts[senderEmail]; exists {
			sender.EmailCount++
		} else {
			senderCounts[senderEmail] = &EmailSender{
				Email:      senderEmail,
				Name:       senderName,
				EmailCount: 1,
			}
		}
	}

	// Convert to slice and sort by count
	senders := make([]EmailSender, 0, len(senderCounts))
	for _, sender := range senderCounts {
		senders = append(senders, *sender)
	}

	sort.Slice(senders, func(i, j int) bool {
		return senders[i].EmailCount > senders[j].EmailCount
	})

	// Limit results
	if len(senders) > limit {
		senders = senders[:limit]
	}

	return senders, nil
}

// DiscoverDomains finds frequent email domains from recent emails
func (c *Client) DiscoverDomains(limit int) ([]EmailDomain, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	// Get recent emails from inbox
	messages, err := c.ListMessages("in:inbox", 500)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Count domains
	domainCounts := make(map[string]int)
	for _, msg := range messages {
		email, err := c.GetMessage(msg.Id)
		if err != nil {
			continue
		}

		senderEmail := ExtractSenderEmail(email.From)
		domain := ExtractDomain(senderEmail)
		if domain != "" {
			domainCounts[domain]++
		}
	}

	// Convert to slice and sort by count
	domains := make([]EmailDomain, 0, len(domainCounts))
	for domain, count := range domainCounts {
		domains = append(domains, EmailDomain{
			Domain:     domain,
			EmailCount: count,
		})
	}

	sort.Slice(domains, func(i, j int) bool {
		return domains[i].EmailCount > domains[j].EmailCount
	})

	// Limit results
	if len(domains) > limit {
		domains = domains[:limit]
	}

	return domains, nil
}
