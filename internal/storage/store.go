package storage

import (
	"context"
	"errors"
	"time"
)

// Error types
var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrInvalidInput   = errors.New("invalid input")
)

// Store defines the minimal persistence surface so we can swap implementations (memory, Postgres,
// TSDB) without changing handler code.
type Store interface {
	// AddHeartbeat records a heartbeat for a device at the given timestamp
	AddHeartbeat(ctx context.Context, deviceID string, sentAt time.Time) error

	// AddUpload records an upload time measurement for a device
	// uploadTime is treated as an opaque duration value in units provided by the device
	AddUpload(ctx context.Context, deviceID string, sentAt time.Time, uploadTime int) error

	// GetStats retrieves computed statistics for a device
	// avgUpload is returned in the same units as the input uploadTime values
	GetStats(ctx context.Context, deviceID string) (uptime float64, avgUpload float64, err error)
}
