package auth

import (
	"context"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserContextKey is the key used to store user info in request context
	UserContextKey contextKey = "user"
)

// User represents the authenticated user in request context
type User struct {
	ID        int64  `json:"id"`
	GoogleID  string `json:"google_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// GetUserFromContext extracts the authenticated user from the request context
func GetUserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// GetUserID extracts the user ID from the request context
// Returns 0 if no user is found
func GetUserID(ctx context.Context) int64 {
	user := GetUserFromContext(ctx)
	if user == nil {
		return 0
	}
	return user.ID
}

// SetUserInContext returns a new context with the user set
func SetUserInContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}
