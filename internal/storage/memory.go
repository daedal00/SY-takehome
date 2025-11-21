package storage

import (
	"context"
	"device-fleet-monitoring/internal/core"
	"sync"
	"time"
)

// DeviceAgg holds aggregate data for a single device; locks live per device so requests for
// different devices rarely contend.
type DeviceAgg struct {
	mu sync.RWMutex

	// Heartbeat tracking
	firstMinute int64              // Unix minute of first heartbeat
	lastMinute  int64              // Unix minute of last heartbeat
	minutes     map[int64]struct{} // Set of minutes with ≥1 heartbeat

	// Upload tracking (incremental average)
	uploadCount int64
	uploadSum   float64
}

// memoryStore is the interview-friendly implementation of Store – easy to reason about and
// intentionally dependency-free.
type memoryStore struct {
	mu      sync.RWMutex
	devices map[string]*DeviceAgg
}

// NewMemoryStore pre-seeds every known device so runtime lookups can reject unknown IDs immediately.
func NewMemoryStore(deviceIDs []string) *memoryStore {
	devices := make(map[string]*DeviceAgg, len(deviceIDs))
	for _, id := range deviceIDs {
		devices[id] = &DeviceAgg{
			minutes: make(map[int64]struct{}),
		}
	}
	return &memoryStore{
		devices: devices,
	}
}

// AddHeartbeat upserts minute buckets and advances the observation window for uptime calculations.
func (m *memoryStore) AddHeartbeat(ctx context.Context, deviceID string, sentAt time.Time) error {
    // Convert sentAt to minute bucket (idempotent per minute, keeps memory bounded).
	minute := sentAt.Unix() / 60

	// Acquire device with read lock on map
	m.mu.RLock()
	device, exists := m.devices[deviceID]
	m.mu.RUnlock()

	if !exists {
		return ErrDeviceNotFound
	}

	// Write lock on device for updates
	device.mu.Lock()
	defer device.mu.Unlock()

    // Update first/last minute so uptime windows only span observed data.
	if len(device.minutes) == 0 {
		device.firstMinute = minute
		device.lastMinute = minute
	} else {
		if minute < device.firstMinute {
			device.firstMinute = minute
		}
		if minute > device.lastMinute {
			device.lastMinute = minute
		}
	}

    // Add minute to set (idempotent)
	device.minutes[minute] = struct{}{}

	return nil
}

// AddUpload tracks uploads via incremental average (sum+count) to avoid storing every datapoint.
func (m *memoryStore) AddUpload(ctx context.Context, deviceID string, sentAt time.Time, uploadTime int) error {
	// Acquire device with read lock on map
	m.mu.RLock()
	device, exists := m.devices[deviceID]
	m.mu.RUnlock()

	if !exists {
		return ErrDeviceNotFound
	}

	// Write lock on device for updates
	device.mu.Lock()
	defer device.mu.Unlock()

    // Update incremental average
	device.uploadCount++
	device.uploadSum += float64(uploadTime)

	return nil
}

// GetStats reads aggregate fields under read locks and defers to pure functions for the math.
func (m *memoryStore) GetStats(ctx context.Context, deviceID string) (uptime float64, avgUpload float64, err error) {
	// Acquire device with read lock on map
	m.mu.RLock()
	device, exists := m.devices[deviceID]
	m.mu.RUnlock()

	if !exists {
		return 0, 0, ErrDeviceNotFound
	}

	// Read lock on device for calculations
	device.mu.RLock()
	defer device.mu.RUnlock()

	// Calculate uptime
	uptime = core.CalculateUptime(device.minutes, device.firstMinute, device.lastMinute)

	// Calculate average upload time
	avgUpload = core.CalculateAverageUpload(device.uploadSum, device.uploadCount)

	return uptime, avgUpload, nil
}
