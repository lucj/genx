package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// InfluxDBSink writes data points to an InfluxDB v2 instance using the
// line-protocol write API (/api/v2/write).
type InfluxDBSink struct {
	writeURL string
	token    string
	client   *http.Client
	render   Renderer
}

func NewInfluxDBSink(rawURL, token, org, bucket, measurement string) (*InfluxDBSink, error) {
	if rawURL == "" {
		rawURL = "http://localhost:8086"
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid --influxdb-url: %w", err)
	}
	u = u.JoinPath("/api/v2/write")
	q := u.Query()
	q.Set("precision", "ns")
	if org != "" {
		q.Set("org", org)
	}
	if bucket != "" {
		q.Set("bucket", bucket)
	}
	u.RawQuery = q.Encode()

	return &InfluxDBSink{
		writeURL: u.String(),
		token:    token,
		client:   &http.Client{Timeout: 30 * time.Second},
		render:   NewInfluxRenderer(measurement),
	}, nil
}

func (s *InfluxDBSink) Send(dp DataPoint) error {
	b, err := s.render(dp)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.writeURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Token "+s.token)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("influxdb write returned %s", resp.Status)
	}
	return nil
}

func (s *InfluxDBSink) Close() error { return nil }
