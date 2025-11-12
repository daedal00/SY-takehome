package platform

import (
	"device-fleet-monitoring/internal/api"
	"encoding/json"
	"net/http"
	"time"
)

// RouterConfig holds configuration for the router
type RouterConfig struct {
	Handlers    *api.Handlers
	Logger      *Logger
	DeviceCount int
}

// NewRouter creates and configures an HTTP router with middleware
func NewRouter(config RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Wrap handlers with logging middleware
	heartbeatHandler := loggingMiddleware(config.Logger, http.HandlerFunc(config.Handlers.HandleHeartbeat))
	statsPostHandler := loggingMiddleware(config.Logger, http.HandlerFunc(config.Handlers.HandleStatsPost))
	statsGetHandler := loggingMiddleware(config.Logger, http.HandlerFunc(config.Handlers.HandleStatsGet))

	// Register API endpoints with /api/v1 prefix
	mux.Handle("/api/v1/devices/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route based on method and path suffix
		if r.URL.Path == "/api/v1/devices/" {
			http.NotFound(w, r)
			return
		}

		// Check if path ends with /heartbeat
		if len(r.URL.Path) > len("/heartbeat") && r.URL.Path[len(r.URL.Path)-len("/heartbeat"):] == "/heartbeat" {
			if r.Method == http.MethodPost {
				heartbeatHandler.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if path ends with /stats
		if len(r.URL.Path) > len("/stats") && r.URL.Path[len(r.URL.Path)-len("/stats"):] == "/stats" {
			if r.Method == http.MethodPost {
				statsPostHandler.ServeHTTP(w, r)
				return
			}
			if r.Method == http.MethodGet {
				statsGetHandler.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		http.NotFound(w, r)
	}))

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"devices": config.DeviceCount,
		})
	})

	return mux
}

// loggingMiddleware logs HTTP requests and responses
func loggingMiddleware(logger *Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log request completion
		duration := time.Since(start)
		logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
