package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration settings
type Config struct {
	// Server configuration
	ServerPort string

	// Mizito API configuration
	MizitoBaseURL    string
	MizitoLoginURL   string
	MizitoChatAPIURL string
	MizitoUsername   string
	MizitoPassword   string
	MizitoLoginCode  string
	MizitoRegID      string
	MizitoDialogID   string
	MizitoFromUserID string

	// JWT token configuration
	JWTTokenFile string

	// App token for API authentication (optional but recommended)
	AppToken string

	// Logging configuration
	LogLevel string
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		ServerPort:       ":8080",
		MizitoBaseURL:    "https://app.mizito.ir",
		MizitoLoginURL:   "https://app.mizito.ir/capi/session/create",
		MizitoChatAPIURL: "https://app.mizito.ir/api/chat/send",
		JWTTokenFile:     "token.json",
		LogLevel:         "info",
		MizitoLoginCode:  "null",
		MizitoRegID:      "null",
	}
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found or error loading it: %v", err)
	}

	config := DefaultConfig()

	// Server configuration
	if port := os.Getenv("SERVER_PORT"); port != "" {
		config.ServerPort = port
	}

	// Mizito configuration
	if baseURL := os.Getenv("MIZITO_BASE_URL"); baseURL != "" {
		config.MizitoBaseURL = baseURL
	}

	if loginURL := os.Getenv("MIZITO_LOGIN_URL"); loginURL != "" {
		config.MizitoLoginURL = loginURL
	}

	if chatURL := os.Getenv("MIZITO_CHAT_API_URL"); chatURL != "" {
		config.MizitoChatAPIURL = chatURL
	}

	if username := os.Getenv("MIZITO_USERNAME"); username != "" {
		config.MizitoUsername = username
	}

	if password := os.Getenv("MIZITO_PASSWORD"); password != "" {
		config.MizitoPassword = password
	}

	if loginCode := os.Getenv("MIZITO_LOGIN_CODE"); loginCode != "" {
		config.MizitoLoginCode = loginCode
	}

	if regID := os.Getenv("MIZITO_REG_ID"); regID != "" {
		config.MizitoRegID = regID
	}

	if dialogID := os.Getenv("MIZITO_DIALOG_ID"); dialogID != "" {
		config.MizitoDialogID = dialogID
	}

	if fromUserID := os.Getenv("MIZITO_FROM_USER_ID"); fromUserID != "" {
		config.MizitoFromUserID = fromUserID
	}

	// JWT configuration
	if tokenFile := os.Getenv("JWT_TOKEN_FILE"); tokenFile != "" {
		config.JWTTokenFile = tokenFile
	}

	// App token for API authentication
	if appToken := os.Getenv("APP_TOKEN"); appToken != "" {
		config.AppToken = appToken
	}

	// Logging configuration
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = strings.ToLower(logLevel)
	}

	// Validate required configuration
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// validate checks if required configuration values are present
func (c *Config) validate() error {
	if c.MizitoUsername == "" {
		return ConfigError("MIZITO_USERNAME is required")
	}

	if c.MizitoPassword == "" {
		return ConfigError("MIZITO_PASSWORD is required")
	}

	if c.MizitoDialogID == "" {
		return ConfigError("MIZITO_DIALOG_ID is required")
	}

	if c.MizitoFromUserID == "" {
		return ConfigError("MIZITO_FROM_USER_ID is required")
	}

	return nil
}

// ConfigError is a custom error type for configuration errors
type ConfigError string

func (e ConfigError) Error() string {
	return string(e)
}

// GetLogLevel returns the log level for logging
func (c *Config) GetLogLevel() string {
	return c.LogLevel
}
