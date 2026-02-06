package database

import "fmt"

// BackfillStatus represents the state of initial source backfill.
type BackfillStatus string

const (
	BackfillStatusInProgress BackfillStatus = "in_progress"
	BackfillStatusCompleted  BackfillStatus = "completed"
	BackfillStatusFailed     BackfillStatus = "failed"
	BackfillStatusSkipped    BackfillStatus = "skipped"
)

func isTerminalBackfillStatus(status BackfillStatus) bool {
	switch status {
	case BackfillStatusCompleted, BackfillStatusFailed, BackfillStatusSkipped:
		return true
	default:
		return false
	}
}

// UpdateChannelInitialBackfillStatus updates the initial backfill status for a channel.
func (d *DB) UpdateChannelInitialBackfillStatus(userID, channelID int64, status BackfillStatus) error {
	query := `UPDATE channels SET initial_backfill_status = ?`
	args := []any{status}

	if isTerminalBackfillStatus(status) {
		query += `, initial_backfill_at = CURRENT_TIMESTAMP`
	}

	query += ` WHERE id = ? AND user_id = ?`
	args = append(args, channelID, userID)

	result, err := d.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update channel backfill status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update channel backfill status: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}
	return nil
}

// UpdateEmailSourceInitialBackfillStatus updates the initial backfill status for an email source.
func (d *DB) UpdateEmailSourceInitialBackfillStatus(userID, sourceID int64, status BackfillStatus) error {
	query := `UPDATE email_sources SET initial_backfill_status = ?`
	args := []any{status}

	if isTerminalBackfillStatus(status) {
		query += `, initial_backfill_at = CURRENT_TIMESTAMP`
	}

	query += `, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`
	args = append(args, sourceID, userID)

	result, err := d.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update email source backfill status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update email source backfill status: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("email source not found")
	}
	return nil
}
