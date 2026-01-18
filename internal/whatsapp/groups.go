package whatsapp

import (
	"context"
	"fmt"
)

// DiscoverableChannel represents a WhatsApp contact or group that can be tracked
type DiscoverableChannel struct {
	Type       string `json:"type"`        // "sender" or "group"
	Identifier string `json:"identifier"`  // phone number for senders, JID for groups
	Name       string `json:"name"`        // display name
	IsTracked  bool   `json:"is_tracked"`  // whether this channel is already tracked
	ChannelID  *int64 `json:"channel_id,omitempty"` // ID if tracked
	Enabled    *bool  `json:"enabled,omitempty"`    // enabled status if tracked
}

// GetDiscoverableChannels returns all contacts and groups as discoverable channels
func (c *Client) GetDiscoverableChannels() ([]DiscoverableChannel, error) {
	var channels []DiscoverableChannel

	// Get all groups
	groups, err := c.WAClient.GetJoinedGroups(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	for _, group := range groups {
		channels = append(channels, DiscoverableChannel{
			Type:       "group",
			Identifier: group.JID.String(),
			Name:       group.Name,
			IsTracked:  false, // Will be enriched by handler
		})
	}

	// Get all contacts
	contacts, err := c.WAClient.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	for jid, contact := range contacts {
		// Skip groups, only return individual contacts
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
