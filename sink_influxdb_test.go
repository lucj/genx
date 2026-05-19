package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestInfluxDBSinkSend(t *testing.T) {
	var body, contentType, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		contentType = r.Header.Get("Content-Type")
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink, err := NewInfluxDBSink(srv.URL, "mytoken", "myorg", "mybucket", "sensors")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v := 22.5
	dp := DataPoint{Device: "dev-1", Timestamp: 1000, Value: &v}
	if err := sink.Send(dp); err != nil {
		t.Fatalf("send error: %v", err)
	}

	if body != "sensors,device=dev-1 value=22.5 1000000000000" {
		t.Errorf("body: got %q", body)
	}
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type: got %q", contentType)
	}
	if auth != "Token mytoken" {
		t.Errorf("Authorization: got %q", auth)
	}
}

func TestInfluxDBSinkNoToken(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink, _ := NewInfluxDBSink(srv.URL, "", "", "", "genx")
	v := 1.0
	sink.Send(DataPoint{Device: "dev", Timestamp: 1, Value: &v})

	if auth != "" {
		t.Errorf("expected no Authorization header when token is empty, got %q", auth)
	}
}

func TestInfluxDBSinkNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	sink, _ := NewInfluxDBSink(srv.URL, "", "", "", "genx")
	v := 1.0
	if err := sink.Send(DataPoint{Device: "dev", Timestamp: 1, Value: &v}); err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}

func TestInfluxDBSinkURLConstruction(t *testing.T) {
	sink, err := NewInfluxDBSink("http://influx.example.com:8086", "tok", "myorg", "mybucket", "m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, _ := url.Parse(sink.writeURL)
	if u.Path != "/api/v2/write" {
		t.Errorf("path: want /api/v2/write, got %s", u.Path)
	}
	q := u.Query()
	if q.Get("precision") != "ns" {
		t.Errorf("precision: want ns, got %s", q.Get("precision"))
	}
	if q.Get("org") != "myorg" {
		t.Errorf("org: want myorg, got %s", q.Get("org"))
	}
	if q.Get("bucket") != "mybucket" {
		t.Errorf("bucket: want mybucket, got %s", q.Get("bucket"))
	}
}

func TestInfluxDBSinkInvalidURL(t *testing.T) {
	if _, err := NewInfluxDBSink("://bad", "", "", "", ""); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestInfluxDBSinkMultiField(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sink, _ := NewInfluxDBSink(srv.URL, "", "", "", "env")
	dp := DataPoint{
		Device:    "room1",
		Timestamp: 5000,
		Fields:    map[string]float64{"humidity": 60.0, "temperature": 22.0},
	}
	sink.Send(dp)

	if !strings.HasPrefix(body, "env,device=room1 ") {
		t.Errorf("unexpected body: %q", body)
	}
	if !strings.Contains(body, "humidity=60") || !strings.Contains(body, "temperature=22") {
		t.Errorf("missing fields in body: %q", body)
	}
}
