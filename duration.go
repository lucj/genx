package main

import (
	"fmt"
	"strconv"
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
