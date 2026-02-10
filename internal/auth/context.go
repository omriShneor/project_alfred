package auth

import (
	"context"
	"fmt"
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
	Timezone  string `json:"timezone,omitempty"`
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
func GetUserID(ctx context.Context) (int64, error) {
	user := GetUserFromContext(ctx)
	if user == nil {
		return -1, fmt.Errorf("Could not extract user from request context")
	}
	return user.ID, nil
}

// SetUserInContext returns a new context with the user set
func SetUserInContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}
