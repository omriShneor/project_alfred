package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

type Config struct {
	// Required
	AnthropicAPIKey       string
	GoogleCredentialsFile string
	GoogleCredentialsJSON string // JSON string (alternative to file)

	// Optional with defaults
	DBPath             string
	WhatsAppDBPath     string
	HTTPPort           int
	DebugAllMessages   bool
	ClaudeModel        string
	ClaudeTemperature  float64
	MessageHistorySize int
	DevMode            bool // Enables dev features like unauthenticated reset endpoint

	// Notification server config (API keys only - user prefs in database)
	ResendAPIKey string
	EmailFrom    string

	// Gmail integration config (enable/disable is in database settings, not here)
	GmailPollInterval int // minutes between polls
	GmailMaxEmails    int // max emails to process per poll

	// Telegram integration config
	TelegramAPIID   int    // API ID from my.telegram.org
	TelegramAPIHash string // API Hash from my.telegram.org
	TelegramDBPath  string // Session database path
}

func LoadFromEnv() *Config {
	cfg := &Config{
		// Required
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		GoogleCredentialsFile: getEnvOrDefault("GOOGLE_CREDENTIALS_FILE", "./credentials.json"),
		GoogleCredentialsJSON: os.Getenv("GOOGLE_CREDENTIALS_JSON"), // Takes precedence over file

		// Optional with defaults
		DBPath:             getEnvOrDefault("ALFRED_DB_PATH", "./alfred.db"),
		WhatsAppDBPath:     getEnvOrDefault("ALFRED_WHATSAPP_DB_PATH", "./whatsapp.db"),
		HTTPPort:           getEnvAsIntOrDefault("PORT", getEnvAsIntOrDefault("ALFRED_HTTP_PORT", 8080)),
		DebugAllMessages:   getEnvAsBoolOrDefault("ALFRED_DEBUG_ALL_MESSAGES", false),
		ClaudeModel:        getEnvOrDefault("ALFRED_CLAUDE_MODEL", "claude-sonnet-4-20250514"),
		ClaudeTemperature:  getEnvAsFloatOrDefault("ALFRED_CLAUDE_TEMPERATURE", 0.1),
		MessageHistorySize: getEnvAsIntOrDefault("ALFRED_MESSAGE_HISTORY_SIZE", 25),
		DevMode:            getEnvAsBoolOrDefault("ALFRED_DEV_MODE", false),

		// Notification server config (API keys only)
		ResendAPIKey: os.Getenv("ALFRED_RESEND_API_KEY"),
		EmailFrom:    getEnvOrDefault("ALFRED_EMAIL_FROM", "Alfred <onboarding@resend.dev>"),

		// Gmail integration config (enable/disable is in database settings)
		GmailPollInterval: getEnvAsIntOrDefault("ALFRED_GMAIL_POLL_INTERVAL", 1),
		GmailMaxEmails:    getEnvAsIntOrDefault("ALFRED_GMAIL_MAX_EMAILS", 10),

		// Telegram integration config
		TelegramAPIID:   getEnvAsIntOrDefault("ALFRED_TELEGRAM_API_ID", 0),
		TelegramAPIHash: os.Getenv("ALFRED_TELEGRAM_API_HASH"),
		TelegramDBPath:  getEnvOrDefault("ALFRED_TELEGRAM_DB_PATH", "./telegram.db"),
	}

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
