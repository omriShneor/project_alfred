package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
)

// DiscoverableChannel represents a Telegram chat that can be tracked
type DiscoverableChannel struct {
	Type       string `json:"type"`       // "contact", "group", "channel"
	Identifier string `json:"identifier"` // Telegram chat ID
	Name       string `json:"name"`       // Display name
	IsTracked  bool   `json:"is_tracked"` // Whether currently tracked
	ChannelID  *int64 `json:"channel_id"` // DB ID if tracked
}

// GetDiscoverableChannels returns all available chats that can be tracked
func (c *Client) GetDiscoverableChannels(ctx context.Context, db *database.DB) ([]DiscoverableChannel, error) {
	c.mu.RLock()
	api := c.api
	c.mu.RUnlock()

	if api == nil {
		return nil, fmt.Errorf("client not connected")
	}

	var channels []DiscoverableChannel

	// Get contacts
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

				// Check if tracked
				tracked, channelID, _, _ := db.IsSourceChannelTracked(source.SourceTypeTelegram, identifier)

				ch := DiscoverableChannel{
					Type:       "contact",
					Identifier: identifier,
					Name:       name,
					IsTracked:  tracked,
				}
				if tracked {
					ch.ChannelID = &channelID
				}
				channels = append(channels, ch)
			}
		}
	}

	// Get dialogs (chats, groups, channels)
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit:      100,
		OffsetPeer: &tg.InputPeerEmpty{},
	})
	if err != nil {
		// Log but don't fail - we still have contacts
		fmt.Printf("Warning: Failed to get dialogs: %v\n", err)
	} else {
		switch d := dialogs.(type) {
		case *tg.MessagesDialogs:
			channels = append(channels, extractChatsFromDialogs(d.Chats, db)...)
		case *tg.MessagesDialogsSlice:
			channels = append(channels, extractChatsFromDialogs(d.Chats, db)...)
		}
	}

	return channels, nil
}

// extractChatsFromDialogs extracts groups and channels from dialog response
func extractChatsFromDialogs(chats []tg.ChatClass, db *database.DB) []DiscoverableChannel {
	var channels []DiscoverableChannel

	for _, chat := range chats {
		var identifier, name, chatType string

		switch c := chat.(type) {
		case *tg.Chat:
			if c.Deactivated || c.Left {
				continue
			}
			identifier = fmt.Sprintf("chat_%d", c.ID)
			name = c.Title
			chatType = "group"
		case *tg.Channel:
			if c.Left {
				continue
			}
			identifier = fmt.Sprintf("channel_%d", c.ID)
			name = c.Title
			if c.Broadcast {
				chatType = "channel"
			} else {
				chatType = "group" // Supergroups are treated as groups
			}
		default:
			continue
		}

		// Check if tracked
		tracked, channelID, _, _ := db.IsSourceChannelTracked(source.SourceTypeTelegram, identifier)

		ch := DiscoverableChannel{
			Type:       chatType,
			Identifier: identifier,
			Name:       name,
			IsTracked:  tracked,
		}
		if tracked {
			ch.ChannelID = &channelID
		}
		channels = append(channels, ch)
	}

	return channels
}
