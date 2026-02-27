package handler

import (
	"encoding/json"
	"net/http"
	"strings"

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
	appToken       string
}

// NewHandler creates a new HTTP handler
func NewHandler(messageService *mizito.MessageService, logger *logger.Logger, appToken string) *Handler {
	return &Handler{
		messageService: messageService,
		logger:         logger,
		appToken:       appToken,
	}
}

// AppTokenMiddleware validates the APP_TOKEN on protected routes.
// It accepts the token via:
//   - Query parameter:          ?token=<token>
//   - Authorization header:     Authorization: Bearer <token>
//   - Gotify-compatible header: X-Gotify-Key: <token>
//
// When APP_TOKEN is not configured the middleware is skipped (open access).
func (h *Handler) AppTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no token is configured, allow all requests
		if h.appToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		var provided string

		// 1. Query parameter: ?token=<token>
		if t := r.URL.Query().Get("token"); t != "" {
			provided = t
		}

		// 2. Authorization: Bearer <token>
		if provided == "" {
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				provided = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if provided != h.appToken {
			h.logger.Warn("Unauthorized request – invalid or missing app token",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Unauthorized",
				"message": "Valid app token required. Pass it via ?token=, Authorization: Bearer, or X-Gotify-Key header.",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
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
	// Public routes (no auth required)
	router.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)
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

	// API v1 public routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)

	// Protected subrouters – app token authentication applied
	protected := router.NewRoute().Subrouter()
	protected.Use(h.AppTokenMiddleware)
	protected.HandleFunc("/message", h.HandleGotifyNotification).Methods(http.MethodPost)

	protectedAPI := api.NewRoute().Subrouter()
	protectedAPI.Use(h.AppTokenMiddleware)
	protectedAPI.HandleFunc("/message", h.HandleGotifyNotification).Methods(http.MethodPost)
}
