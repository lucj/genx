package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FileSink writes JSON-lines to disk, optionally rotating by size or age.
// When rotation is configured, each file is named with a UTC timestamp suffix
// (e.g. out.20060102T150405.jsonl). Collisions within the same second are
// resolved by appending a counter (.2, .3, …).
// With no rotation configured, data is written to the exact path given.
type FileSink struct {
	basePath string
	maxBytes int64
	maxAge   time.Duration
	render   Renderer
	mu       sync.Mutex
	file     *os.File
	written  int64
	openedAt time.Time
	rotating bool
}

func NewFileSink(basePath string, maxBytes int64, maxAge time.Duration, render Renderer) (*FileSink, error) {
	s := &FileSink{
		basePath: basePath,
		maxBytes: maxBytes,
		maxAge:   maxAge,
		render:   render,
		rotating: maxBytes > 0 || maxAge > 0,
	}
	if err := s.openNew(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileSink) openNew() error {
	f, err := os.Create(s.nextPath())
	if err != nil {
		return err
	}
	s.file = f
	s.written = 0
	s.openedAt = time.Now()
	return nil
}

// nextPath returns the path to open for a new file.
// With rotation enabled, a timestamp suffix is inserted before the extension.
// A counter is appended when a file with the same timestamp already exists.
func (s *FileSink) nextPath() string {
	if !s.rotating {
		return s.basePath
	}
	ts := time.Now().UTC().Format("20060102T150405")
	ext := filepath.Ext(s.basePath)
	stem := strings.TrimSuffix(s.basePath, ext)
	candidate := stem + "." + ts + ext
	if _, err := os.Stat(candidate); err != nil {
		return candidate
	}
	for i := 2; ; i++ {
		candidate = stem + "." + ts + "." + strconv.Itoa(i) + ext
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}

func (s *FileSink) needsRotation(incoming int64) bool {
	if s.maxBytes > 0 && s.written+incoming > s.maxBytes {
		return true
	}
	if s.maxAge > 0 && time.Since(s.openedAt) >= s.maxAge {
		return true
	}
	return false
}

func (s *FileSink) Send(dp DataPoint) error {
	b, err := s.render(dp)
	if err != nil {
		return err
	}
	line := append(b, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rotating && s.needsRotation(int64(len(line))) {
		if err := s.file.Close(); err != nil {
			return fmt.Errorf("closing file before rotation: %w", err)
		}
		if err := s.openNew(); err != nil {
			return fmt.Errorf("opening file after rotation: %w", err)
		}
	}

	n, err := s.file.Write(line)
	s.written += int64(n)
	return err
}

func (s *FileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// ParseSize parses a human-readable byte size (e.g. "10MB", "1GB", "512KB", "1024").
// Supported suffixes: K/KB, M/MB, G/GB (case-insensitive).
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}
	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"G", 1 << 30},
		{"M", 1 << 20},
		{"K", 1 << 10},
	}
	upper := strings.ToUpper(s)
	for _, m := range multipliers {
		if strings.HasSuffix(upper, m.suffix) {
			n, err := strconv.ParseInt(strings.TrimSpace(strings.TrimSuffix(upper, m.suffix)), 10, 64)
			if err != nil || n <= 0 {
				return 0, fmt.Errorf("invalid size %q", s)
			}
			return n * m.mult, nil
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q: expected a number with optional K/KB/M/MB/G/GB suffix", s)
	}
	return n, nil
}
