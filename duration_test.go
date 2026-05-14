package main

import "testing"

func TestGetSeconds(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"1d", 86400, false},
		{"2d", 172800, false},
		{"1h", 3600, false},
		{"2h", 7200, false},
		{"30m", 1800, false},
		{"60s", 60, false},
		{"", 0, true},
		{"d", 0, true},
		{"0d", 0, true},
		{"-1h", 0, true},
		{"5x", 0, true},
		{"abc", 0, true},
	}
	for _, tc := range tests {
		got, err := GetSeconds(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("GetSeconds(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("GetSeconds(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
