package processor

import (
	"fmt"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
)

// storeSourceMessage saves a message to the message history with source type
func (p *Processor) storeSourceMessage(msg source.Message) (*database.SourceMessage, error) {
	record, err := p.db.StoreSourceMessage(
		msg.SourceType,
		msg.SourceID,
		msg.SenderID,
		msg.SenderName,
		msg.Text,
		msg.Subject,
		msg.Timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store message: %w", err)
	}
	return record, nil
}
