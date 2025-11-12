package api

import (
	"encoding/json"
	"fmt"
	"time"
)

// FlexTime is a custom time type that can unmarshal from both RFC3339 strings and Unix timestamps
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

// HeartbeatRequest represents the payload for POST /devices/{device_id}/heartbeat
type HeartbeatRequest struct {
	SentAt FlexTime `json:"sent_at"`
}

// StatsPostRequest represents the payload for POST /devices/{device_id}/stats
type StatsPostRequest struct {
	SentAt     FlexTime `json:"sent_at"`
	UploadTime int      `json:"upload_time"`
}

// StatsGetResponse represents the response for GET /devices/{device_id}/stats
type StatsGetResponse struct {
	Uptime        float64 `json:"uptime"`
	AvgUploadTime string  `json:"avg_upload_time"`
}

// ErrorResponse represents error responses for all endpoints
type ErrorResponse struct {
	Msg string `json:"msg"`
}
