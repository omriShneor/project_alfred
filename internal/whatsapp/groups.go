package whatsapp

import (
	"context"
	"fmt"
)

// DiscoverableChannel represents a WhatsApp contact that can be tracked
type DiscoverableChannel struct {
	Type       string `json:"type"`       // "sender" (contacts only)
	Identifier string `json:"identifier"` // phone number
	Name       string `json:"name"`       // display name
	IsTracked  bool   `json:"is_tracked"` // whether this channel is already tracked
	ChannelID  *int64 `json:"channel_id,omitempty"` // ID if tracked
	Enabled    *bool  `json:"enabled,omitempty"`    // enabled status if tracked
}

// GetDiscoverableChannels returns all contacts as discoverable channels (no groups)
func (c *Client) GetDiscoverableChannels() ([]DiscoverableChannel, error) {
	var channels []DiscoverableChannel

	// Get all contacts (groups not supported)
	contacts, err := c.WAClient.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	for jid, contact := range contacts {
		// Only return individual contacts (not groups)
		if jid.Server != "s.whatsapp.net" {
			continue
		}

		name := contact.FullName
		if name == "" {
			name = contact.PushName
		}
		if name == "" {
			name = jid.User
		}

		channels = append(channels, DiscoverableChannel{
			Type:       "sender",
			Identifier: jid.User,
			Name:       name,
			IsTracked:  false, // Will be enriched by handler
		})
	}

	return channels, nil
}
