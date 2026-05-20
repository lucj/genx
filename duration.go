package main

import (
	"fmt"
	"strconv"
	"time"
)

// GetSeconds parses a duration string like "2d", "3h", "30m", "60s"
// and returns the equivalent number of seconds.
func GetSeconds(ts string) (int, error) {
	if len(ts) < 2 {
		return 0, fmt.Errorf("invalid duration %q", ts)
	}
	ind := string(ts[len(ts)-1])
	mul := 0
	switch ind {
	case "d":
		mul = 24 * 60 * 60
	case "h":
		mul = 60 * 60
	case "m":
		mul = 60
	case "s":
		mul = 1
	default:
		return 0, fmt.Errorf("unrecognized duration unit %q (use d/h/m/s)", ind)
	}
	nbr, err := strconv.Atoi(ts[:len(ts)-1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration value in %q: %w", ts, err)
	}
	if nbr <= 0 {
		return 0, fmt.Errorf("duration must be positive, got %q", ts)
	}
	return nbr * mul, nil
}

// ParseFromTime parses a --from value and returns a Unix timestamp.
// Accepted formats: ISO 8601 with timezone, date-only (UTC midnight), Unix epoch integer.
func ParseFromTime(s string) (int64, error) {
	if s == "" {
		return time.Now().Unix(), nil
	}
	// Unix epoch integer
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ts, nil
	}
	// ISO 8601 with timezone, then without, then date-only
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Unix(), nil
		}
	}
	return 0, fmt.Errorf("cannot parse %q as a timestamp — use ISO 8601 (e.g. 2024-01-01T00:00:00Z) or a Unix epoch integer", s)
}
