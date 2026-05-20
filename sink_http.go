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

// HTTPServerSink exposes the most recent N data points as a JSON array at GET /.
// Points are stored in a fixed-size ring buffer; when full, the oldest entry is
// overwritten. The optional ?n= query parameter limits the number of points
// returned to at most n (must be ≤ buffer size).
type HTTPServerSink struct {
	mu    sync.RWMutex
	buf   []DataPoint
	size  int // ring buffer capacity
	head  int // index of the oldest stored entry
	count int // number of valid entries (0..size)

	server *http.Server
	errCh  chan error
}

func NewHTTPServerSink(port, bufSize int) (*HTTPServerSink, error) {
	if bufSize < 1 {
		bufSize = 1
	}

	s := &HTTPServerSink{
		buf:   make([]DataPoint, bufSize),
		size:  bufSize,
		errCh: make(chan error, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handlePoints)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.errCh <- err
		}
	}()

	// Brief pause to surface immediate startup errors (e.g. port already in use).
	select {
	case err := <-s.errCh:
		return nil, fmt.Errorf("http-server listener: %w", err)
	case <-time.After(50 * time.Millisecond):
	}

	return s, nil
}

func (s *HTTPServerSink) Send(dp DataPoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos := (s.head + s.count) % s.size
	s.buf[pos] = dp
	if s.count < s.size {
		s.count++
	} else {
		s.head = (s.head + 1) % s.size
	}
	return nil
}

func (s *HTTPServerSink) handlePoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	n := s.count
	if qn := r.URL.Query().Get("n"); qn != "" {
		if parsed, err := strconv.Atoi(qn); err == nil && parsed > 0 && parsed < n {
			n = parsed
		}
	}
	// Copy the n most recent points in chronological order.
	points := make([]DataPoint, n)
	start := (s.head + s.count - n + s.size) % s.size
	for i := range points {
		points[i] = s.buf[(start+i)%s.size]
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(points)
}

func (s *HTTPServerSink) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
