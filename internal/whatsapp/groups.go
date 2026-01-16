package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
)

type GroupInfo struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

type ContactInfo struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

// GetGroups returns all WhatsApp groups (limit parameter kept for API compatibility)
// Note: whatsmeow doesn't provide sorting by recent activity
func (c *Client) GetGroups(limit int) ([]GroupInfo, error) {
	groups, err := c.WAClient.GetJoinedGroups(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	result := make([]GroupInfo, 0, len(groups))
	for _, group := range groups {
		result = append(result, GroupInfo{
			JID:  group.JID.String(),
			Name: group.Name,
		})
	}

	// Apply limit if specified and smaller than total
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// GetRecentContacts returns contacts from the contact list
// Note: whatsmeow doesn't provide sorting by recent activity
func (c *Client) GetRecentContacts(limit int) ([]ContactInfo, error) {
	contacts, err := c.WAClient.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	result := make([]ContactInfo, 0, len(contacts))
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
		result = append(result, ContactInfo{
			Phone: jid.User,
			Name:  name,
		})
	}

	// Apply limit if specified and smaller than total
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// ListGroups prints all groups to stdout (legacy function for CLI)
func ListGroups(client *whatsmeow.Client) error {
	groups, err := client.GetJoinedGroups(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	if len(groups) == 0 {
		fmt.Println("No groups found.")
		return nil
	}

	fmt.Println("\n=== Your WhatsApp Groups ===")
	fmt.Println("Copy the JID to your config.yaml file:\n")

	for _, group := range groups {
		fmt.Printf("  Name: %s\n", group.Name)
		fmt.Printf("  JID:  %s\n\n", group.JID.String())
	}

	fmt.Printf("Total: %d groups\n", len(groups))
	return nil
}
