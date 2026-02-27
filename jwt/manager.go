package jwt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ebrahimkhodadadi/MizitoForwarder/config"
	"github.com/ebrahimkhodadadi/MizitoForwarder/logger"
)

// TokenData represents the structure of the stored JWT token
type TokenData struct {
	Token        string    `json:"token"`
	LastLoginUID string    `json:"last_login_uid"`
	ExpiresAt    time.Time `json:"expires_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Manager handles JWT token storage and retrieval
type Manager struct {
	config    *config.Config
	tokenData *TokenData
	Mutex     sync.RWMutex
	logger    *logger.Logger
}

// NewManager creates a new JWT token manager
func NewManager(config *config.Config, logger *logger.Logger) *Manager {
	return &Manager{
		config: config,
		logger: logger,
	}
}

// LoadToken loads the JWT token from file
func (m *Manager) LoadToken() error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	file, err := ioutil.ReadFile(m.config.JWTTokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Info("No existing token file found")
			return nil
		}
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenData TokenData
	if err := json.Unmarshal(file, &tokenData); err != nil {
		return fmt.Errorf("failed to parse token data: %w", err)
	}

	m.tokenData = &tokenData
	m.logger.Info("JWT token loaded successfully",
		"expires_at", tokenData.ExpiresAt.Format(time.RFC3339),
		"updated_at", tokenData.UpdatedAt.Format(time.RFC3339))

	return nil
}

// SaveToken saves the JWT token to file
func (m *Manager) SaveToken(token, lastLoginUID string) error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	tokenData := &TokenData{
		Token:        token,
		LastLoginUID: lastLoginUID,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // Assuming 24 hour expiry
		UpdatedAt:    time.Now(),
	}

	file, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	// Ensure the parent directory exists (important when path is e.g. /app/data/token.json)
	if dir := filepath.Dir(m.config.JWTTokenFile); dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create token directory: %w", err)
		}
	}

	if err := ioutil.WriteFile(m.config.JWTTokenFile, file, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	m.tokenData = tokenData
	m.logger.Info("JWT token saved successfully",
		"expires_at", tokenData.ExpiresAt.Format(time.RFC3339))

	return nil
}

// GetToken returns the current JWT token
func (m *Manager) GetToken() (string, bool) {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	if m.tokenData == nil || m.tokenData.Token == "" {
		return "", false
	}

	return m.tokenData.Token, true
}

// GetTokenWithUID returns both token and last login UID
func (m *Manager) GetTokenWithUID() (string, string, bool) {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	if m.tokenData == nil || m.tokenData.Token == "" {
		return "", "", false
	}

	return m.tokenData.Token, m.tokenData.LastLoginUID, true
}

// IsTokenExpired checks if the current token is expired
func (m *Manager) IsTokenExpired() bool {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	if m.tokenData == nil {
		return true
	}

	return time.Now().After(m.tokenData.ExpiresAt)
}

// ClearToken removes the stored JWT token
func (m *Manager) ClearToken() error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if err := os.Remove(m.config.JWTTokenFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}

	m.tokenData = nil
	m.logger.Info("JWT token cleared successfully")

	return nil
}

// HasValidToken checks if there's a valid (non-expired) token
func (m *Manager) HasValidToken() bool {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	if m.tokenData == nil || m.tokenData.Token == "" {
		return false
	}

	return time.Now().Before(m.tokenData.ExpiresAt)
}
