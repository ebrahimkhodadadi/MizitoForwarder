package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ebrahimkhodadadi/MizitoForwarder/logger"
	"github.com/ebrahimkhodadadi/MizitoForwarder/mizito"
	"github.com/gorilla/mux"
)

// GotifyNotificationRequest represents the request structure for Gotify notifications
type GotifyNotificationRequest struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
	Extras   struct {
		ClientDisplay struct {
			ContentType string `json:"contentType"`
		} `json:"client::display"`
	} `json:"extras"`
}

// NotificationResponse represents the response structure
type NotificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Handler handles HTTP requests
type Handler struct {
	messageService *mizito.MessageService
	logger         *logger.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(messageService *mizito.MessageService, logger *logger.Logger) *Handler {
	return &Handler{
		messageService: messageService,
		logger:         logger,
	}
}

// HandleGotifyNotification handles POST requests to /notification/gotify
func (h *Handler) HandleGotifyNotification(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received Gotify notification request")

	// Parse request body
	var req GotifyNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse request body", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.logger.Debug("Parsed request", "title", req.Title, "message", req.Message, "priority", req.Priority)

	// Validate required fields
	if req.Title == "" && req.Message == "" {
		h.logger.Warn("Empty notification request")
		http.Error(w, "Title or message is required", http.StatusBadRequest)
		return
	}

	// Combine title and message
	notificationText := ""
	if req.Title != "" {
		notificationText += req.Title
		if req.Message != "" {
			notificationText += ": "
		}
	}
	if req.Message != "" {
		notificationText += req.Message
	}

	// Send message to Mizito
	h.logger.Info("Sending notification to Mizito", "combined_message", notificationText)
	
	if err := h.messageService.SendMessage(notificationText); err != nil {
		h.logger.Error("Failed to send message to Mizito", "error", err)
		
		response := NotificationResponse{
			Success: false,
			Message: "Failed to send notification: " + err.Error(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success response
	response := NotificationResponse{
		Success: true,
		Message: "Notification sent successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Notification processed successfully")
}

// HealthCheck handles GET requests to /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Health check requested")

	response := map[string]string{
		"status":  "healthy",
		"message": "Mizito Forwarder is running",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Root level routes for Gotify compatibility
	router.HandleFunc("/message", h.HandleGotifyNotification).
		Methods(http.MethodPost)
	
	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()
	
	// Gotify notification endpoint (also available under API v1)
	api.HandleFunc("/message", h.HandleGotifyNotification).
		Methods(http.MethodPost)
	
	// Health check
	api.HandleFunc("/health", h.HealthCheck).
		Methods(http.MethodGet)
	
	// Root health check
	router.HandleFunc("/health", h.HealthCheck).
		Methods(http.MethodGet)
	
	// Root endpoint
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{
			"service": "Mizito Forwarder",
			"version": "1.0.0",
			"status":  "running",
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}).Methods(http.MethodGet)
}