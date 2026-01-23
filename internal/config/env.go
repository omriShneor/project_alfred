package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if it exists (silently ignore if not found)
	_ = godotenv.Load()
}

type Config struct {
	// Required
	AnthropicAPIKey       string
	GoogleCredentialsFile string
	GoogleTokenFile       string

	// Optional with defaults
	DBPath             string
	HTTPPort           int
	DebugAllMessages   bool
	ClaudeModel        string
	ClaudeTemperature  float64
	MessageHistorySize int
	DevMode            bool
}

func LoadFromEnv() *Config {
	cfg := &Config{
		// Required
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		GoogleCredentialsFile: getEnvOrDefault("GOOGLE_CREDENTIALS_FILE", "./credentials.json"),
		GoogleTokenFile:       getEnvOrDefault("GOOGLE_TOKEN_FILE", "./token.json"),

		// Optional with defaults
		DBPath:             getEnvOrDefault("ALFRED_DB_PATH", "./alfred.db"),
		HTTPPort:           getEnvAsIntOrDefault("ALFRED_HTTP_PORT", 8080),
		DebugAllMessages:   getEnvAsBoolOrDefault("ALFRED_DEBUG_ALL_MESSAGES", false),
		ClaudeModel:        getEnvOrDefault("ALFRED_CLAUDE_MODEL", "claude-sonnet-4-20250514"),
		ClaudeTemperature:  getEnvAsFloatOrDefault("ALFRED_CLAUDE_TEMPERATURE", 0.1),
		MessageHistorySize: getEnvAsIntOrDefault("ALFRED_MESSAGE_HISTORY_SIZE", 25),
		DevMode:            getEnvAsBoolOrDefault("ALFRED_DEV_MODE", false),
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
