package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Required
	AnthropicAPIKey       string
	GoogleCredentialsFile string

	// Optional with defaults
	DBPath           string
	HTTPPort         int
	DebugAllMessages bool
	ClaudeModel      string
}

func LoadFromEnv() *Config {
	cfg := &Config{
		// Required
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		GoogleCredentialsFile: getEnvOrDefault("GOOGLE_CREDENTIALS_FILE", "./credentials.json"),

		// Optional with defaults
		DBPath:           getEnvOrDefault("ALFRED_DB_PATH", "./alfred.db"),
		HTTPPort:         getEnvAsIntOrDefault("ALFRED_HTTP_PORT", 8080),
		DebugAllMessages: getEnvAsBoolOrDefault("ALFRED_DEBUG_ALL_MESSAGES", false),
		ClaudeModel:      getEnvOrDefault("ALFRED_CLAUDE_MODEL", "claude-sonnet-4-20250514"),
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
