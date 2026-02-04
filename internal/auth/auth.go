package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	goauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

const (
	// SessionDuration is how long session tokens are valid
	SessionDuration = 30 * 24 * time.Hour // 30 days
)

// ProfileScopes - minimum scopes for login (user identity only)
var ProfileScopes = []string{
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

// GmailScopes - for email scanning (requested separately)
var GmailScopes = []string{
	gmail.GmailReadonlyScope,
}

// CalendarScopes - for calendar sync (requested separately)
var CalendarScopes = []string{
	calendar.CalendarScope,
}

// OAuthScopes - all scopes combined (for backward compatibility with tests)
var OAuthScopes = append(append(ProfileScopes, GmailScopes...), CalendarScopes...)

// Service handles authentication operations
type Service struct {
	db        *sql.DB
	config    *oauth2.Config
	encryptor *Encryptor
}

// NewService creates a new authentication service
func NewService(db *sql.DB, oauthConfig *oauth2.Config) (*Service, error) {
	encryptor, err := NewEncryptor(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &Service{
		db:        db,
		config:    oauthConfig,
		encryptor: encryptor,
	}, nil
}

// GetAuthURL returns the Google OAuth authorization URL (with all scopes - for backward compatibility)
func (s *Service) GetAuthURL(state string) string {
	return s.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// GetAuthURLWithScopes generates OAuth URL for specific scopes
// If includeGranted is true, adds include_granted_scopes=true for incremental auth
func (s *Service) GetAuthURLWithScopes(scopes []string, state string, includeGranted bool) string {
	// Create a modified config with the specific scopes
	modifiedConfig := &oauth2.Config{
		ClientID:     s.config.ClientID,
		ClientSecret: s.config.ClientSecret,
		Endpoint:     s.config.Endpoint,
		RedirectURL:  s.config.RedirectURL,
		Scopes:       scopes,
	}

	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	}

	if includeGranted {
		// Add include_granted_scopes for incremental authorization
		opts = append(opts, oauth2.SetAuthURLParam("include_granted_scopes", "true"))
	}

	return modifiedConfig.AuthCodeURL(state, opts...)
}

// GetUserScopes returns the scopes a user has granted
// If no scopes are stored (legacy users), returns all scopes for backward compatibility
func (s *Service) GetUserScopes(userID int64) ([]string, error) {
	var scopesJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT scopes FROM google_tokens WHERE user_id = ?
	`, userID).Scan(&scopesJSON)
	if err == sql.ErrNoRows {
		return nil, nil // No token stored
	}
	if err != nil {
		return nil, err
	}

	// If scopes is empty/null, assume all scopes (backward compatibility)
	if !scopesJSON.Valid || scopesJSON.String == "" || scopesJSON.String == "null" {
		return nil, fmt.Errorf("Google scopes are empty/nil")
	}

	var scopes []string
	if err := json.Unmarshal([]byte(scopesJSON.String), &scopes); err != nil {
		// If parsing fails, assume all scopes
		return nil, err
	}

	return scopes, nil
}

// HasScope checks if a user has granted a specific scope
func (s *Service) HasScope(userID int64, scope string) (bool, error) {
	scopes, err := s.GetUserScopes(userID)
	if err != nil {
		return false, err
	}

	for _, s := range scopes {
		if s == scope {
			return true, nil
		}
	}
	return false, nil
}

// HasGmailScope checks if user has Gmail read scope
func (s *Service) HasGmailScope(userID int64) (bool, error) {
	return s.HasScope(userID, gmail.GmailReadonlyScope)
}

// HasCalendarScope checks if user has Calendar scope
func (s *Service) HasCalendarScope(userID int64) (bool, error) {
	return s.HasScope(userID, calendar.CalendarScope)
}

// mergeScopes combines two scope slices, removing duplicates
func mergeScopes(existing, newScopes []string) []string {
	scopeSet := make(map[string]bool)
	for _, s := range existing {
		scopeSet[s] = true
	}
	for _, s := range newScopes {
		scopeSet[s] = true
	}

	result := make([]string, 0, len(scopeSet))
	for s := range scopeSet {
		result = append(result, s)
	}
	return result
}

// ExchangeCodeAndAddScopes exchanges an OAuth code and merges new scopes with existing token
// This is used for incremental authorization
func (s *Service) ExchangeCodeAndAddScopes(ctx context.Context, userID int64, code string, newScopes []string) error {
	// Exchange code for new token
	token, err := s.config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get existing scopes
	existingScopes, err := s.GetUserScopes(userID)
	if err != nil {
		return err
	}

	// Merge scopes
	mergedScopes := mergeScopes(existingScopes, newScopes)

	// Store updated token with merged scopes
	if err := s.storeGoogleTokenWithScopes(userID, token, mergedScopes); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// GetOAuthConfig returns the OAuth config for use by other packages
func (s *Service) GetOAuthConfig() *oauth2.Config {
	return s.config
}

// ExchangeCodeAndLogin exchanges an OAuth code for tokens and creates/updates the user
// Returns the user and a session token
// If redirectURI is provided, it will be used for the token exchange (must match the one used to generate the auth URL)
func (s *Service) ExchangeCodeAndLogin(ctx context.Context, code string, deviceInfo string, redirectURI string) (*User, string, error) {
	// Exchange code for token
	// If a custom redirect URI was used for the auth URL, we need to use the same one for the exchange
	var token *oauth2.Token
	var err error
	if redirectURI != "" {
		// Create a temporary config with the custom redirect URI
		tempConfig := &oauth2.Config{
			ClientID:     s.config.ClientID,
			ClientSecret: s.config.ClientSecret,
			Endpoint:     s.config.Endpoint,
			RedirectURL:  redirectURI,
			Scopes:       s.config.Scopes,
		}
		token, err = tempConfig.Exchange(ctx, code)
	} else {
		token, err = s.config.Exchange(ctx, code)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from Google
	googleUser, err := s.getGoogleUserInfo(ctx, token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user info: %w", err)
	}

	// Create or update user in database
	user, err := s.upsertUser(googleUser)
	if err != nil {
		return nil, "", fmt.Errorf("failed to upsert user: %w", err)
	}

	// Store Google tokens
	if err := s.storeGoogleToken(user.ID, token); err != nil {
		return nil, "", fmt.Errorf("failed to store token: %w", err)
	}

	// Create session
	sessionToken, err := s.createSession(user.ID, deviceInfo)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	// Initialize user settings if new
	if err := s.initializeUserSettings(user.ID); err != nil {
		// Log but don't fail - settings can be created later
		fmt.Printf("Warning: failed to initialize user settings: %v\n", err)
	}

	return user, sessionToken, nil
}

// getGoogleUserInfo fetches user profile from Google
func (s *Service) getGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*goauth2.Userinfo, error) {
	client := s.config.Client(ctx, token)
	oauth2Service, err := goauth2.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	userInfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

// upsertUser creates or updates a user based on Google ID
func (s *Service) upsertUser(googleUser *goauth2.Userinfo) (*User, error) {
	now := time.Now()

	// Try to find existing user
	var user User
	err := s.db.QueryRow(`
		SELECT id, google_id, email, name, avatar_url
		FROM users WHERE google_id = ?
	`, googleUser.Id).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL)

	if err == sql.ErrNoRows {
		// Create new user
		result, err := s.db.Exec(`
			INSERT INTO users (google_id, email, name, avatar_url, created_at, updated_at, last_login_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, googleUser.Id, googleUser.Email, googleUser.Name, googleUser.Picture, now, now, now)
		if err != nil {
			return nil, err
		}

		userID, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}

		return &User{
			ID:        userID,
			GoogleID:  googleUser.Id,
			Email:     googleUser.Email,
			Name:      googleUser.Name,
			AvatarURL: googleUser.Picture,
		}, nil
	} else if err != nil {
		return nil, err
	}

	// Update existing user
	_, err = s.db.Exec(`
		UPDATE users SET email = ?, name = ?, avatar_url = ?, updated_at = ?, last_login_at = ?
		WHERE id = ?
	`, googleUser.Email, googleUser.Name, googleUser.Picture, now, now, user.ID)
	if err != nil {
		return nil, err
	}

	user.Email = googleUser.Email
	user.Name = googleUser.Name
	user.AvatarURL = googleUser.Picture

	return &user, nil
}

// storeGoogleToken stores encrypted OAuth tokens for a user with ProfileScopes (login only)
func (s *Service) storeGoogleToken(userID int64, token *oauth2.Token) error {
	return s.storeGoogleTokenWithScopes(userID, token, ProfileScopes)
}

// storeGoogleTokenWithScopes stores encrypted OAuth tokens with specific scopes
func (s *Service) storeGoogleTokenWithScopes(userID int64, token *oauth2.Token, scopes []string) error {
	accessEncrypted, err := s.encryptor.Encrypt([]byte(token.AccessToken))
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	// Handle empty refresh token (incremental auth may not return a new refresh token)
	var refreshEncrypted []byte
	if token.RefreshToken != "" {
		refreshEncrypted, err = s.encryptor.Encrypt([]byte(token.RefreshToken))
		if err != nil {
			return fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
	}

	scopesJSON, _ := json.Marshal(scopes)

	if refreshEncrypted != nil {
		// Full token update (has refresh token)
		_, err = s.db.Exec(`
			INSERT INTO google_tokens (user_id, access_token_encrypted, refresh_token_encrypted, token_type, expiry, scopes, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(user_id) DO UPDATE SET
				access_token_encrypted = excluded.access_token_encrypted,
				refresh_token_encrypted = excluded.refresh_token_encrypted,
				token_type = excluded.token_type,
				expiry = excluded.expiry,
				scopes = excluded.scopes,
				updated_at = CURRENT_TIMESTAMP
		`, userID, accessEncrypted, refreshEncrypted, token.TokenType, token.Expiry, string(scopesJSON))
	} else {
		// Partial update (no new refresh token - keep existing)
		_, err = s.db.Exec(`
			UPDATE google_tokens SET
				access_token_encrypted = ?,
				token_type = ?,
				expiry = ?,
				scopes = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ?
		`, accessEncrypted, token.TokenType, token.Expiry, string(scopesJSON), userID)
	}

	return err
}

// GetGoogleToken retrieves and decrypts the Google OAuth token for a user
func (s *Service) GetGoogleToken(userID int64) (*oauth2.Token, error) {
	var accessEncrypted, refreshEncrypted []byte
	var tokenType string
	var expiry time.Time

	err := s.db.QueryRow(`
		SELECT access_token_encrypted, refresh_token_encrypted, token_type, expiry
		FROM google_tokens WHERE user_id = ?
	`, userID).Scan(&accessEncrypted, &refreshEncrypted, &tokenType, &expiry)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.encryptor.Decrypt(accessEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}

	refreshToken, err := s.encryptor.Decrypt(refreshEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  string(accessToken),
		RefreshToken: string(refreshToken),
		TokenType:    tokenType,
		Expiry:       expiry,
	}, nil
}

// createSession creates a new session for a user
func (s *Service) createSession(userID int64, deviceInfo string) (string, error) {
	// Generate random session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash the token for storage
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])

	expiresAt := time.Now().Add(SessionDuration)

	_, err := s.db.Exec(`
		INSERT INTO user_sessions (user_id, token_hash, expires_at, device_info)
		VALUES (?, ?, ?, ?)
	`, userID, tokenHash, expiresAt, deviceInfo)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateSession validates a session token and returns the user
func (s *Service) ValidateSession(token string) (*User, error) {
	// Hash the token
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])

	// Find session and user
	var user User
	var expiresAt time.Time

	err := s.db.QueryRow(`
		SELECT u.id, u.google_id, u.email, u.name, u.avatar_url, s.expires_at
		FROM user_sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token_hash = ?
	`, tokenHash).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid session")
	} else if err != nil {
		return nil, err
	}

	// Check expiration
	if time.Now().After(expiresAt) {
		// Delete expired session
		s.db.Exec(`DELETE FROM user_sessions WHERE token_hash = ?`, tokenHash)
		return nil, fmt.Errorf("session expired")
	}

	return &user, nil
}

// Logout invalidates a session token
func (s *Service) Logout(token string) error {
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.StdEncoding.EncodeToString(hash[:])

	_, err := s.db.Exec(`DELETE FROM user_sessions WHERE token_hash = ?`, tokenHash)
	return err
}

// GetUserByID retrieves a user by their ID
func (s *Service) GetUserByID(userID int64) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT id, google_id, email, name, avatar_url
		FROM users WHERE id = ?
	`, userID).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// initializeUserSettings creates default settings rows for a new user
func (s *Service) initializeUserSettings(userID int64) error {
	// Create feature_settings
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO feature_settings (user_id) VALUES (?)
	`, userID)
	if err != nil {
		return err
	}

	// Create notification preferences
	_, err = s.db.Exec(`
		INSERT OR IGNORE INTO user_notification_preferences (user_id) VALUES (?)
	`, userID)
	if err != nil {
		return err
	}

	// Create gmail_settings
	_, err = s.db.Exec(`
		INSERT OR IGNORE INTO gmail_settings (user_id) VALUES (?)
	`, userID)
	if err != nil {
		return err
	}

	// Create gcal_settings
	_, err = s.db.Exec(`
		INSERT OR IGNORE INTO gcal_settings (user_id) VALUES (?)
	`, userID)
	if err != nil {
		return err
	}

	return nil
}

// CleanupExpiredSessions removes all expired sessions
func (s *Service) CleanupExpiredSessions() error {
	_, err := s.db.Exec(`DELETE FROM user_sessions WHERE expires_at < ?`, time.Now())
	return err
}

// ListUsersWithGoogleToken returns user IDs that have stored Google tokens
func (s *Service) ListUsersWithGoogleToken() ([]int64, error) {
	rows, err := s.db.Query(`SELECT user_id FROM google_tokens`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, nil
}
