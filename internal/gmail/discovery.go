package gmail

import (
	"fmt"
	"sort"
	"strings"
)

// EmailSender represents a frequent email sender
type EmailSender struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	EmailCount int    `json:"email_count"`
}

// automatedSenderPatterns contains patterns to identify automated/non-human senders
var automatedSenderPatterns = []string{
	"noreply", "no-reply", "do-not-reply", "donotreply",
	"notifications", "notification", "notify",
	"calendar-notification", "calendar@google",
	"mailer-daemon", "postmaster", "bounce",
	"newsletter", "news@", "updates@",
	"automated", "auto@", "system@",
}

// isAutomatedSender checks if an email address belongs to an automated sender
func isAutomatedSender(email string) bool {
	email = strings.ToLower(email)
	for _, pattern := range automatedSenderPatterns {
		if strings.Contains(email, pattern) {
			return true
		}
	}
	return false
}

// DiscoverTopContacts finds the top N email contacts efficiently
// Uses metadata-only fetching and filters out automated senders
func (c *Client) DiscoverTopContacts(limit int) ([]EmailSender, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	// Use category:primary to exclude Promotions, Social, Updates at API level
	messages, err := c.ListMessages("in:inbox category:primary", 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Count senders using metadata-only fetch (much faster)
	senderCounts := make(map[string]*EmailSender)
	for _, msg := range messages {
		// Use efficient headers-only fetch instead of full message
		from, err := c.GetMessageHeaders(msg.Id)
		if err != nil || from == "" {
			continue
		}

		senderEmail := ExtractSenderEmail(from)
		senderName := ExtractSenderName(from)

		if senderEmail == "" {
			continue
		}

		// Skip automated senders (noreply, notifications, newsletters, etc.)
		if isAutomatedSender(senderEmail) {
			fmt.Printf("Gmail discovery: filtering out automated sender: %s\n", senderEmail)
			continue
		}

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
