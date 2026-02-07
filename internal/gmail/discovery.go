package gmail

import (
	"fmt"
	"math"
	"net/mail"
	"sort"
	"strings"
	"time"
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

// DiscoverContacts finds all email contacts from recent messages.
// Uses metadata-only fetching and filters out automated senders.
// TODO: Consider adding a simple ML spam filter to decide if a message is human or an auto-reply/mail-list
// since the current naive solution is not good enough.
func (c *Client) DiscoverContacts() ([]EmailSender, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	// Use category:primary to exclude Promotions, Social, Updates at API level
	// Also exclude common automated senders via Gmail search
	inboxQuery := "in:inbox category:primary " + buildExcludeFromQuery()
	sentQuery := "in:sent"

	inboxMessages, err := c.ListMessages(inboxQuery, NumMessagesToQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to list inbox messages: %w", err)
	}

	sentMessages, err := c.ListMessages(sentQuery, NumMessagesToQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to list sent messages: %w", err)
	}

	type senderScore struct {
		Email string
		Name  string
		Score float64
	}

	senderScores := make(map[string]*senderScore)

	addScore := func(email, name string, baseWeight float64, internalDateMs int64, dateHeader string) {
		email = normalizeEmail(email)
		if email == "" {
			return
		}

		t := parseMessageTime(internalDateMs, dateHeader)
		weight := baseWeight * recencyMultiplier(t)

		if sender, exists := senderScores[email]; exists {
			sender.Score += weight
			if sender.Name == "" && name != "" {
				sender.Name = name
			}
		} else {
			senderScores[email] = &senderScore{
				Email: email,
				Name:  name,
				Score: weight,
			}
		}
	}

	// Count inbox senders using metadata-only fetch (much faster)
	for _, msg := range inboxMessages {
		headers, internalDateMs, err := c.GetMessageMetadata(msg.Id, "From", "Date")
		if err != nil {
			continue
		}

		from := headers["from"]
		if from == "" {
			continue
		}

		senderEmail := ExtractSenderEmail(from)
		senderName := ExtractSenderName(from)
		addScore(senderEmail, senderName, 1.0, internalDateMs, headers["date"])
	}

	// Count sent recipients with lower weight
	for _, msg := range sentMessages {
		headers, internalDateMs, err := c.GetMessageMetadata(msg.Id, "To", "Date")
		if err != nil {
			continue
		}

		toHeader := headers["to"]
		if toHeader == "" {
			continue
		}

		recipients := splitAddressList(toHeader)
		for _, r := range recipients {
			recipientEmail := ExtractSenderEmail(r)
			recipientName := ExtractSenderName(r)
			addScore(recipientEmail, recipientName, 0.5, internalDateMs, headers["date"])
		}
	}

	// Convert to slice and sort by score
	scoreList := make([]*senderScore, 0, len(senderScores))
	for _, sender := range senderScores {
		if sender.Score <= 0 {
			continue
		}
		scoreList = append(scoreList, sender)
	}

	sort.Slice(scoreList, func(i, j int) bool {
		return scoreList[i].Score > scoreList[j].Score
	})

	// Build response — return all contacts with score > 0, sorted by score
	senders := make([]EmailSender, 0, len(scoreList))
	for _, s := range scoreList {
		senders = append(senders, EmailSender{
			Email:      s.Email,
			Name:       s.Name,
			EmailCount: int(math.Round(s.Score)),
		})
	}

	return senders, nil
}

func recencyMultiplier(t time.Time) float64 {
	if t.IsZero() {
		return 1.0
	}
	age := time.Since(t)
	switch {
	case age <= 30*24*time.Hour:
		return 1.5
	case age <= 90*24*time.Hour:
		return 1.0
	case age <= 365*24*time.Hour:
		return 0.7
	default:
		return 0.4
	}
}

func parseMessageTime(internalDateMs int64, dateHeader string) time.Time {
	if internalDateMs > 0 {
		return time.UnixMilli(internalDateMs)
	}
	if dateHeader == "" {
		return time.Time{}
	}
	if t, err := mail.ParseDate(dateHeader); err == nil {
		return t
	}
	return time.Time{}
}

func splitAddressList(toHeader string) []string {
	parts := strings.Split(toHeader, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func normalizeEmail(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	local, domain := parts[0], parts[1]
	if domain == "gmail.com" || domain == "googlemail.com" {
		if plusIdx := strings.Index(local, "+"); plusIdx >= 0 {
			local = local[:plusIdx]
		}
	}
	return local + "@" + domain
}
