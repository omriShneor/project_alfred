package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps the Gmail API client
type Client struct {
	service *gmail.Service
	token   *oauth2.Token
	config  *oauth2.Config
}

// Email represents a parsed email message
type Email struct {
	ID         string
	ThreadID   string
	Subject    string
	From       string
	To         string
	Date       string
	ReceivedAt time.Time
	Body       string // Plain text body
	Snippet    string
	Labels     []string
	MessageID  string // RFC 2822 Message-ID header
}

// NewClient creates a new Gmail client using an existing OAuth2 config and token
// This reuses the same credentials as Google Calendar
func NewClient(config *oauth2.Config, token *oauth2.Token) (*Client, error) {
	if token == nil {
		return &Client{config: config}, nil
	}

	client := &Client{
		config: config,
		token:  token,
	}

	if err := client.initService(context.Background()); err != nil {
		return nil, err
	}

	return client, nil
}

// initService initializes the Gmail service with the current token
func (c *Client) initService(ctx context.Context) error {
	if c.token == nil {
		return fmt.Errorf("no token available")
	}

	httpClient := c.config.Client(ctx, c.token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	c.service = service
	return nil
}

// SetToken updates the token and reinitializes the service
func (c *Client) SetToken(token *oauth2.Token) error {
	c.token = token
	return c.initService(context.Background())
}

// IsAuthenticated returns true if the client has a valid service
func (c *Client) IsAuthenticated() bool {
	return c.service != nil
}

// ListMessages retrieves messages matching the query
// query follows Gmail search syntax (e.g., "is:unread", "from:user@example.com", "label:INBOX")
func (c *Client) ListMessages(query string, maxResults int64) ([]*gmail.Message, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	call := c.service.Users.Messages.List("me").Q(query).MaxResults(maxResults)
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	return resp.Messages, nil
}

// GetMessage retrieves a full message by ID
func (c *Client) GetMessage(messageID string) (*Email, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	msg, err := c.service.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return c.parseMessage(msg), nil
}

// GetMessageHeaders retrieves only the From header of a message (much faster than full message)
func (c *Client) GetMessageHeaders(messageID string) (from string, err error) {
	if c.service == nil {
		return "", fmt.Errorf("Gmail service not initialized")
	}

	msg, err := c.service.Users.Messages.Get("me", messageID).
		Format("metadata").
		MetadataHeaders("From").
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to get message headers: %w", err)
	}

	for _, header := range msg.Payload.Headers {
		if strings.EqualFold(header.Name, "From") {
			return header.Value, nil
		}
	}

	return "", nil
}

// GetMessageMetadata retrieves selected headers plus internal date (metadata-only).
func (c *Client) GetMessageMetadata(messageID string, headerNames ...string) (map[string]string, int64, error) {
	if c.service == nil {
		return nil, 0, fmt.Errorf("Gmail service not initialized")
	}

	call := c.service.Users.Messages.Get("me", messageID).
		Format("metadata").
		MetadataHeaders(headerNames...)
	msg, err := call.Do()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get message metadata: %w", err)
	}

	headers := make(map[string]string, len(headerNames))
	for _, header := range msg.Payload.Headers {
		headers[strings.ToLower(header.Name)] = header.Value
	}

	return headers, msg.InternalDate, nil
}

// GetMessagesSince retrieves messages received after a specific history ID or timestamp
// Use query like "after:2024/01/20" for date-based filtering
func (c *Client) GetMessagesSince(query string, maxResults int64) ([]*Email, error) {
	messages, err := c.ListMessages(query, maxResults)
	if err != nil {
		return nil, err
	}

	var emails []*Email
	for _, msg := range messages {
		email, err := c.GetMessage(msg.Id)
		if err != nil {
			// Log but continue with other messages
			fmt.Printf("Warning: failed to get message %s: %v\n", msg.Id, err)
			continue
		}
		emails = append(emails, email)
	}

	return emails, nil
}

// parseMessage converts a Gmail API message to our Email struct
func (c *Client) parseMessage(msg *gmail.Message) *Email {
	email := &Email{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
		Labels:   msg.LabelIds,
	}

	// Gmail internal date is epoch milliseconds in UTC.
	if msg.InternalDate > 0 {
		email.ReceivedAt = time.Unix(0, msg.InternalDate*int64(time.Millisecond)).UTC()
	}

	// Extract headers
	for _, header := range msg.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "subject":
			email.Subject = header.Value
		case "from":
			email.From = header.Value
		case "to":
			email.To = header.Value
		case "date":
			email.Date = header.Value
		case "message-id":
			email.MessageID = header.Value
		}
	}

	// Extract body
	email.Body = c.extractBody(msg.Payload)

	return email
}

// extractBody extracts plain text body from message payload
func (c *Client) extractBody(payload *gmail.MessagePart) string {
	// First try to find plain text part
	if payload.MimeType == "text/plain" {
		return decodeBase64(payload.Body.Data)
	}

	// Check multipart messages
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" {
			return decodeBase64(part.Body.Data)
		}
		// Recursively check nested parts
		if len(part.Parts) > 0 {
			if body := c.extractBodyFromParts(part.Parts); body != "" {
				return body
			}
		}
	}

	// Fallback to HTML if no plain text found
	for _, part := range payload.Parts {
		if part.MimeType == "text/html" {
			return decodeBase64(part.Body.Data)
		}
	}

	// Last resort: use the body data directly if available
	if payload.Body != nil && payload.Body.Data != "" {
		return decodeBase64(payload.Body.Data)
	}

	return ""
}

// extractBodyFromParts recursively extracts body from message parts
func (c *Client) extractBodyFromParts(parts []*gmail.MessagePart) string {
	for _, part := range parts {
		if part.MimeType == "text/plain" {
			return decodeBase64(part.Body.Data)
		}
		if len(part.Parts) > 0 {
			if body := c.extractBodyFromParts(part.Parts); body != "" {
				return body
			}
		}
	}
	return ""
}

// decodeBase64 decodes base64 URL-encoded data
func decodeBase64(data string) string {
	if data == "" {
		return ""
	}
	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		// Try standard base64
		decoded, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			return ""
		}
	}
	return string(decoded)
}

// ThreadMessage represents a message within an email thread
type ThreadMessage struct {
	ID       string
	From     string
	To       string
	Date     string
	Subject  string
	Body     string
	Snippet  string
	IsLatest bool // True for the most recent message in the thread
}

// Thread represents an email thread with all its messages
type Thread struct {
	ID       string
	Messages []ThreadMessage
}

// GetThread retrieves a thread by ID with up to maxMessages
func (c *Client) GetThread(threadID string, maxMessages int) (*Thread, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	thread, err := c.service.Users.Threads.Get("me", threadID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	result := &Thread{
		ID:       thread.Id,
		Messages: make([]ThreadMessage, 0, len(thread.Messages)),
	}

	// Process messages (oldest first from API, limit to last N)
	startIdx := 0
	if len(thread.Messages) > maxMessages {
		startIdx = len(thread.Messages) - maxMessages
	}

	for i := startIdx; i < len(thread.Messages); i++ {
		msg := thread.Messages[i]
		parsed := c.parseMessage(msg)

		result.Messages = append(result.Messages, ThreadMessage{
			ID:       parsed.ID,
			From:     parsed.From,
			To:       parsed.To,
			Date:     parsed.Date,
			Subject:  parsed.Subject,
			Body:     CleanEmailBody(parsed.Body),
			Snippet:  parsed.Snippet,
			IsLatest: i == len(thread.Messages)-1,
		})
	}

	return result, nil
}

// GetLabels returns all labels in the user's mailbox
func (c *Client) GetLabels() ([]*gmail.Label, error) {
	if c.service == nil {
		return nil, fmt.Errorf("Gmail service not initialized")
	}

	resp, err := c.service.Users.Labels.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}

	return resp.Labels, nil
}
