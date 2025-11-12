package api

import "time"

// HeartbeatRequest represents the payload for POST /devices/{device_id}/heartbeat
type HeartbeatRequest struct {
	SentAt time.Time `json:"sent_at"`
}

// StatsPostRequest represents the payload for POST /devices/{device_id}/stats
type StatsPostRequest struct {
	SentAt     time.Time `json:"sent_at"`
	UploadTime int       `json:"upload_time"`
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
