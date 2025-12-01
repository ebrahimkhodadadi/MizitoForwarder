package mizito

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/ebrahimkhodadadi/MizitoForwarder/config"
	"github.com/ebrahimkhodadadi/MizitoForwarder/logger"
)

// MessageRequest represents the message request structure based on the provided curl example
type MessageRequest struct {
	Underscore          string                 `json:"_"`
	ID                  int                    `json:"_id"`
	Local               int                    `json:"local"`
	Dialog              string                 `json:"dialog"`
	Out                 bool                   `json:"out"`
	Message             string                 `json:"message"`
	Media               interface{}            `json:"media"`
	From                string                 `json:"from"`
	Date                int64                  `json:"date"`
	SeenCount           int                    `json:"seen_count"`
	RandomID            float64                `json:"randomId"`
	Pending             bool                   `json:"pending"`
	Mid                 int                    `json:"mid"`
	Id                  int                    `json:"id"`
	RichMessageEntities []interface{}          `json:"richMessageEntities"`
	RichMessage         map[string]interface{} `json:"richMessage"`
	RDate               string                 `json:"rDate"`
	RTime               string                 `json:"rTime"`
	RFullDate           string                 `json:"rFullDate"`
	Seen                bool                   `json:"seen"`
	StartUnread         bool                   `json:"start_unread"`
	NeedAvatar          bool                   `json:"needAvatar"`
	NeedDate            bool                   `json:"needDate"`
	Dir                 bool                   `json:"dir"`
}

// MessageResponse represents the message send response structure
type MessageResponse struct {
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// BoolResponse represents a boolean response from Mizito API
type BoolResponse bool

// MessageService handles sending messages to Mizito chat API
type MessageService struct {
	config *config.Config
	auth   *AuthService
	logger *logger.Logger
	client *http.Client
}

// NewMessageService creates a new message service
func NewMessageService(config *config.Config, auth *AuthService, logger *logger.Logger) *MessageService {
	return &MessageService{
		config: config,
		auth:   auth,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

// SendMessage sends a message to the specified dialog
func (m *MessageService) SendMessage(messageText string) error {
	m.logger.Info("Sending message to Mizito chat", "message", messageText)

	// Get JWT token
	token, err := m.auth.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get JWT token: %w", err)
	}

	// Generate current time in milliseconds
	now := time.Now()
	date := now.UnixNano() / int64(time.Millisecond)

	// Generate random ID
	randomID := rand.Float64()

	// Create Persian date/time strings (simplified for demo)
	persianDate := m.formatPersianDate(now)
	persianTime := m.formatPersianTime(now)
	persianFullDate := fmt.Sprintf("%s - %s", persianTime, persianDate)

	// Prepare message request
	msgReq := MessageRequest{
		Underscore:          "message",
		ID:                  1,
		Local:               1,
		Dialog:              m.config.MizitoDialogID,
		Out:                 true,
		Message:             messageText,
		Media:               nil,
		From:                m.config.MizitoFromUserID,
		Date:                date,
		SeenCount:           1,
		RandomID:            randomID,
		Pending:             true,
		Mid:                 1,
		Id:                  1,
		RichMessageEntities: []interface{}{},
		RichMessage:         map[string]interface{}{},
		RDate:               persianDate,
		RTime:               persianTime,
		RFullDate:           persianFullDate,
		Seen:                false,
		StartUnread:         false,
		NeedAvatar:          true,
		NeedDate:            true,
		Dir:                 true,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(msgReq)
	if err != nil {
		return fmt.Errorf("failed to marshal message request: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", m.config.MizitoChatAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create message request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://office.mizito.ir/")
	req.Header.Set("x-token", token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", "\"Chromium\";v=\"142\", \"Brave\";v=\"142\", \"Not_A Brand\";v=\"99\"")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "\"Windows\"")

	m.logger.Debug("Message request headers", "headers", req.Header)
	m.logger.Debug("Message request body", "body", string(jsonData))

	// Make request with retry logic for unauthorized errors
	return m.sendMessageWithRetry(req, 2)
}

// sendMessageWithRetry sends a message with retry logic for unauthorized errors
func (m *MessageService) sendMessageWithRetry(req *http.Request, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			m.logger.Info("Retrying message send", "attempt", attempt)
			// Refresh token on retry
			if err := m.auth.RefreshToken(); err != nil {
				lastErr = fmt.Errorf("failed to refresh token on retry: %w", err)
				continue
			}

			// Update token in request
			token, err := m.auth.GetToken()
			if err != nil {
				lastErr = fmt.Errorf("failed to get refreshed token: %w", err)
				continue
			}
			req.Header.Set("x-token", token)
		}

		// Make request
		resp, err := m.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("message request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		// Read response
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read message response: %w", err)
			continue
		}

		m.logger.Debug("Message response status", "status", resp.StatusCode)
		m.logger.Debug("Message response body", "body", string(body))

		// Check HTTP status
		if resp.StatusCode == http.StatusUnauthorized {
			if attempt < maxRetries {
				m.logger.Warn("Unauthorized response, will retry with fresh token")
				continue
			}
			return fmt.Errorf("message send failed with unauthorized status after %d retries", maxRetries+1)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("message send failed with status: %d, body: %s", resp.StatusCode, string(body))
		}

		// Try to parse as boolean first
		var boolResp BoolResponse
		if err := json.Unmarshal(body, &boolResp); err == nil {
			// Response is a boolean
			if bool(boolResp) {
				m.logger.Debug("Message sent successfully (boolean response)")
				return nil
			} else {
				return fmt.Errorf("message send failed: received false response")
			}
		}

		// Try to parse as JSON response
		var msgResp MessageResponse
		if err := json.Unmarshal(body, &msgResp); err == nil {
			// Check response status
			if msgResp.Status != 1 {
				return fmt.Errorf("message send failed with status: %d, message: %s", msgResp.Status, msgResp.Message)
			}
		} else {
			// Could not parse response at all
			m.logger.Warn("Could not parse Mizito response", "response", string(body))
			return fmt.Errorf("message send failed: unexpected response format")
		}

		m.logger.Info("Message sent successfully to Mizito chat")
		return nil
	}

	return fmt.Errorf("message send failed after %d attempts, last error: %w", maxRetries+1, lastErr)
}

// formatPersianDate formats date in Persian (simplified implementation)
func (m *MessageService) formatPersianDate(t time.Time) string {
	// This is a simplified implementation
	// In a real application, you might want to use a proper Persian calendar library
	weekdays := []string{"یکشنبه", "دوشنبه", "سه‌شنبه", "چهارشنبه", "پنج‌شنبه", "جمعه", "شنبه"}
	weekday := weekdays[t.Weekday()]

	// Get Persian day and month (simplified)
	persianDay := t.Day()
	persianMonth := t.Month()

	// Persian months (simplified)
	months := []string{"", "فروردین", "اردیبهشت", "خرداد", "تیر", "مرداد", "شهریور", "مهر", "آبان", "آذر", "دی", "بهمن", "اسفند"}

	return fmt.Sprintf("%s %d %s", weekday, persianDay, months[persianMonth])
}

// formatPersianTime formats time in Persian
func (m *MessageService) formatPersianTime(t time.Time) string {
	hour := t.Hour()
	minute := t.Minute()

	// Convert to Persian numerals (simplified)
	persianHour := m.toPersianNumeral(hour)
	persianMinute := m.toPersianNumeral(minute)

	return fmt.Sprintf("%s:%s", persianHour, persianMinute)
}

// toPersianNumeral converts numbers to Persian numerals (simplified)
func (m *MessageService) toPersianNumeral(num int) string {
	// This is a simplified implementation
	// Persian numerals: ۰۱۲۳۴۵۶۷۸۹
	persianDigits := []rune{'۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'}

	result := ""
	for num > 0 {
		digit := num % 10
		result = string(persianDigits[digit]) + result
		num /= 10
	}

	if result == "" {
		result = "۰۰"
	}

	return result
}
