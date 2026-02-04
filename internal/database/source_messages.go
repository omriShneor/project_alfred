package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
)

// SourceMessage represents a stored message in the history with source type
type SourceMessage struct {
	ID          int64             `json:"id"`
	SourceType  source.SourceType `json:"source_type"`
	ChannelID   int64             `json:"channel_id"`
	SenderID    string            `json:"sender_id"`
	SenderName  string            `json:"sender_name"`
	MessageText string            `json:"message_text"`
	Subject     string            `json:"subject"` // For emails
	Timestamp   time.Time         `json:"timestamp"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ToHistoryMessage converts a SourceMessage to source.HistoryMessage
func (sm *SourceMessage) ToHistoryMessage() source.HistoryMessage {
	return source.HistoryMessage{
		ID:         sm.ID,
		SenderID:   sm.SenderID,
		SenderName: sm.SenderName,
		Text:       sm.MessageText,
		Subject:    sm.Subject,
		Timestamp:  sm.Timestamp,
	}
}

// StoreSourceMessage saves a message to the history with source type
func (d *DB) StoreSourceMessage(sourceType source.SourceType, channelID int64, senderID, senderName, text, subject string, timestamp time.Time) (*SourceMessage, error) {
	// user_id is derived from the channel's user_id via subquery
	result, err := d.Exec(`
		INSERT INTO message_history (user_id, source_type, channel_id, sender_jid, sender_name, message_text, subject, timestamp)
		SELECT user_id, ?, ?, ?, ?, ?, ?, ?
		FROM channels
		WHERE id = ?
	`, sourceType, channelID, senderID, senderName, text, subject, timestamp, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to store source message: %w", err)
	}

	// Check if any row was actually inserted (channel must exist)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("channel %d does not exist", channelID)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get message id: %w", err)
	}

	return &SourceMessage{
		ID:          id,
		SourceType:  sourceType,
		ChannelID:   channelID,
		SenderID:    senderID,
		SenderName:  senderName,
		MessageText: text,
		Subject:     subject,
		Timestamp:   timestamp,
		CreatedAt:   time.Now(),
	}, nil
}

// GetSourceMessageHistory retrieves the last N messages for a source type and channel, ordered chronologically
func (d *DB) GetSourceMessageHistory(sourceType source.SourceType, channelID int64, limit int) ([]SourceMessage, error) {
	rows, err := d.Query(`
		SELECT id, COALESCE(source_type, 'whatsapp'), channel_id, sender_jid, sender_name, message_text, COALESCE(subject, ''), timestamp, created_at
		FROM message_history
		WHERE source_type = ? AND channel_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, sourceType, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query source message history: %w", err)
	}
	defer rows.Close()

	var messages []SourceMessage
	for rows.Next() {
		var m SourceMessage
		if err := rows.Scan(&m.ID, &m.SourceType, &m.ChannelID, &m.SenderID, &m.SenderName, &m.MessageText, &m.Subject, &m.Timestamp, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan source message: %w", err)
		}
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating source messages: %w", err)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// PruneSourceMessages keeps only the last N messages for a source type and channel
func (d *DB) PruneSourceMessages(sourceType source.SourceType, channelID int64, keepCount int) error {
	_, err := d.Exec(`
		DELETE FROM message_history
		WHERE source_type = ? AND channel_id = ? AND id NOT IN (
			SELECT id FROM message_history
			WHERE source_type = ? AND channel_id = ?
			ORDER BY timestamp DESC
			LIMIT ?
		)
	`, sourceType, channelID, sourceType, channelID, keepCount)
	if err != nil {
		return fmt.Errorf("failed to prune source messages: %w", err)
	}
	return nil
}

// GetSourceMessageByID retrieves a specific message by ID with source type validation
func (d *DB) GetSourceMessageByID(id int64) (*SourceMessage, error) {
	var m SourceMessage
	err := d.QueryRow(`
		SELECT id, COALESCE(source_type, 'whatsapp'), channel_id, sender_jid, sender_name, message_text, COALESCE(subject, ''), timestamp, created_at
		FROM message_history
		WHERE id = ?
	`, id).Scan(&m.ID, &m.SourceType, &m.ChannelID, &m.SenderID, &m.SenderName, &m.MessageText, &m.Subject, &m.Timestamp, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source message: %w", err)
	}
	return &m, nil
}

// CountSourceMessages returns the number of messages stored for a source type and channel
func (d *DB) CountSourceMessages(sourceType source.SourceType, channelID int64) (int, error) {
	var count int
	err := d.QueryRow(`
		SELECT COUNT(*) FROM message_history
		WHERE source_type = ? AND channel_id = ?
	`, sourceType, channelID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count source messages: %w", err)
	}
	return count, nil
}

// GetAllSourceMessages retrieves all messages for a source type (useful for debugging)
func (d *DB) GetAllSourceMessages(sourceType source.SourceType, limit int) ([]SourceMessage, error) {
	rows, err := d.Query(`
		SELECT id, COALESCE(source_type, 'whatsapp'), channel_id, sender_jid, sender_name, message_text, COALESCE(subject, ''), timestamp, created_at
		FROM message_history
		WHERE source_type = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, sourceType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query all source messages: %w", err)
	}
	defer rows.Close()

	var messages []SourceMessage
	for rows.Next() {
		var m SourceMessage
		if err := rows.Scan(&m.ID, &m.SourceType, &m.ChannelID, &m.SenderID, &m.SenderName, &m.MessageText, &m.Subject, &m.Timestamp, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan source message: %w", err)
		}
		messages = append(messages, m)
	}

	return messages, rows.Err()
}
