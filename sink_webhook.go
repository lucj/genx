package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebhookSink sends each data point as a JSON POST request.
type WebhookSink struct {
	url    string
	token  string
	client *http.Client
	render Renderer
}

func NewWebhookSink(url, token string, render Renderer) *WebhookSink {
	return &WebhookSink{
		url:    url,
		token:  token,
		render: render,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *WebhookSink) Send(dp DataPoint) error {
	b, err := s.render(dp)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain so the connection can be reused
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %s", resp.Status)
	}
	return nil
}

func (s *WebhookSink) Close() error { return nil }
