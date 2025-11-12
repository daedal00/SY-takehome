package storage

import (
	"context"
	"testing"
	"time"
)

func TestAddHeartbeat(t *testing.T) {
	store := NewMemoryStore([]string{"device1"})
	ctx := context.Background()

	// Test adding first heartbeat
	t1 := time.Unix(60, 0) // Minute 1
	err := store.AddHeartbeat(ctx, "device1", t1)
	if err != nil {
		t.Fatalf("AddHeartbeat failed: %v", err)
	}

	// Verify minute was added
	device := store.devices["device1"]
	device.mu.RLock()
	if len(device.minutes) != 1 {
		t.Errorf("Expected 1 minute, got %d", len(device.minutes))
	}
	if _, exists := device.minutes[1]; !exists {
		t.Error("Expected minute 1 to be recorded")
	}
	device.mu.RUnlock()

	// Test deduplication - add same minute again
	t2 := time.Unix(90, 0) // Still minute 1
	err = store.AddHeartbeat(ctx, "device1", t2)
	if err != nil {
		t.Fatalf("AddHeartbeat failed: %v", err)
	}

	device.mu.RLock()
	if len(device.minutes) != 1 {
		t.Errorf("Expected 1 minute after deduplication, got %d", len(device.minutes))
	}
	device.mu.RUnlock()

	// Test adding different minute
	t3 := time.Unix(180, 0) // Minute 3
	err = store.AddHeartbeat(ctx, "device1", t3)
	if err != nil {
		t.Fatalf("AddHeartbeat failed: %v", err)
	}

	device.mu.RLock()
	if len(device.minutes) != 2 {
		t.Errorf("Expected 2 minutes, got %d", len(device.minutes))
	}
	if device.firstMinute != 1 {
		t.Errorf("Expected firstMinute=1, got %d", device.firstMinute)
	}
	if device.lastMinute != 3 {
		t.Errorf("Expected lastMinute=3, got %d", device.lastMinute)
	}
	device.mu.RUnlock()
}
