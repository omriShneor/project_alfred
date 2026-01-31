package gmail

import (
	"fmt"
	"sort"
	"strings"
)

const (
	NumMessagesToQuery = 400
)

// EmailSender represents a frequent email sender
type EmailSender struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	EmailCount int    `json:"email_count"`
}

// automatedSenderPatterns contains patterns to identify automated/non-human senders
var automatedSenderPatterns = []string{
	"noreply", "no-reply", "do-not-reply", "donotreply", "do_not_reply",
	"notifications", "notification", "notify",
	"calendar-notification", "calendar@google",
	"mailer-daemon", "postmaster", "bounce",
	"newsletter", "news@", "updates@",
	"automated", "auto@", "system@",
}

// buildExcludeFromQuery builds a Gmail search query string excluding automated senders
func buildExcludeFromQuery() string {
	var parts []string
	for _, pattern := range automatedSenderPatterns {
		parts = append(parts, "-from:"+pattern)
	}
	return strings.Join(parts, " ")
}

// DiscoverTopContacts finds the top N email contacts efficiently
// Uses metadata-only fetching and filters out automated senders
// TODO: Consider adding a simple ML spam filter to decide if a message is human or an auto-reply/mail-list
// since the current naive solution is not good enough.
func (c *Client) DiscoverTopContacts(limit int) ([]EmailSender, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	// Use category:primary to exclude Promotions, Social, Updates at API level
	// Also exclude common automated senders via Gmail search
	query := "in:inbox category:primary " + buildExcludeFromQuery()
	messages, err := c.ListMessages(query, NumMessagesToQuery)
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

	// Convert to slice and sort by count all senders with > 1 messages
	senders := make([]EmailSender, 0, len(senderCounts))
	for _, sender := range senderCounts {
		if sender.EmailCount > 1 {
			senders = append(senders, *sender)
		}
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
