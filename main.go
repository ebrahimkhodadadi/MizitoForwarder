package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebrahimkhodadadi/MizitoForwarder/config"
	"github.com/ebrahimkhodadadi/MizitoForwarder/handler"
	"github.com/ebrahimkhodadadi/MizitoForwarder/jwt"
	"github.com/ebrahimkhodadadi/MizitoForwarder/logger"
	"github.com/ebrahimkhodadadi/MizitoForwarder/mizito"
	"github.com/gorilla/mux"
)

func main() {
	// Initialize logger
	log, err := logger.NewLogger("info")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("Starting Mizito Forwarder...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", "error", err)
	}

	log.Info("Configuration loaded successfully", "server_port", cfg.ServerPort)

	// Initialize JWT manager
	jwtMgr := jwt.NewManager(cfg, log)

	// Initialize Mizito authentication service
	authService := mizito.NewAuthService(cfg, jwtMgr, log)

	// Initialize Mizito message service
	messageService := mizito.NewMessageService(cfg, authService, log)

	// Initialize HTTP handler
	httpHandler := handler.NewHandler(messageService, log)

	// Setup HTTP router
	router := mux.NewRouter()

	// Add middleware for logging
	router.Use(loggingMiddleware(log))

	// Register routes
	httpHandler.RegisterRoutes(router)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Load existing JWT token on startup if available
	log.Info("Loading existing JWT token...")
	if err := jwtMgr.LoadToken(); err != nil {
		log.Warn("Failed to load existing JWT token on startup", "error", err)
		// Continue startup, token will be obtained when needed
	} else {
		log.Info("Existing JWT token loaded successfully")
	}

	// Start server in a goroutine
	go func() {
		log.Info("Server starting", "address", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Give outstanding requests time to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}

// loggingMiddleware adds request logging to all HTTP requests
func loggingMiddleware(log *logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Log request
			log.Debug("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent())

			// Create response writer wrapper to capture status code
			lrw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call next handler
			next.ServeHTTP(lrw, r)

			// Log response
			duration := time.Since(start)
			log.Info("HTTP request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.statusCode,
				"duration", duration)
		})
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures both status code and response data
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	return lrw.ResponseWriter.Write(b)
}
