package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// WebhookSink sends each data point as a JSON POST request.
type WebhookSink struct {
	url    string
	client *http.Client
}

func NewWebhookSink(url string) *WebhookSink {
	return &WebhookSink{url: url, client: &http.Client{}}
}

func (s *WebhookSink) Send(dp DataPoint) error {
	b, err := json.Marshal(dp)
	if err != nil {
		return err
	}
	resp, err := s.client.Post(s.url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %s", resp.Status)
	}
	return nil
}

func (s *WebhookSink) Close() error { return nil }
