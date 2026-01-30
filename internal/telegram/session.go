package telegram

import (
	"context"
	"encoding/json"
	"os"

	"github.com/gotd/td/session"
)

// FileSessionStorage implements session.Storage for file-based persistence
type FileSessionStorage struct {
	Path string
}

// LoadSession loads the session from file
func (s *FileSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

// StoreSession saves the session to file
func (s *FileSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	return os.WriteFile(s.Path, data, 0600)
}

// JSONSessionStorage wraps session data for JSON storage
type JSONSessionStorage struct {
	Path string
	data []byte
}

// LoadSession loads session from JSON file
func (s *JSONSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var stored struct {
		Session []byte `json:"session"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		// Try raw session data
		return data, nil
	}
	return stored.Session, nil
}

// StoreSession saves session to JSON file
func (s *JSONSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	stored := struct {
		Session []byte `json:"session"`
	}{Session: data}

	jsonData, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, jsonData, 0600)
}
