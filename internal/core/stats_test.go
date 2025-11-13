package core

import (
	"testing"
)

func TestCalculateUptime(t *testing.T) {
	tests := []struct {
		name        string
		minutes     map[int64]struct{}
		firstMinute int64
		lastMinute  int64
		want        float64
	}{
		{
			name:        "no heartbeats",
			minutes:     map[int64]struct{}{},
			firstMinute: 0,
			lastMinute:  0,
			want:        0.0,
		},
		{
			name:        "single minute",
			minutes:     map[int64]struct{}{100: {}},
			firstMinute: 100,
			lastMinute:  100,
			want:        100.0,
		},
		{
			name:        "consecutive minutes - 100% uptime",
			minutes:     map[int64]struct{}{0: {}, 1: {}, 2: {}},
			firstMinute: 0,
			lastMinute:  2,
			want:        150.0, // 3 minutes / 2 span = 150%
		},
		{
			name:        "sparse minutes - 75% uptime",
			minutes:     map[int64]struct{}{0: {}, 2: {}, 4: {}},
			firstMinute: 0,
			lastMinute:  4,
			want:        75.0, // 3 minutes / 4 span = 75%
		},
		{
			name:        "sparse minutes - 75% uptime",
			minutes:     map[int64]struct{}{10: {}, 12: {}, 14: {}},
			firstMinute: 10,
			lastMinute:  14,
			want:        75.0, // 3 minutes / 4 span = 75%
		},
		{
			name:        "two minutes at edges",
			minutes:     map[int64]struct{}{0: {}, 10: {}},
			firstMinute: 0,
			lastMinute:  10,
			want:        20.0, // 2 minutes / 10 span = 20%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateUptime(tt.minutes, tt.firstMinute, tt.lastMinute)
			if got != tt.want {
				t.Errorf("CalculateUptime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateAverageUpload(t *testing.T) {
	tests := []struct {
		name        string
		uploadSum   float64
		uploadCount int64
		want        float64
	}{
		{
			name:        "no uploads",
			uploadSum:   0,
			uploadCount: 0,
			want:        0.0,
		},
		{
			name:        "single upload",
			uploadSum:   100.0,
			uploadCount: 1,
			want:        100.0,
		},
		{
			name:        "multiple uploads - integer average",
			uploadSum:   300.0,
			uploadCount: 3,
			want:        100.0,
		},
		{
			name:        "multiple uploads - fractional average",
			uploadSum:   250.0,
			uploadCount: 3,
			want:        83.33333333333333,
		},
		{
			name:        "large values",
			uploadSum:   1500000.0,
			uploadCount: 10,
			want:        150000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateAverageUpload(tt.uploadSum, tt.uploadCount)
			if got != tt.want {
				t.Errorf("CalculateAverageUpload() = %v, want %v", got, tt.want)
			}
		})
	}
}
