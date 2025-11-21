package platform

import (
	"device-fleet-monitoring/internal/api"
	"encoding/json"
	"net/http"
	"time"
)

// RouterConfig holds configuration for the router
// It lets main() wire in dependencies (handlers, logging, device count)
// without hard-coding those things inside the router package
type RouterConfig struct {
	Handlers    *api.Handlers
	Logger      *Logger
	DeviceCount int
}

// NewRouter wires the HTTP surface area without external frameworks so the interviewers can see
// exactly how requests are dispatched and how middleware is layered.
// Design goals:
// - Keep routing simple and dependency-free for this small API surface
// - Centralize route wiring in one place
// - Attach cross-cutting middleware (logging) without pulling in a full framework
func NewRouter(config RouterConfig) http.Handler {
	// Use Go's standard multiplexer. For a small numer of routes, http.serveMux is more than enough
	mux := http.NewServeMux()

    // Wrap handlers with logging middleware so we still get framework-like observability without
    // paying the dependency cost.
	heartbeatHandler := loggingMiddleware(config.Logger, http.HandlerFunc(config.Handlers.HandleHeartbeat))
	statsPostHandler := loggingMiddleware(config.Logger, http.HandlerFunc(config.Handlers.HandleStatsPost))
	statsGetHandler := loggingMiddleware(config.Logger, http.HandlerFunc(config.Handlers.HandleStatsGet))

    // Register a single prefix route for all device-related endpoints so we can keep all
    // device-aware routing logic in one closure instead of scattering mux.HandleFunc calls.
    // The inner handler does manual routing based on method and path suffix
	// Routes covered here:
	// POST /api/v1/devices/{id}/heartbeat
	// POST /api/v1/devices/{id}/stats
	// GET /api/v1/devices/{id}/stats
	mux.Handle("/api/v1/devices/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route based on method and path suffix
		// If someone hits exactly /api/v1/devices, return 404.
		// We expect an ID and a suffix after the prefix
		if r.URL.Path == "/api/v1/devices/" {
			http.NotFound(w, r)
			return
		}

		// Check if path ends with /heartbeat
		// ex: /api/v1/devices/abc-123/heartbeat
		if len(r.URL.Path) > len("/heartbeat") && r.URL.Path[len(r.URL.Path)-len("/heartbeat"):] == "/heartbeat" {
			// only support POST for heartbeat
			if r.Method == http.MethodPost {
				heartbeatHandler.ServeHTTP(w, r)
				return
			}
			// any other method is explicity rejected
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if path ends with /stats
		// Ex:
		// POST /api/v1/devices/abc-123/stats -> ingest upload metrics
		// GET /api/v1/devices/abc-123/stats -> read aggregate stats
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

		// if path doesn't match any ofthe supported suffixes, return 404
		http.NotFound(w, r)
	}))

	// Health check endpoint
	// Inentionally simple and dependency-free so it works in any environment
	// (local dev, container, k8s, etc)
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

// loggingMiddleware is the hand-rolled equivalent of chi/gin request logging so we can explain the
// moving parts in an interview without referencing a black-box dependency.
// Wraps the next handler, records the start time, captures the status code, and then logs method,
// path, status, and duration using the injected logger.
func loggingMiddleware(logger *Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer so we can intercept WriteHeader calls and  to capture final status code for logging.
		wrapped := &responseWriter{
			ResponseWriter: w, 
			statusCode: http.StatusOK, // default assumption until WriteHeader is called
		}

		// Invoke the next handler in the chain
		next.ServeHTTP(wrapped, r)

		// After the handler finishes, compute request duration and log
		duration := time.Since(start)
		logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// responseWriter mirrors the common middleware pattern of decorating http.ResponseWriter so we can
// capture the status code even when handlers only call Write().
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates underlying writer
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
