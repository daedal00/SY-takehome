package api

import (
	"encoding/json"
	"fmt"
	"time"
)

// FlexTime demonstrates resiliency: devices can send either RFC3339 strings or Unix timestamps and we
// still parse them into a single canonical time.Time for downstream code.
type FlexTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler to handle both string and numeric timestamps
func (ft *FlexTime) UnmarshalJSON(b []byte) error {
	// Try parsing as Unix timestamp (integer)
	var unixTime int64
	if err := json.Unmarshal(b, &unixTime); err == nil {
		ft.Time = time.Unix(unixTime, 0)
		return nil
	}

	// Try parsing as RFC3339 string
	var timeStr string
	if err := json.Unmarshal(b, &timeStr); err == nil {
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
		ft.Time = t
		return nil
	}

	return fmt.Errorf("sent_at must be either Unix timestamp or RFC3339 string")
}

// HeartbeatRequest mirrors the OpenAPI schema for heartbeat ingest.
type HeartbeatRequest struct {
	SentAt FlexTime `json:"sent_at"`
}

// StatsPostRequest mirrors the stats ingest payload.
type StatsPostRequest struct {
	SentAt     FlexTime `json:"sent_at"`
	UploadTime int      `json:"upload_time"`
}

// StatsGetResponse is serialized back to the client exactly as described in the spec.
type StatsGetResponse struct {
	Uptime        float64 `json:"uptime"`
	AvgUploadTime string  `json:"avg_upload_time"`
}

// ErrorResponse ensures 4xx/5xx replies stay uniform (single msg field).
type ErrorResponse struct {
	Msg string `json:"msg"`
}
