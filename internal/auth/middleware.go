package auth

import (
	"net/http"
	"strings"
)

// Middleware provides HTTP middleware for authentication
type Middleware struct {
	service *Service
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(service *Service) *Middleware {
	return &Middleware{
		service: service,
	}
}

// RequireAuth is middleware that requires a valid session token
// The user is extracted from the token and added to the request context
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, `{"error": "missing authorization token"}`, http.StatusUnauthorized)
			return
		}

		// Validate token and get user
		user, err := m.service.ValidateSession(token)
		if err != nil {
			http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := SetUserInContext(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is middleware that validates a token if present, but doesn't require it
// Useful for endpoints that behave differently for authenticated vs anonymous users
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token != "" {
			user, err := m.service.ValidateSession(token)
			if err == nil {
				ctx := SetUserInContext(r.Context(), user)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// extractBearerToken extracts the token from the Authorization header
// Expects format: "Bearer <token>"
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
