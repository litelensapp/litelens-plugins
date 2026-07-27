package helm

import (
	"testing"
	"time"
)

func TestHelmAge(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"minutes: under one hour", 45 * time.Minute, "45m"},
		{"minutes: zero", 0, "0m"},
		{"hours: exactly one hour", 1 * time.Hour, "1h"},
		{"hours: 23 hours", 23 * time.Hour, "23h"},
		{"days: exactly one day", 24 * time.Hour, "1d"},
		{"days: 7 days", 7 * 24 * time.Hour, "7d"},
		{"days: 30 days", 30 * 24 * time.Hour, "30d"},
		{"future timestamp clamps to zero", -5 * time.Minute, "0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Negative tt.d means a future timestamp (clamp case).
			// time.Now().Add(-tt.d) moves the anchor in the correct direction for both cases.
			ts := time.Now().Add(-tt.d)
			got := helmAge(ts)
			if got != tt.want {
				t.Errorf("helmAge(%v) = %q; want %q", tt.d, got, tt.want)
			}
		})
	}
}
