package main

import (
	"bytes"
	crand "crypto/rand"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// NewCSVRenderer returns a Renderer that emits comma-separated values.
// The header row is written once (on the first call) and prepended to that
// call's output. Subsequent calls return only the data row.
//
// Single-field mode columns: device, timestamp, value
// Multi-field mode columns:  device, timestamp, <field names sorted A-Z>
//
// Compatible with --iso-time: when isoTime is true the timestamp column
// contains an ISO 8601 UTC string instead of a Unix epoch integer.
func NewCSVRenderer(isoTime bool) Renderer {
	var once sync.Once
	var fieldNames []string

	formatTS := func(ts int64) string {
		if isoTime {
			return time.Unix(ts, 0).UTC().Format(time.RFC3339)
		}
		return strconv.FormatInt(ts, 10)
	}

	return func(dp DataPoint) ([]byte, error) {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)

		if len(dp.Fields) > 0 {
			once.Do(func() {
				fieldNames = make([]string, 0, len(dp.Fields))
				for k := range dp.Fields {
					fieldNames = append(fieldNames, k)
				}
				sort.Strings(fieldNames)
				header := append([]string{"device", "timestamp"}, fieldNames...)
				_ = w.Write(header)
			})
			row := make([]string, 0, 2+len(fieldNames))
			row = append(row, dp.Device, formatTS(dp.Timestamp))
			for _, name := range fieldNames {
				row = append(row, strconv.FormatFloat(dp.Fields[name], 'f', -1, 64))
			}
			_ = w.Write(row)
		} else {
			once.Do(func() {
				_ = w.Write([]string{"device", "timestamp", "value"})
			})
			val := ""
			if dp.Value != nil {
				val = strconv.FormatFloat(*dp.Value, 'f', -1, 64)
			}
			_ = w.Write([]string{dp.Device, formatTS(dp.Timestamp), val})
		}

		w.Flush()
		if err := w.Error(); err != nil {
			return nil, err
		}
		// csv.Writer appends \n after each record; trim the trailing one since
		// sinks add their own line ending.
		return bytes.TrimRight(buf.Bytes(), "\n"), nil
	}
}

// NewInfluxRenderer returns a Renderer that emits InfluxDB line protocol.
//
// Format: <measurement>,device=<device> <field>=<value>[,...] <unix_nanoseconds>
//
// Single-field DataPoints use "value" as the field key. Multi-field DataPoints
// use the sorted field names from Fields. Special characters in the measurement
// name, tag values, and field keys are escaped per the line-protocol spec.
func NewInfluxRenderer(measurement string) Renderer {
	if measurement == "" {
		measurement = "genx"
	}
	escapeMeasurement := func(s string) string {
		s = strings.ReplaceAll(s, ",", `\,`)
		s = strings.ReplaceAll(s, " ", `\ `)
		return s
	}
	escapeTag := func(s string) string {
		s = strings.ReplaceAll(s, ",", `\,`)
		s = strings.ReplaceAll(s, "=", `\=`)
		s = strings.ReplaceAll(s, " ", `\ `)
		return s
	}
	m := escapeMeasurement(measurement)
	return func(dp DataPoint) ([]byte, error) {
		tag := escapeTag(dp.Device)
		tsNano := dp.Timestamp * int64(time.Second)

		var fieldSet string
		if len(dp.Fields) > 0 {
			names := make([]string, 0, len(dp.Fields))
			for k := range dp.Fields {
				names = append(names, k)
			}
			sort.Strings(names)
			parts := make([]string, len(names))
			for i, name := range names {
				parts[i] = escapeTag(name) + "=" + strconv.FormatFloat(dp.Fields[name], 'f', -1, 64)
			}
			fieldSet = strings.Join(parts, ",")
		} else {
			val := 0.0
			if dp.Value != nil {
				val = *dp.Value
			}
			fieldSet = "value=" + strconv.FormatFloat(val, 'f', -1, 64)
		}

		return []byte(fmt.Sprintf("%s,device=%s %s %d", m, tag, fieldSet, tsNano)), nil
	}
}

// NewCloudEventRenderer returns a Renderer that wraps each DataPoint in a
// CloudEvents 1.0 structured-content-mode JSON envelope.
//
//	source   URI-reference identifying the event producer (e.g. "/myapp").
//	          The device name is appended as a path segment automatically.
//	eventType Reverse-DNS event type (e.g. "io.genx.measurement").
//
// The rendered bytes should be sent with Content-Type application/cloudevents+json.
func NewCloudEventRenderer(source, eventType string) Renderer {
	if source == "" {
		source = "/genx"
	}
	if eventType == "" {
		eventType = "io.genx.measurement"
	}
	return func(dp DataPoint) ([]byte, error) {
		data, err := json.Marshal(dp)
		if err != nil {
			return nil, err
		}
		type cloudEvent struct {
			SpecVersion     string          `json:"specversion"`
			ID              string          `json:"id"`
			Source          string          `json:"source"`
			Type            string          `json:"type"`
			Time            string          `json:"time"`
			DataContentType string          `json:"datacontenttype"`
			Data            json.RawMessage `json:"data"`
		}
		ce := cloudEvent{
			SpecVersion:     "1.0",
			ID:              newEventID(),
			Source:          source + "/" + dp.Device,
			Type:            eventType,
			Time:            time.Unix(dp.Timestamp, 0).UTC().Format(time.RFC3339),
			DataContentType: "application/json",
			Data:            json.RawMessage(data),
		}
		return json.Marshal(ce)
	}
}

// newEventID returns a random UUID v4 string using crypto/rand.
func newEventID() string {
	var b [16]byte
	_, _ = crand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
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
