package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ringBuffer is a fixed-capacity FIFO that overwrites the oldest entry when full.
type ringBuffer[T any] struct {
	buf   []T
	size  int
	head  int // index of the oldest entry
	count int // number of valid entries (0..size)
}

func newRingBuffer[T any](size int) *ringBuffer[T] {
	return &ringBuffer[T]{buf: make([]T, size), size: size}
}

func (r *ringBuffer[T]) push(v T) {
	pos := (r.head + r.count) % r.size
	r.buf[pos] = v
	if r.count < r.size {
		r.count++
	} else {
		r.head = (r.head + 1) % r.size
	}
}

// latest returns the n most recent items in chronological order.
// If n exceeds the number of stored items, all stored items are returned.
func (r *ringBuffer[T]) latest(n int) []T {
	if n > r.count {
		n = r.count
	}
	out := make([]T, n)
	start := (r.head + r.count - n + r.size) % r.size
	for i := range out {
		out[i] = r.buf[(start+i)%r.size]
	}
	return out
}

// HTTPServerSink exposes the most recent N data points as a JSON array at GET /.
// Points are stored in a fixed-size ring buffer; when full, the oldest entry is
// overwritten. The optional ?n= query parameter limits the number of points
// returned to at most n (must be ≤ buffer size).
type HTTPServerSink struct {
	mu     sync.RWMutex
	ring   *ringBuffer[DataPoint]
	server *http.Server
}

func NewHTTPServerSink(port, bufSize int) (*HTTPServerSink, error) {
	if bufSize < 1 {
		bufSize = 1
	}

	s := &HTTPServerSink{
		ring: newRingBuffer[DataPoint](bufSize),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handlePoints)

	srv, err := startHTTPServer(port, mux)
	if err != nil {
		return nil, fmt.Errorf("http-server: %w", err)
	}
	s.server = srv

	return s, nil
}

func (s *HTTPServerSink) Send(dp DataPoint) error {
	s.mu.Lock()
	s.ring.push(dp)
	s.mu.Unlock()
	return nil
}

func (s *HTTPServerSink) handlePoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	n := s.ring.count
	if qn := r.URL.Query().Get("n"); qn != "" {
		if parsed, err := strconv.Atoi(qn); err == nil && parsed > 0 && parsed < n {
			n = parsed
		}
	}
	points := s.ring.latest(n)
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(points)
}

func (s *HTTPServerSink) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
