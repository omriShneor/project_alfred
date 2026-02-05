package database

import (
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
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

// TopContactStats represents a contact with message count for top contacts feature
type TopContactStats struct {
	ChannelID    int64  `json:"channel_id"`
	Identifier   string `json:"identifier"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	MessageCount int    `json:"message_count"`
	IsTracked    bool   `json:"is_tracked"`
}

// GetTopContactsBySourceType returns top contacts based on message count for a given source type
func (d *DB) GetTopContactsBySourceType(sourceType string, limit int) ([]TopContactStats, error) {
	rows, err := d.Query(`
		SELECT
			c.id,
			c.identifier,
			c.name,
			c.type,
			COUNT(mh.id) as message_count,
			c.enabled as is_tracked
		FROM channels c
		LEFT JOIN message_history mh ON c.id = mh.channel_id
		WHERE c.source_type = ?
		GROUP BY c.id
		HAVING message_count > 0
		ORDER BY message_count DESC
		LIMIT ?
	`, sourceType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top contacts: %w", err)
	}
	defer rows.Close()

	var contacts []TopContactStats
	for rows.Next() {
		var c TopContactStats
		if err := rows.Scan(&c.ChannelID, &c.Identifier, &c.Name, &c.Type, &c.MessageCount, &c.IsTracked); err != nil {
			return nil, fmt.Errorf("failed to scan top contact: %w", err)
		}
		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}

// GetTopContactsBySourceTypeForUser returns top contacts based on message count for a user and source type.
// This is a fallback when channel-level total_message_count isn't available yet.
func (d *DB) GetTopContactsBySourceTypeForUser(userID int64, sourceType source.SourceType, limit int) ([]TopContactStats, error) {
	rows, err := d.Query(`
		SELECT
			c.id,
			c.identifier,
			c.name,
			c.type,
			COUNT(mh.id) as message_count,
			c.enabled as is_tracked
		FROM message_history mh
		JOIN channels c ON c.id = mh.channel_id
		WHERE mh.user_id = ? AND mh.source_type = ?
			AND c.user_id = ? AND c.source_type = ? AND c.type = ?
		GROUP BY c.id
		ORDER BY message_count DESC
		LIMIT ?
	`, userID, sourceType, userID, sourceType, source.ChannelTypeSender, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top contacts for user: %w", err)
	}
	defer rows.Close()

	var contacts []TopContactStats
	for rows.Next() {
		var c TopContactStats
		if err := rows.Scan(&c.ChannelID, &c.Identifier, &c.Name, &c.Type, &c.MessageCount, &c.IsTracked); err != nil {
			return nil, fmt.Errorf("failed to scan top contact: %w", err)
		}
		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}
