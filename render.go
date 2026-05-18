package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"
)

// Renderer serialises a DataPoint to bytes for a sink to transmit.
type Renderer func(DataPoint) ([]byte, error)

// JSONRenderer is the default renderer — plain JSON marshalling with Unix epoch timestamp.
func JSONRenderer(dp DataPoint) ([]byte, error) {
	return json.Marshal(dp)
}

// isoDataPoint mirrors DataPoint but emits timestamp as an ISO 8601 UTC string.
type isoDataPoint struct {
	Device    string             `json:"device"`
	Timestamp string             `json:"timestamp"`
	Value     *float64           `json:"value,omitempty"`
	Fields    map[string]float64 `json:"fields,omitempty"`
}

// ISOJSONRenderer renders a DataPoint with timestamp as ISO 8601 UTC string.
func ISOJSONRenderer(dp DataPoint) ([]byte, error) {
	return json.Marshal(isoDataPoint{
		Device:    dp.Device,
		Timestamp: time.Unix(dp.Timestamp, 0).UTC().Format(time.RFC3339),
		Value:     dp.Value,
		Fields:    dp.Fields,
	})
}

// templateData is the value passed to a payload template on each render.
type templateData struct {
	Device       string
	Timestamp    int64
	TimestampISO string
	Value        float64
	Fields       map[string]float64
}

// NewTemplateRenderer returns a Renderer that executes tmpl and validates the
// output as JSON before returning it.
func NewTemplateRenderer(tmpl *template.Template, isoTime bool) Renderer {
	return func(dp DataPoint) ([]byte, error) {
		td := templateData{
			Device:       dp.Device,
			Timestamp:    dp.Timestamp,
			TimestampISO: time.Unix(dp.Timestamp, 0).UTC().Format(time.RFC3339),
			Fields:       dp.Fields,
		}
		if dp.Value != nil {
			td.Value = *dp.Value
		}
		if td.Fields == nil {
			td.Fields = map[string]float64{}
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, td); err != nil {
			return nil, fmt.Errorf("template render error: %w", err)
		}

		out := buf.Bytes()
		if !json.Valid(out) {
			return nil, fmt.Errorf("rendered template is not valid JSON: %s", out)
		}
		return out, nil
	}
}
