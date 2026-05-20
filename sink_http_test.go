package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- ringBuffer tests ---

func TestRingBufferPushAndLatest(t *testing.T) {
	r := newRingBuffer[int](3)

	r.push(1)
	r.push(2)
	r.push(3)

	got := r.latest(3)
	want := []int{1, 2, 3}
	if !slicesEqual(got, want) {
		t.Errorf("latest(3) = %v, want %v", got, want)
	}
}

func TestRingBufferWrap(t *testing.T) {
	r := newRingBuffer[int](3)

	for _, v := range []int{1, 2, 3, 4, 5} {
		r.push(v)
	}

	// Buffer holds [3, 4, 5] after wrapping.
	got := r.latest(3)
	want := []int{3, 4, 5}
	if !slicesEqual(got, want) {
		t.Errorf("after wrap, latest(3) = %v, want %v", got, want)
	}
}

func TestRingBufferLatestN(t *testing.T) {
	r := newRingBuffer[int](5)
	for i := 1; i <= 5; i++ {
		r.push(i)
	}

	got := r.latest(2)
	want := []int{4, 5}
	if !slicesEqual(got, want) {
		t.Errorf("latest(2) = %v, want %v", got, want)
	}
}

func TestRingBufferLatestExceedsCount(t *testing.T) {
	r := newRingBuffer[int](5)
	r.push(10)
	r.push(20)

	got := r.latest(10) // ask for more than stored
	want := []int{10, 20}
	if !slicesEqual(got, want) {
		t.Errorf("latest(10) with 2 stored = %v, want %v", got, want)
	}
}

func TestRingBufferEmpty(t *testing.T) {
	r := newRingBuffer[int](3)
	got := r.latest(3)
	if len(got) != 0 {
		t.Errorf("latest on empty buffer = %v, want []", got)
	}
}

// --- HTTPServerSink tests ---

func TestHTTPServerSinkSendAndRetrieve(t *testing.T) {
	sink, err := NewHTTPServerSink(0, 5)
	if err != nil {
		t.Fatalf("NewHTTPServerSink: %v", err)
	}
	defer sink.Close()

	// Directly push points without going through HTTP.
	v1, v2 := 1.0, 2.0
	sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: &v1})
	sink.Send(DataPoint{Device: "d", Timestamp: 2, Value: &v2})

	sink.mu.RLock()
	points := sink.ring.latest(sink.ring.count)
	sink.mu.RUnlock()

	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if *points[0].Value != 1.0 || *points[1].Value != 2.0 {
		t.Errorf("unexpected values: %v", points)
	}
}

func TestHTTPServerSinkHandlerReturnsJSON(t *testing.T) {
	sink, err := NewHTTPServerSink(0, 10)
	if err != nil {
		t.Fatalf("NewHTTPServerSink: %v", err)
	}
	defer sink.Close()

	v := 42.0
	sink.Send(DataPoint{Device: "sensor", Timestamp: 1000, Value: &v})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	sink.handlePoints(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var points []DataPoint
	if err := json.NewDecoder(rec.Body).Decode(&points); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Device != "sensor" || *points[0].Value != 42.0 {
		t.Errorf("unexpected point: %+v", points[0])
	}
}

func TestHTTPServerSinkHandlerQueryN(t *testing.T) {
	sink, err := NewHTTPServerSink(0, 10)
	if err != nil {
		t.Fatalf("NewHTTPServerSink: %v", err)
	}
	defer sink.Close()

	for i := 1; i <= 5; i++ {
		v := float64(i)
		sink.Send(DataPoint{Device: "d", Timestamp: int64(i), Value: &v})
	}

	req := httptest.NewRequest(http.MethodGet, "/?n=2", nil)
	rec := httptest.NewRecorder()
	sink.handlePoints(rec, req)

	var points []DataPoint
	json.NewDecoder(rec.Body).Decode(&points)
	if len(points) != 2 {
		t.Fatalf("expected 2 points with ?n=2, got %d", len(points))
	}
	if *points[0].Value != 4.0 || *points[1].Value != 5.0 {
		t.Errorf("expected last 2 points, got %v", points)
	}
}

func TestHTTPServerSinkHandlerMethodNotAllowed(t *testing.T) {
	sink, err := NewHTTPServerSink(0, 5)
	if err != nil {
		t.Fatalf("NewHTTPServerSink: %v", err)
	}
	defer sink.Close()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	sink.handlePoints(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestNewHTTPServerSinkPortConflict(t *testing.T) {
	s1, err := NewHTTPServerSink(0, 1)
	if err != nil {
		t.Fatalf("first sink: %v", err)
	}
	defer s1.Close()

	// Parse the actual bound port from the server address (set by startHTTPServer).
	addr, err := net.ResolveTCPAddr("tcp", s1.server.Addr)
	if err != nil {
		t.Fatalf("resolve addr: %v", err)
	}

	// s1 holds the port open, so the second bind must fail immediately.
	_, err = NewHTTPServerSink(addr.Port, 1)
	if err == nil {
		t.Error("expected error binding to already-used port, got nil")
	}
}

// slicesEqual reports whether two int slices have identical elements.
func slicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
