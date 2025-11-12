package storage

import (
	"context"
	"device-fleet-monitoring/internal/core"
	"sync"
	"time"
)

// DeviceAgg holds aggregate data for a single device
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

// memoryStore implements the Store interface with in-memory storage
type memoryStore struct {
	mu      sync.RWMutex
	devices map[string]*DeviceAgg
}

// NewMemoryStore creates a new in-memory store initialized with the given device IDs
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

// AddHeartbeat records a heartbeat for a device at the given timestamp
func (m *memoryStore) AddHeartbeat(ctx context.Context, deviceID string, sentAt time.Time) error {
	// Convert sentAt to minute bucket
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

	// Update firstMinute and lastMinute
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

// AddUpload records an upload time measurement for a device
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

// GetStats retrieves computed statistics for a device
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
