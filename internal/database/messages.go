package database

import (
	"fmt"
	"time"
)

// MessageRecord represents a stored message in the history
type MessageRecord struct {
	ID          int64     `json:"id"`
	ChannelID   int64     `json:"channel_id"`
	SenderJID   string    `json:"sender_jid"`
	SenderName  string    `json:"sender_name"`
	MessageText string    `json:"message_text"`
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
}

// StoreMessage saves a message to the history
func (d *DB) StoreMessage(channelID int64, senderJID, senderName, text string, timestamp time.Time) (*MessageRecord, error) {
	result, err := d.Exec(`
		INSERT INTO message_history (channel_id, sender_jid, sender_name, message_text, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`, channelID, senderJID, senderName, text, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to store message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get message id: %w", err)
	}

	return &MessageRecord{
		ID:          id,
		ChannelID:   channelID,
		SenderJID:   senderJID,
		SenderName:  senderName,
		MessageText: text,
		Timestamp:   timestamp,
		CreatedAt:   time.Now(),
	}, nil
}

// GetMessageHistory retrieves the last N messages for a channel, ordered by timestamp descending
func (d *DB) GetMessageHistory(channelID int64, limit int) ([]MessageRecord, error) {
	rows, err := d.Query(`
		SELECT id, channel_id, sender_jid, sender_name, message_text, timestamp, created_at
		FROM message_history
		WHERE channel_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query message history: %w", err)
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.SenderJID, &m.SenderName, &m.MessageText, &m.Timestamp, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// PruneMessages keeps only the last N messages for a channel, deleting older ones
func (d *DB) PruneMessages(channelID int64, keepCount int) error {
	_, err := d.Exec(`
		DELETE FROM message_history
		WHERE channel_id = ? AND id NOT IN (
			SELECT id FROM message_history
			WHERE channel_id = ?
			ORDER BY timestamp DESC
			LIMIT ?
		)
	`, channelID, channelID, keepCount)
	if err != nil {
		return fmt.Errorf("failed to prune messages: %w", err)
	}
	return nil
}

// GetMessageByID retrieves a specific message by ID
func (d *DB) GetMessageByID(id int64) (*MessageRecord, error) {
	var m MessageRecord
	err := d.QueryRow(`
		SELECT id, channel_id, sender_jid, sender_name, message_text, timestamp, created_at
		FROM message_history
		WHERE id = ?
	`, id).Scan(&m.ID, &m.ChannelID, &m.SenderJID, &m.SenderName, &m.MessageText, &m.Timestamp, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &m, nil
}

// CountMessages returns the number of messages stored for a channel
func (d *DB) CountMessages(channelID int64) (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM message_history WHERE channel_id = ?`, channelID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}
	return count, nil
}
