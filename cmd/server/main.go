package main

import (
	"device-fleet-monitoring/internal/api"
	"device-fleet-monitoring/internal/platform"
	"device-fleet-monitoring/internal/storage"
	"encoding/csv"
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	// Define command-line flags
	port := flag.String("port", getEnv("PORT", "6733"), "HTTP server port")
	devicesCSV := flag.String("devices", getEnv("DEVICES_CSV", "devices.csv"), "Path to devices CSV file")
	flag.Parse()

	// Initialize logger
	logger := platform.NewLogger()

	// Load device IDs from CSV
	deviceIDs, err := loadDeviceIDs(*devicesCSV)
	if err != nil {
		logger.Error("failed to load devices from CSV",
			"file", *devicesCSV,
			"error", err)
		os.Exit(1)
	}

	logger.Info("loaded devices from CSV",
		"file", *devicesCSV,
		"count", len(deviceIDs))

	// Create memory store with loaded device IDs
	store := storage.NewMemoryStore(deviceIDs)

	// Create handlers with store
	handlers := api.NewHandlers(store)

	// Set up router with handlers
	router := platform.NewRouter(platform.RouterConfig{
		Handlers:    handlers,
		Logger:      logger,
		DeviceCount: len(deviceIDs),
	})

	// Start HTTP server
	addr := ":" + *port
	logger.Info("starting server",
		"port", *port,
		"address", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server failed",
			"error", err)
		os.Exit(1)
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadDeviceIDs reads device IDs from a CSV file
func loadDeviceIDs(filename string) ([]string, error) {
	// Open CSV file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Validate CSV has at least header row
	if len(records) < 1 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Validate header
	if len(records[0]) < 1 || records[0][0] != "device_id" {
		return nil, fmt.Errorf("CSV must have 'device_id' column header")
	}

	// Extract device IDs (skip header row)
	deviceIDs := make([]string, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 1 {
			continue // Skip empty rows
		}
		deviceID := records[i][0]
		if deviceID != "" {
			deviceIDs = append(deviceIDs, deviceID)
		}
	}

	if len(deviceIDs) == 0 {
		return nil, fmt.Errorf("no device IDs found in CSV")
	}

	return deviceIDs, nil
}
