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

	sink := NewWebhookSink(srv.URL, "", "", JSONRenderer)
	v := 1.0
	if err := sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: &v}); err != nil {
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

	sink := NewWebhookSink(srv.URL, "mysecrettoken", "", JSONRenderer)
	v := 1.0
	if err := sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: &v}); err != nil {
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

	sink := NewWebhookSink(srv.URL, "", "", JSONRenderer)
	v := 1.0
	err := sink.Send(DataPoint{Device: "d", Timestamp: 1, Value: &v})
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}
