package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
)

// DiscoverableChannel represents a Telegram contact that can be tracked
type DiscoverableChannel struct {
	Type           string `json:"type"`            // "contact" (contacts only)
	Identifier     string `json:"identifier"`      // Telegram user ID
	Name           string `json:"name"`            // Display name
	SecondaryLabel string `json:"secondary_label"` // Pre-formatted: "@username" or ""
	IsTracked      bool   `json:"is_tracked"`      // Whether currently tracked
	ChannelID      *int64 `json:"channel_id"`      // DB ID if tracked
}

// GetDiscoverableChannels returns all contacts that can be tracked (no groups/channels)
func (c *Client) GetDiscoverableChannels(ctx context.Context, db *database.DB) ([]DiscoverableChannel, error) {
	c.mu.RLock()
	api := c.api
	c.mu.RUnlock()

	if api == nil {
		return nil, fmt.Errorf("client not connected")
	}

	var channels []DiscoverableChannel

	// Get contacts only (groups/channels not supported)
	contacts, err := api.ContactsGetContacts(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	if contactsResult, ok := contacts.(*tg.ContactsContacts); ok {
		for _, user := range contactsResult.Users {
			if u, ok := user.(*tg.User); ok {
				if u.Bot || u.Self {
					continue
				}

				identifier := fmt.Sprintf("%d", u.ID)
				name := getUserName(u)

				// Format username as secondary label
				secondaryLabel := ""
				if u.Username != "" {
					secondaryLabel = "@" + u.Username
				}

				// Check if tracked
				tracked, channelID, _, _ := db.IsSourceChannelTracked(source.SourceTypeTelegram, identifier)

				ch := DiscoverableChannel{
					Type:           "contact",
					Identifier:     identifier,
					Name:           name,
					SecondaryLabel: secondaryLabel,
					IsTracked:      tracked,
				}
				if tracked {
					ch.ChannelID = &channelID
				}
				channels = append(channels, ch)
			}
		}
	}

	return channels, nil
}
