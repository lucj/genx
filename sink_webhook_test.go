package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookSinkNoAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sink := NewWebhookSink(srv.URL, "")
	if err := sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: func() *float64 { v := 1.0; return &v }()}); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestWebhookSinkBearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sink := NewWebhookSink(srv.URL, "mysecrettoken")
	if err := sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: func() *float64 { v := 1.0; return &v }()}); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if gotAuth != "Bearer mysecrettoken" {
		t.Errorf("expected %q, got %q", "Bearer mysecrettoken", gotAuth)
	}
}

func TestWebhookSinkNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	sink := NewWebhookSink(srv.URL, "")
	err := sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: func() *float64 { v := 1.0; return &v }()})
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}
