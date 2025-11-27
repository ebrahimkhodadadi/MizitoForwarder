package mizito

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ebrahimkhodadadi/MizitoForwarder/config"
	"github.com/ebrahimkhodadadi/MizitoForwarder/jwt"
	"github.com/ebrahimkhodadadi/MizitoForwarder/logger"
)

// LoginRequest represents the login request structure
type LoginRequest struct {
	Username  string      `json:"username"`
	Password  string      `json:"password"`
	LoginCode interface{} `json:"loginCode"`
	RegID     interface{} `json:"regId"`
}

// LoginResponse represents the login response structure
type LoginResponse struct {
	Status       int    `json:"status"`
	Token        string `json:"token"`
	LastLoginUID string `json:"last_login_uid"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
}

// AuthService handles Mizito authentication
type AuthService struct {
	config *config.Config
	jwtMgr *jwt.Manager
	logger *logger.Logger
	client *http.Client
}

// NewAuthService creates a new authentication service
func NewAuthService(config *config.Config, jwtMgr *jwt.Manager, logger *logger.Logger) *AuthService {
	return &AuthService{
		config: config,
		jwtMgr: jwtMgr,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
	}
}

// Login performs authentication with Mizito API
func (a *AuthService) Login() error {
	a.logger.Info("Attempting to authenticate with Mizito API")

	// Prepare login request
	loginReq := LoginRequest{
		Username:  a.config.MizitoUsername,
		Password:  a.config.MizitoPassword,
		LoginCode: nil, // Will be converted to null in JSON
		RegID:     nil, // Will be converted to null in JSON
	}

	// Handle nullable fields
	if a.config.MizitoLoginCode != "" && a.config.MizitoLoginCode != "null" {
		loginReq.LoginCode = a.config.MizitoLoginCode
	}

	if a.config.MizitoRegID != "" && a.config.MizitoRegID != "null" {
		loginReq.RegID = a.config.MizitoRegID
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", a.config.MizitoLoginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://office.mizito.ir")
	req.Header.Set("Referer", "https://office.mizito.ir/")

	a.logger.Debug("Login request headers", "headers", req.Header)

	// Make request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	a.logger.Debug("Login response status", "status", resp.StatusCode)
	a.logger.Debug("Login response body", "body", string(body))

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login request failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Check response status
	if loginResp.Status != 1 {
		// Try to get error message
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("login failed with status %d: %s", loginResp.Status, errorResp.Message)
		}
		return fmt.Errorf("login failed with status: %d", loginResp.Status)
	}

	// Save token
	if err := a.jwtMgr.SaveToken(loginResp.Token, loginResp.LastLoginUID); err != nil {
		return fmt.Errorf("failed to save JWT token: %w", err)
	}

	a.logger.Info("Successfully authenticated with Mizito API")
	return nil
}

// EnsureValidToken ensures there's a valid JWT token, authenticating if needed
func (a *AuthService) EnsureValidToken() error {
	// Check if we have a valid token
	if a.jwtMgr.HasValidToken() {
		a.logger.Debug("JWT token is still valid")
		return nil
	}

	// Try to load existing token
	if err := a.jwtMgr.LoadToken(); err != nil {
		a.logger.Warn("Failed to load existing token", "error", err)
	}

	// Check again if token is now available and valid
	if a.jwtMgr.HasValidToken() {
		a.logger.Info("Loaded existing JWT token")
		return nil
	}

	// Need to authenticate
	a.logger.Info("No valid JWT token found, authenticating")
	return a.Login()
}

// RefreshToken refreshes the JWT token by authenticating again
func (a *AuthService) RefreshToken() error {
	a.logger.Info("Refreshing JWT token")
	
	// Clear existing token
	if err := a.jwtMgr.ClearToken(); err != nil {
		a.logger.Warn("Failed to clear existing token", "error", err)
	}

	// Authenticate again
	return a.Login()
}

// GetToken returns the current JWT token, ensuring it's valid first
func (a *AuthService) GetToken() (string, error) {
	if err := a.EnsureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	token, exists := a.jwtMgr.GetToken()
	if !exists {
		return "", fmt.Errorf("no JWT token available")
	}

	return token, nil
}