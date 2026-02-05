package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// getEncryptionKey derives a 32-byte key for AES-256 encryption
func getEncryptionKey() ([]byte, error) {
	// Try ALFRED_ENCRYPTION_KEY first
	if envKey := os.Getenv("ALFRED_ENCRYPTION_KEY"); envKey != "" {
		hash := sha256.Sum256([]byte(envKey))
		return hash[:], nil
	}

	// Fall back to deriving from ANTHROPIC_API_KEY
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		hash := sha256.Sum256([]byte("alfred-encryption-" + apiKey))
		return hash[:], nil
	}

	return nil, fmt.Errorf("no encryption key available: set ALFRED_ENCRYPTION_KEY or ANTHROPIC_API_KEY")
}

// encryptToken encrypts an OAuth token for storage
func encryptToken(token string) ([]byte, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)
	return ciphertext, nil
}

// decryptToken decrypts an OAuth token from storage
func decryptToken(ciphertext []byte) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GetGoogleToken retrieves the OAuth2 token for a user
func (d *DB) GetGoogleToken(userID int64) (*oauth2.Token, error) {
	var accessTokenEnc, refreshTokenEnc []byte
	var tokenType string
	var expiry sql.NullTime
	var scopes sql.NullString

	err := d.QueryRow(`
		SELECT access_token_encrypted, refresh_token_encrypted, token_type, expiry, scopes
		FROM google_tokens WHERE user_id = ?
	`, userID).Scan(&accessTokenEnc, &refreshTokenEnc, &tokenType, &expiry, &scopes)

	if err == sql.ErrNoRows {
		return nil, nil // No token stored
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get google token: %w", err)
	}

	// Decrypt tokens
	accessToken, err := decryptToken(accessTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}

	refreshToken, err := decryptToken(refreshTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
	}

	if expiry.Valid {
		token.Expiry = expiry.Time
	}

	return token, nil
}

// SaveGoogleToken stores an OAuth2 token for a user (upsert)
func (d *DB) SaveGoogleToken(userID int64, token *oauth2.Token, email string, scopes []string) error {
	// Encrypt tokens
	accessTokenEnc, err := encryptToken(token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	refreshTokenEnc, err := encryptToken(token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	var expiry *time.Time
	if !token.Expiry.IsZero() {
		expiry = &token.Expiry
	}

	// Marshal scopes to JSON
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	// Upsert token
	_, err = d.Exec(`
		INSERT INTO google_tokens (user_id, access_token_encrypted, refresh_token_encrypted, token_type, expiry, scopes, email, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			access_token_encrypted = excluded.access_token_encrypted,
			refresh_token_encrypted = excluded.refresh_token_encrypted,
			token_type = excluded.token_type,
			expiry = excluded.expiry,
			scopes = excluded.scopes,
			email = excluded.email,
			updated_at = CURRENT_TIMESTAMP
	`, userID, accessTokenEnc, refreshTokenEnc, token.TokenType, expiry, scopesJSON, email)

	if err != nil {
		return fmt.Errorf("failed to save google token: %w", err)
	}

	return nil
}

// DeleteGoogleToken removes the OAuth2 token for a user
func (d *DB) DeleteGoogleToken(userID int64) error {
	_, err := d.Exec(`DELETE FROM google_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete google token: %w", err)
	}
	return nil
}

// RemoveGoogleTokenScope removes a specific scope from a user's token
// If removing the scope results in only profile scopes remaining, the entire token is deleted
func (d *DB) RemoveGoogleTokenScope(userID int64, scopeToRemove string) error {
	// Get current token info
	tokenInfo, err := d.GetGoogleTokenInfo(userID)
	if err != nil {
		return fmt.Errorf("failed to get token info: %w", err)
	}

	if !tokenInfo.HasToken {
		return nil // No token to remove scope from
	}

	// Map scope names to full URLs
	scopeURLs := map[string]string{
		"gmail":    "https://www.googleapis.com/auth/gmail.readonly",
		"calendar": "https://www.googleapis.com/auth/calendar",
		"profile": "openid https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile",
	}

	scopeURL, ok := scopeURLs[scopeToRemove]
	if !ok {
		return fmt.Errorf("unknown scope: %s", scopeToRemove)
	}

	// Filter out the scope to remove
	var newScopes []string
	scopeURLsToRemove := make(map[string]bool)
	for _, url := range strings.Fields(scopeURL) {
		scopeURLsToRemove[url] = true
	}

	for _, scope := range tokenInfo.Scopes {
		if !scopeURLsToRemove[scope] {
			newScopes = append(newScopes, scope)
		}
	}

	// Check if only profile scopes remain
	profileScopes := []string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}

	hasNonProfileScope := false
	for _, scope := range newScopes {
		isProfileScope := false
		for _, ps := range profileScopes {
			if scope == ps {
				isProfileScope = true
				break
			}
		}
		if !isProfileScope {
			hasNonProfileScope = true
			break
		}
	}

	// If only profile scopes remain, delete the entire token
	if !hasNonProfileScope {
		return d.DeleteGoogleToken(userID)
	}

	// Otherwise, update the scopes
	scopesJSON, err := json.Marshal(newScopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	_, err = d.Exec(`
		UPDATE google_tokens SET scopes = ? WHERE user_id = ?
	`, string(scopesJSON), userID)
	if err != nil {
		return fmt.Errorf("failed to update scopes: %w", err)
	}

	return nil
}

// UpdateGoogleToken updates just the access token and expiry (used after token refresh)
func (d *DB) UpdateGoogleToken(userID int64, token *oauth2.Token) error {
	accessTokenEnc, err := encryptToken(token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	var expiry *time.Time
	if !token.Expiry.IsZero() {
		expiry = &token.Expiry
	}

	// Also update refresh token if it changed
	refreshTokenEnc, err := encryptToken(token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	_, err = d.Exec(`
		UPDATE google_tokens SET
			access_token_encrypted = ?,
			refresh_token_encrypted = ?,
			expiry = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, accessTokenEnc, refreshTokenEnc, expiry, userID)

	if err != nil {
		return fmt.Errorf("failed to update google token: %w", err)
	}

	return nil
}

// GoogleTokenInfo represents token metadata without the actual token values
type GoogleTokenInfo struct {
	UserID    int64
	Email     string
	HasToken  bool
	ExpiresAt *time.Time
	Scopes    []string
}

// GetGoogleTokenInfo retrieves token metadata without exposing the actual tokens
func (d *DB) GetGoogleTokenInfo(userID int64) (*GoogleTokenInfo, error) {
	var email sql.NullString
	var expiry sql.NullTime
	var scopes sql.NullString

	err := d.QueryRow(`
		SELECT email, expiry, scopes
		FROM google_tokens WHERE user_id = ?
	`, userID).Scan(&email, &expiry, &scopes)

	if err == sql.ErrNoRows {
		return &GoogleTokenInfo{UserID: userID, HasToken: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get google token info: %w", err)
	}

	info := &GoogleTokenInfo{
		UserID:   userID,
		Email:    email.String,
		HasToken: true,
	}

	if expiry.Valid {
		info.ExpiresAt = &expiry.Time
	}

	if scopes.Valid && scopes.String != "" {
		info.Scopes = splitScopes(scopes.String)
	}

	return info, nil
}

// splitScopes parses a JSON array of scopes into a slice
func splitScopes(scopeStr string) []string {
	if scopeStr == "" {
		return nil
	}

	// Try to parse as JSON array first (current format)
	var scopes []string
	if err := json.Unmarshal([]byte(scopeStr), &scopes); err == nil {
		return scopes
	}

	// Fallback to space-separated for backward compatibility
	return strings.Fields(scopeStr)
}

// ListUsersWithGoogleToken returns user IDs that have stored Google tokens
func (d *DB) ListUsersWithGoogleToken() ([]int64, error) {
	rows, err := d.Query(`SELECT user_id FROM google_tokens`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users with google token: %w", err)
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

// TokenJSON is used for JSON serialization of oauth2.Token (for debugging/export)
type TokenJSON struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// TokenToJSON converts an oauth2.Token to JSON bytes (for debugging)
func TokenToJSON(token *oauth2.Token) ([]byte, error) {
	tj := TokenJSON{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}
	return json.Marshal(tj)
}

// TokenFromJSON parses JSON bytes to an oauth2.Token
func TokenFromJSON(data []byte) (*oauth2.Token, error) {
	var tj TokenJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, err
	}
	return &oauth2.Token{
		AccessToken:  tj.AccessToken,
		RefreshToken: tj.RefreshToken,
		TokenType:    tj.TokenType,
		Expiry:       tj.Expiry,
	}, nil
}
