package core

// CalculateUptime isolates the uptime math so it can be unit-tested and discussed without touching
// HTTP/storage layers.
// Returns the percentage of minutes with heartbeats within the observation window.
// Edge cases:
// - No heartbeats: returns 0.0
// - Single minute: returns 100.0 (device was online for entire observed window)
// - Multiple minutes: returns (observed minutes / total window) * 100
func CalculateUptime(minutes map[int64]struct{}, firstMinute, lastMinute int64) float64 {
	if len(minutes) == 0 {
		return 0.0
	}
	if firstMinute == lastMinute {
		return 100.0
	}
	observedMinutes := int64(len(minutes))
	totalWindow := lastMinute - firstMinute // Number of minutes between first and last
	return (float64(observedMinutes) / float64(totalWindow)) * 100.0
}

// CalculateAverageUpload performs the incremental average math (sum/count) and can later be swapped
// for more advanced statistics without touching storage handlers.
// Returns 0.0 if no uploads have been recorded.
func CalculateAverageUpload(uploadSum float64, uploadCount int64) float64 {
	if uploadCount == 0 {
		return 0.0
	}
	return uploadSum / float64(uploadCount)
}
