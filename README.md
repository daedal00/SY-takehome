# Device Fleet Monitoring Service

A Go-based HTTP API service that monitors a fleet of edge devices (cameras, servers, switches) and reports per-device uptime and average upload time metrics.

## Overview

This service accepts periodic heartbeat pings and video upload telemetry from devices, then computes and exposes aggregated statistics through a RESTful API. The implementation conforms to the OpenAPI specification and passes validation from the device simulator.

## Features

- **Device Registration**: Load device definitions from CSV on startup
- **Heartbeat Tracking**: Record device online status with minute-level granularity
- **Upload Statistics**: Track video upload durations and calculate averages
- **Uptime Calculation**: Compute device availability as a percentage
- **Concurrent-Safe**: Handle multiple simultaneous requests without data corruption
- **Structured Logging**: Key-value formatted logs for monitoring and debugging

## Requirements

- Go 1.21 or higher (tested with Go 1.25.2)
- No external dependencies beyond Go standard library

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go           # Server entry point
├── internal/
│   ├── api/
│   │   ├── handlers.go       # HTTP request handlers
│   │   ├── handlers_test.go  # Handler tests
│   │   └── models.go         # Request/response models
│   ├── core/
│   │   ├── stats.go          # Statistics calculation logic
│   │   └── stats_test.go     # Statistics tests
│   ├── platform/
│   │   └── router.go         # HTTP routing setup
│   └── storage/
│       ├── store.go          # Storage interface
│       ├── memory.go         # In-memory implementation
│       └── memory_test.go    # Storage tests
├── devices.csv               # Device registry
└── README.md
```

## Quick Start

### 1. Build the Server

```bash
go build -o server cmd/server/main.go
```

### 2. Prepare Device Registry

Create a `devices.csv` file with device IDs (one per line):

```csv
device_id
60-6b-44-84-dc-64
b4-45-52-a2-f1-3c
26-9a-66-01-33-83
18-b8-87-e7-1f-06
38-4e-73-e0-33-59
```

### 3. Run the Server

```bash
./server -devices devices.csv
```

The server will start on port 6733 by default.

### 4. Run the Simulator

The device simulator validates the implementation against expected behavior:

```bash
# Ensure server is running on port 6733
./device-simulator-mac-arm64

# Expected output: "all done!" with matching uptime and avg_upload_time values
# See TEST_RESULTS.md for detailed validation results
```

## Configuration

The server accepts the following command-line flags:

- `-devices <path>`: Path to devices CSV file (default: `devices.csv`)
- `-port <port>`: HTTP server port (default: `6733`)

Environment variables:

- `PORT`: Override the default port (command-line flag takes precedence)

## API Endpoints

### Health Check

```bash
GET /healthz
```

Returns 200 OK when the service is operational.

### Register Heartbeat

```bash
POST /api/v1/devices/{device_id}/heartbeat
Content-Type: application/json

{
  "sent_at": "2024-04-02T16:00:00Z"
}
```

**Responses:**

- `204 No Content`: Heartbeat recorded successfully
- `400 Bad Request`: Invalid request payload
- `404 Not Found`: Device not found

### Report Upload Statistics

```bash
POST /api/v1/devices/{device_id}/stats
Content-Type: application/json

{
  "sent_at": "2024-04-02T16:00:00Z",
  "upload_time": 123456789
}
```

**Parameters:**

- `upload_time`: Duration in nanoseconds

**Responses:**

- `204 No Content`: Statistics recorded successfully
- `400 Bad Request`: Invalid request payload
- `404 Not Found`: Device not found

### Get Device Statistics

```bash
GET /api/v1/devices/{device_id}/stats
```

**Response:**

```json
{
  "uptime": 99.79167,
  "avg_upload_time": "3m7.893379134s"
}
```

**Fields:**

- `uptime`: Percentage of time device was online (0-100)
- `avg_upload_time`: Average upload duration as a Go duration string

**Responses:**

- `200 OK`: Statistics retrieved successfully
- `404 Not Found`: Device not found

## Metrics Calculations

### Uptime

```
uptime = (count of distinct minute buckets with heartbeats / minutes between first and last heartbeat) × 100
```

- Each heartbeat is bucketed into a minute (Unix timestamp / 60)
- Duplicate heartbeats in the same minute are deduplicated
- If only one heartbeat exists, uptime is 100%
- If no heartbeats exist, uptime is 0%

### Average Upload Time

```
avg_upload_time = mean of all upload_time values
```

- Calculated as the arithmetic mean of all reported upload times
- Formatted as a Go duration string (e.g., "3m7.893379134s")
- If no uploads exist, returns "0s"

## Testing

### Run Unit Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...
```

### Run Tests with Verbose Output

```bash
go test -v ./...
```

## Example Usage

### Using curl

```bash
# Register a heartbeat
curl -X POST http://127.0.0.1:6733/api/v1/devices/60-6b-44-84-dc-64/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"sent_at":"2024-04-02T16:00:00Z"}'

# Report upload statistics
curl -X POST http://127.0.0.1:6733/api/v1/devices/60-6b-44-84-dc-64/stats \
  -H "Content-Type: application/json" \
  -d '{"sent_at":"2024-04-02T16:00:00Z","upload_time":187893379134}'

# Get device statistics
curl http://127.0.0.1:6733/api/v1/devices/60-6b-44-84-dc-64/stats
```

## Design Decisions

### In-Memory Storage

The service uses an in-memory data store for simplicity and performance. Device data is held in concurrent-safe maps with mutex protection. This approach is suitable for the assessment scope but would need persistence for production use.

### Minute Bucketing

Heartbeats are bucketed by minute (Unix timestamp / 60) to efficiently track device online status. This provides minute-level granularity while keeping memory usage reasonable.

### Flexible Timestamp Parsing

The `FlexTime` type accepts both RFC3339 strings and Unix timestamps (integers) to accommodate different client implementations. Note: This extends beyond the OpenAPI spec which specifies RFC3339 format only, but was necessary to handle the simulator's behavior.

### Uptime Calculation

The uptime formula uses `lastMinute - firstMinute` (not `+1`) to match the "minutes between" interpretation, which represents the span rather than an inclusive range.

## Logging

The service provides structured logging with the following levels:

- **INFO**: Startup messages, request completion
- **ERROR**: Request validation failures, internal errors
- **DEBUG**: Raw request bodies (when enabled)

Log format:

```
INFO: 2025/11/12 15:29:28 starting server
  port=6733
  address=:6733
```

## Performance Considerations

- Concurrent request handling with goroutine-safe storage
- O(1) device lookup using maps
- O(1) heartbeat recording using map-based minute buckets
- O(n) uptime calculation where n = number of distinct minutes
- O(1) average upload time calculation (running sum and count)

## Limitations

- No data persistence (in-memory only)
- No authentication or authorization
- No rate limiting
- No metrics export (Prometheus, etc.)
- No distributed deployment support

## Solution Write-Up

### Development Time and Challenges

**Time Spent:** Approximately 4-5 hours total, including:

- Initial design and architecture planning (45 min)
- Core implementation (2 hours)
- Testing and debugging (1.5 hours)
- Documentation (45 min)

**Most Difficult Part:** The most challenging aspect was debugging the uptime calculation discrepancy with the simulator. The formula "minutes between first and last heartbeat" was ambiguous - it could mean an inclusive range (`lastMinute - firstMinute + 1`) or just the span (`lastMinute - firstMinute`). The simulator expected the span interpretation, which I validated by comparing expected vs actual results.

The second challenge was handling the simulator's `sent_at` field for stats POST requests, which sent `"0001-01-01T00:00:00Z"` (the zero time). The OpenAPI spec only requires a valid RFC3339 string, not a non-zero value, so I removed the zero-time validation to match the spec.

### Extensibility for Additional Metrics

The current architecture is designed for extensibility. To add new metric types:

**1. Storage Layer Extension:**

```go
// Add new fields to DeviceData struct
type DeviceData struct {
    // Existing fields
    heartbeats  map[int64]struct{}
    uploadSum   float64
    uploadCount int64

    // New metrics
    errorCount    int64
    lastErrorTime time.Time
    cpuUsageSum   float64
    cpuUsageCount int64
}

// Add new methods to Store interface
type Store interface {
    // Existing methods...
    AddError(ctx context.Context, deviceID string, errorType string, timestamp time.Time) error
    AddCPUUsage(ctx context.Context, deviceID string, usage float64) error
    GetExtendedStats(ctx context.Context, deviceID string) (ExtendedStats, error)
}
```

**2. API Layer Extension:**

```go
// Add new endpoints in router.go
mux.HandleFunc("POST /api/v1/devices/{device_id}/errors", handlers.HandleError)
mux.HandleFunc("POST /api/v1/devices/{device_id}/cpu", handlers.HandleCPUUsage)

// Add new response models
type ExtendedStatsResponse struct {
    Uptime         float64 `json:"uptime"`
    AvgUploadTime  string  `json:"avg_upload_time"`
    ErrorRate      float64 `json:"error_rate"`
    AvgCPUUsage    float64 `json:"avg_cpu_usage"`
}
```

**3. Calculation Layer Extension:**

```go
// Add new calculation functions in core/stats.go
func CalculateErrorRate(errorCount, totalRequests int64) float64 {
    if totalRequests == 0 {
        return 0.0
    }
    return (float64(errorCount) / float64(totalRequests)) * 100.0
}
```

**Key Design Patterns for Extensibility:**

- **Interface-based storage**: The `Store` interface allows swapping implementations (in-memory, Redis, PostgreSQL) without changing handlers
- **Separation of concerns**: Handlers, storage, and calculations are in separate packages
- **Dependency injection**: Handlers receive the store via constructor, making testing and mocking easy
- **Stateless calculations**: Pure functions in `core/stats.go` can be tested independently

### Runtime Complexity Analysis

**Storage Operations:**

| Operation            | Time Complexity | Space Complexity | Notes                                        |
| -------------------- | --------------- | ---------------- | -------------------------------------------- |
| Device lookup        | O(1)            | O(d)             | Hash map lookup; d = number of devices       |
| Add heartbeat        | O(1)            | O(m)             | Map insertion; m = unique minutes per device |
| Add upload           | O(1)            | O(1)             | Running sum and count                        |
| Calculate uptime     | O(1)            | O(1)             | Read map length and two scalars              |
| Calculate avg upload | O(1)            | O(1)             | Simple division                              |

**API Request Processing:**

| Endpoint        | Time Complexity | Notes                         |
| --------------- | --------------- | ----------------------------- |
| POST /heartbeat | O(1)            | Device lookup + map insertion |
| POST /stats     | O(1)            | Device lookup + arithmetic    |
| GET /stats      | O(1)            | Device lookup + read scalars  |

**Memory Usage:**

- Per device: O(m + u) where m = unique minutes with heartbeats, u = upload count (stored as sum)
- Total: O(d × m) where d = number of devices
- For 10,000 devices over 30 days: ~10,000 × 43,200 minutes = 432M entries worst case
- With sparse heartbeats (e.g., 99% uptime): ~428M entries
- At ~24 bytes per map entry: ~10GB memory (manageable with proper capacity planning)

**Concurrency:**

- Mutex-based locking provides O(1) lock acquisition in the uncontended case
- Lock contention scales with concurrent requests per device
- Current implementation uses per-device locking (fine-grained) for better concurrency

**Optimization Opportunities:**

1. **Minute bucket pruning**: Archive or aggregate old data beyond a retention window
2. **Lock-free structures**: Consider atomic operations for upload counters
3. **Batch processing**: Buffer heartbeats and process in batches for high-throughput scenarios
4. **Connection pooling**: If adding persistence, implement proper connection management

### Production Readiness

**Current Production-Safe Features:**

- ✅ Concurrent-safe data structures with RWMutex protection
- ✅ Graceful error handling with appropriate HTTP status codes
- ✅ Structured key-value logging for observability
- ✅ Basic input validation (device existence, payload format, negative upload times)
- ✅ No panics in request handlers

**Production Gaps (Identified):**

- ❌ **No persistence**: Data lost on restart (would need PostgreSQL/Redis)
- ❌ **No authentication**: Anyone can send telemetry (would need API keys/JWT)
- ❌ **No rate limiting**: Vulnerable to DoS (would need token bucket or leaky bucket)
- ❌ **No metrics export**: Can't monitor service health (would need Prometheus)
- ❌ **No distributed deployment**: Single point of failure (would need leader election or sharding)
- ❌ **No data retention policy**: Minute buckets grow unbounded (would need TTL or archival)
- ❌ **No circuit breakers**: No protection against downstream failures
- ❌ **No request timeouts**: Long-running requests could exhaust resources
- ❌ **Incomplete validation**: Stats POST doesn't validate sent_at is non-zero (accepts zero time per OpenAPI spec)

**Recommended Production Enhancements:**

1. Add PostgreSQL with time-series optimizations (e.g., TimescaleDB)
2. Implement Redis caching layer for hot data
3. Add middleware for authentication, rate limiting, and request tracing
4. Export metrics to Prometheus (request latency, error rates, memory usage)
5. Implement graceful shutdown with connection draining
6. Add health checks with dependency status (database, memory usage)
7. Use structured logging with correlation IDs for request tracing
8. Implement data retention policies (e.g., keep 90 days, aggregate older data)
