package api

import (
	"bytes"
	"device-fleet-monitoring/internal/storage"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Handlers holds dependencies for HTTP handlers
type Handlers struct {
	store storage.Store
}

// NewHandlers creates a new Handlers instance with the given store
func NewHandlers(store storage.Store) *Handlers {
	return &Handlers{
		store: store,
	}
}

// HandleHeartbeat handles POST /devices/{device_id}/heartbeat
func (h *Handlers) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	// Parse device_id from URL path
	deviceID := extractDeviceID(r.URL.Path, "/api/v1/devices/", "/heartbeat")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "invalid device_id in path")
		log.Printf("ERROR: invalid device_id in path, endpoint=/heartbeat")
		return
	}

	// Parse and validate JSON body
	bodyBytes, _ := io.ReadAll(r.Body)
	log.Printf("DEBUG: raw request body, device_id=%s, endpoint=/heartbeat, body=%s", deviceID, string(bodyBytes))
	
	var req HeartbeatRequest
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		log.Printf("ERROR: failed to decode JSON, device_id=%s, endpoint=/heartbeat, error=%v", deviceID, err)
		return
	}

	// Validate sent_at is valid (time.Time zero value check)
	if req.SentAt.IsZero() {
		writeError(w, http.StatusBadRequest, "invalid sent_at timestamp")
		log.Printf("ERROR: invalid sent_at timestamp, device_id=%s, endpoint=/heartbeat", deviceID)
		return
	}

	// Call store.AddHeartbeat
	if err := h.store.AddHeartbeat(r.Context(), deviceID, req.SentAt.Time); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			log.Printf("ERROR: device not found, device_id=%s, endpoint=/heartbeat, error=%v", deviceID, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		log.Printf("ERROR: internal error, device_id=%s, endpoint=/heartbeat, error=%v", deviceID, err)
		return
	}

	// Return 204 on success
	w.WriteHeader(http.StatusNoContent)
	log.Printf("INFO: request completed, method=POST, path=/devices/%s/heartbeat, device_id=%s, status=204", deviceID, deviceID)
}

// HandleStatsPost handles POST /devices/{device_id}/stats
func (h *Handlers) HandleStatsPost(w http.ResponseWriter, r *http.Request) {
	// Parse device_id from URL path
	deviceID := extractDeviceID(r.URL.Path, "/api/v1/devices/", "/stats")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "invalid device_id in path")
		log.Printf("ERROR: invalid device_id in path, endpoint=/stats")
		return
	}

	// Parse and validate JSON body
	bodyBytes, _ := io.ReadAll(r.Body)
	log.Printf("DEBUG: raw request body, device_id=%s, endpoint=/stats, body=%s", deviceID, string(bodyBytes))
	
	var req StatsPostRequest
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		log.Printf("ERROR: failed to decode JSON, device_id=%s, endpoint=/stats, error=%v", deviceID, err)
		return
	}

	// Validate upload_time >= 0
	if req.UploadTime < 0 {
		writeError(w, http.StatusBadRequest, "upload_time must be non-negative")
		log.Printf("ERROR: negative upload_time, device_id=%s, endpoint=/stats, upload_time=%d", deviceID, req.UploadTime)
		return
	}

	// Call store.AddUpload
	if err := h.store.AddUpload(r.Context(), deviceID, req.SentAt.Time, req.UploadTime); err != nil{
		if errors.Is(err, storage.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			log.Printf("ERROR: device not found, device_id=%s, endpoint=/stats, error=%v", deviceID, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		log.Printf("ERROR: internal error, device_id=%s, endpoint=/stats, error=%v", deviceID, err)
		return
	}

	// Return 204 on success
	w.WriteHeader(http.StatusNoContent)
	log.Printf("INFO: request completed, method=POST, path=/devices/%s/stats, device_id=%s, status=204", deviceID, deviceID)
}

// HandleStatsGet handles GET /devices/{device_id}/stats
func (h *Handlers) HandleStatsGet(w http.ResponseWriter, r *http.Request) {
	// Parse device_id from URL path
	deviceID := extractDeviceID(r.URL.Path, "/api/v1/devices/", "/stats")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "invalid device_id in path")
		log.Printf("ERROR: invalid device_id in path, endpoint=/stats")
		return
	}

	// Call store.GetStats
	uptime, avgUpload, err := h.store.GetStats(r.Context(), deviceID)
	if err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			writeError(w, http.StatusNotFound, "device not found")
			log.Printf("ERROR: device not found, device_id=%s, endpoint=/stats, error=%v", deviceID, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		log.Printf("ERROR: internal error, device_id=%s, endpoint=/stats, error=%v", deviceID, err)
		return
	}

	// Format avg_upload_time as duration string (input is in nanoseconds)
	avgUploadTimeStr := formatDuration(avgUpload)

	// Return 200 with JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(StatsGetResponse{
		Uptime:        uptime,
		AvgUploadTime: avgUploadTimeStr,
	})
	log.Printf("INFO: request completed, method=GET, path=/devices/%s/stats, device_id=%s, status=200", deviceID, deviceID)
}

// extractDeviceID extracts device_id from URL path
// Example: /devices/abc-123/heartbeat -> abc-123
func extractDeviceID(path, prefix, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	path = strings.TrimPrefix(path, prefix)
	if suffix != "" && strings.HasSuffix(path, suffix) {
		path = strings.TrimSuffix(path, suffix)
	}
	return path
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Msg: message})
}

// formatDuration formats a float64 (nanoseconds) as a Go duration string
func formatDuration(nanoseconds float64) string {
	if nanoseconds == 0 {
		return "0s"
	}
	duration := time.Duration(nanoseconds)
	return duration.String()
}
