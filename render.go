package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// templateData is the value passed to a payload template on each render.
type templateData struct {
	Device    string
	Timestamp int64
	Value     float64
	Fields    map[string]float64
}

// payloadTmpl is non-nil when a payload template has been configured.
var payloadTmpl *template.Template

// initTemplate parses tmplStr and stores it for use by renderPayload.
func initTemplate(tmplStr string) error {
	t, err := template.New("payload").Parse(tmplStr)
	if err != nil {
		return err
	}
	payloadTmpl = t
	return nil
}

// renderPayload serialises dp either through the configured template or as JSON.
// When a template is active the rendered output is validated as JSON before returning.
func renderPayload(dp DataPoint) ([]byte, error) {
	if payloadTmpl == nil {
		return json.Marshal(dp)
	}

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
	if err := payloadTmpl.Execute(&buf, td); err != nil {
		return nil, fmt.Errorf("template render error: %w", err)
	}

	out := buf.Bytes()
	if !json.Valid(out) {
		return nil, fmt.Errorf("rendered template is not valid JSON: %s", out)
	}
	return out, nil
}
