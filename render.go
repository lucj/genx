package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// Renderer serialises a DataPoint to bytes for a sink to transmit.
type Renderer func(DataPoint) ([]byte, error)

// JSONRenderer is the default renderer — plain JSON marshalling.
func JSONRenderer(dp DataPoint) ([]byte, error) {
	return json.Marshal(dp)
}

// templateData is the value passed to a payload template on each render.
type templateData struct {
	Device    string
	Timestamp int64
	Value     float64
	Fields    map[string]float64
}

// NewTemplateRenderer returns a Renderer that executes tmpl and validates the
// output as JSON before returning it.
func NewTemplateRenderer(tmpl *template.Template) Renderer {
	return func(dp DataPoint) ([]byte, error) {
		td := templateData{
			Device:    dp.Device,
			Timestamp: dp.Timestamp,
			Fields:    dp.Fields,
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
