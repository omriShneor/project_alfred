package processor

import (
	"fmt"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/whatsapp"
)

// storeMessage saves a WhatsApp message to the message history
func (p *Processor) storeMessage(msg whatsapp.FilteredMessage) (*database.MessageRecord, error) {
	record, err := p.db.StoreMessage(
		msg.SourceID,
		msg.SenderJID,
		msg.SenderName,
		msg.Text,
		msg.Timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store message: %w", err)
	}
	return record, nil
}
