package api

import (
	"bytes"
	"context"
	"device-fleet-monitoring/internal/storage"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockStore is a mock implementation of storage.Store for testing
type mockStore struct {
	addHeartbeatFunc func(ctx context.Context, deviceID string, sentAt time.Time) error
	addUploadFunc    func(ctx context.Context, deviceID string, sentAt time.Time, uploadTime int) error
	getStatsFunc     func(ctx context.Context, deviceID string) (float64, float64, error)
}

func (m *mockStore) AddHeartbeat(ctx context.Context, deviceID string, sentAt time.Time) error {
	if m.addHeartbeatFunc != nil {
		return m.addHeartbeatFunc(ctx, deviceID, sentAt)
	}
	return nil
}

func (m *mockStore) AddUpload(ctx context.Context, deviceID string, sentAt time.Time, uploadTime int) error {
	if m.addUploadFunc != nil {
		return m.addUploadFunc(ctx, deviceID, sentAt, uploadTime)
	}
	return nil
}

func (m *mockStore) GetStats(ctx context.Context, deviceID string) (float64, float64, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, deviceID)
	}
	return 0, 0, nil
}

// TestHandleHeartbeat_Success tests successful heartbeat recording
func TestHandleHeartbeat_Success(t *testing.T) {
	store := &mockStore{
		addHeartbeatFunc: func(ctx context.Context, deviceID string, sentAt time.Time) error {
			if deviceID != "test-device" {
				t.Errorf("expected deviceID 'test-device', got '%s'", deviceID)
			}
			return nil
		},
	}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":"2024-01-01T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/heartbeat", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleHeartbeat(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

// TestHandleHeartbeat_DeviceNotFound tests 404 response for unknown device
func TestHandleHeartbeat_DeviceNotFound(t *testing.T) {
	store := &mockStore{
		addHeartbeatFunc: func(ctx context.Context, deviceID string, sentAt time.Time) error {
			return storage.ErrDeviceNotFound
		},
	}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":"2024-01-01T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/devices/unknown-device/heartbeat", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleHeartbeat(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Msg != "device not found" {
		t.Errorf("expected error message 'device not found', got '%s'", errResp.Msg)
	}
}

// TestHandleHeartbeat_MalformedJSON tests 400 response for invalid JSON
func TestHandleHeartbeat_MalformedJSON(t *testing.T) {
	store := &mockStore{}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":"invalid`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/heartbeat", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleHeartbeat_InvalidSentAt tests 400 response for invalid sent_at
func TestHandleHeartbeat_InvalidSentAt(t *testing.T) {
	store := &mockStore{}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":""}`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/heartbeat", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleHeartbeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleStatsPost_Success tests successful stats recording
func TestHandleStatsPost_Success(t *testing.T) {
	store := &mockStore{
		addUploadFunc: func(ctx context.Context, deviceID string, sentAt time.Time, uploadTime int) error {
			if deviceID != "test-device" {
				t.Errorf("expected deviceID 'test-device', got '%s'", deviceID)
			}
			if uploadTime != 1500 {
				t.Errorf("expected uploadTime 1500, got %d", uploadTime)
			}
			return nil
		},
	}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":"2024-01-01T12:00:00Z","upload_time":1500}`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/stats", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleStatsPost(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

// TestHandleStatsPost_DeviceNotFound tests 404 response for unknown device
func TestHandleStatsPost_DeviceNotFound(t *testing.T) {
	store := &mockStore{
		addUploadFunc: func(ctx context.Context, deviceID string, sentAt time.Time, uploadTime int) error {
			return storage.ErrDeviceNotFound
		},
	}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":"2024-01-01T12:00:00Z","upload_time":1500}`
	req := httptest.NewRequest(http.MethodPost, "/devices/unknown-device/stats", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleStatsPost(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestHandleStatsPost_NegativeUploadTime tests 400 response for negative upload_time
func TestHandleStatsPost_NegativeUploadTime(t *testing.T) {
	store := &mockStore{}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":"2024-01-01T12:00:00Z","upload_time":-100}`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/stats", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleStatsPost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Msg != "upload_time must be non-negative" {
		t.Errorf("expected error message 'upload_time must be non-negative', got '%s'", errResp.Msg)
	}
}

// TestHandleStatsPost_InvalidSentAt tests 400 response for invalid sent_at
func TestHandleStatsPost_InvalidSentAt(t *testing.T) {
	store := &mockStore{}
	handlers := NewHandlers(store)

	reqBody := `{"sent_at":"","upload_time":1500}`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/stats", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handlers.HandleStatsPost(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleStatsGet_Success tests successful stats retrieval
func TestHandleStatsGet_Success(t *testing.T) {
	store := &mockStore{
		getStatsFunc: func(ctx context.Context, deviceID string) (float64, float64, error) {
			if deviceID != "test-device" {
				t.Errorf("expected deviceID 'test-device', got '%s'", deviceID)
			}
			return 95.5, 1234.56, nil
		},
	}
	handlers := NewHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/devices/test-device/stats", nil)
	w := httptest.NewRecorder()

	handlers.HandleStatsGet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp StatsGetResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Uptime != 95.5 {
		t.Errorf("expected uptime 95.5, got %f", resp.Uptime)
	}
	if resp.AvgUploadTime != "1234.56" {
		t.Errorf("expected avg_upload_time '1234.56', got '%s'", resp.AvgUploadTime)
	}
}

// TestHandleStatsGet_DeviceNotFound tests 404 response for unknown device
func TestHandleStatsGet_DeviceNotFound(t *testing.T) {
	store := &mockStore{
		getStatsFunc: func(ctx context.Context, deviceID string) (float64, float64, error) {
			return 0, 0, storage.ErrDeviceNotFound
		},
	}
	handlers := NewHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/devices/unknown-device/stats", nil)
	w := httptest.NewRecorder()

	handlers.HandleStatsGet(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestIntegration_HeartbeatThenGetStats tests that heartbeat affects stats
func TestIntegration_HeartbeatThenGetStats(t *testing.T) {
	// Use real memory store for integration test
	memStore := storage.NewMemoryStore([]string{"test-device"})
	handlers := NewHandlers(memStore)

	// Send heartbeat
	reqBody := `{"sent_at":"2024-01-01T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/heartbeat", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handlers.HandleHeartbeat(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("heartbeat failed with status %d", w.Code)
	}

	// Get stats
	req = httptest.NewRequest(http.MethodGet, "/devices/test-device/stats", nil)
	w = httptest.NewRecorder()
	handlers.HandleStatsGet(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get stats failed with status %d", w.Code)
	}

	var resp StatsGetResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Uptime <= 0 {
		t.Errorf("expected uptime > 0 after heartbeat, got %f", resp.Uptime)
	}
}

// TestIntegration_StatsPostThenGetStats tests that upload affects stats
func TestIntegration_StatsPostThenGetStats(t *testing.T) {
	// Use real memory store for integration test
	memStore := storage.NewMemoryStore([]string{"test-device"})
	handlers := NewHandlers(memStore)

	// Send upload stats
	reqBody := `{"sent_at":"2024-01-01T12:00:00Z","upload_time":2500}`
	req := httptest.NewRequest(http.MethodPost, "/devices/test-device/stats", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handlers.HandleStatsPost(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("stats post failed with status %d", w.Code)
	}

	// Get stats
	req = httptest.NewRequest(http.MethodGet, "/devices/test-device/stats", nil)
	w = httptest.NewRecorder()
	handlers.HandleStatsGet(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get stats failed with status %d", w.Code)
	}

	var resp StatsGetResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AvgUploadTime == "0" {
		t.Errorf("expected avg_upload_time != '0' after upload, got '%s'", resp.AvgUploadTime)
	}
}
