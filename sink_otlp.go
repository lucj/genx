package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// OTLPSink pushes metrics to an OpenTelemetry collector via OTLP gRPC or HTTP.
// Each DataPoint field becomes a Float64Gauge; instruments are created lazily
// and cached. The PeriodicReader exports every second; Shutdown flushes on Close.
type OTLPSink struct {
	provider   *sdkmetric.MeterProvider
	meter      metric.Meter
	gaugesMu   sync.Mutex
	gauges     map[string]metric.Float64Gauge
	metricName string
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewOTLPSink(endpoint string, useHTTP bool, headers map[string]string, insecure bool, metricName string) (*OTLPSink, error) {
	if metricName == "" {
		metricName = "genx"
	}

	ctx := context.Background()

	var exp sdkmetric.Exporter
	var err error

	if useHTTP {
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(headers))
		}
		exp, err = otlpmetrichttp.New(ctx, opts...)
	} else {
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(headers))
		}
		exp, err = otlpmetricgrpc.New(ctx, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("genx"),
	)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(time.Second)),
		),
	)

	sinkCtx, cancel := context.WithCancel(context.Background())
	return &OTLPSink{
		provider:   provider,
		meter:      provider.Meter("genx"),
		gauges:     make(map[string]metric.Float64Gauge),
		metricName: metricName,
		ctx:        sinkCtx,
		cancel:     cancel,
	}, nil
}

func (s *OTLPSink) getOrCreate(name string) (metric.Float64Gauge, error) {
	s.gaugesMu.Lock()
	defer s.gaugesMu.Unlock()
	if g, ok := s.gauges[name]; ok {
		return g, nil
	}
	g, err := s.meter.Float64Gauge(name)
	if err != nil {
		return nil, err
	}
	s.gauges[name] = g
	return g, nil
}

func (s *OTLPSink) Send(dp DataPoint) error {
	attrs := metric.WithAttributes(attribute.String("device", dp.Device))

	if len(dp.Fields) > 0 {
		for name, val := range dp.Fields {
			g, err := s.getOrCreate(s.metricName + "." + name)
			if err != nil {
				return err
			}
			g.Record(s.ctx, val, attrs)
		}
	} else if dp.Value != nil {
		g, err := s.getOrCreate(s.metricName)
		if err != nil {
			return err
		}
		g.Record(s.ctx, *dp.Value, attrs)
	}
	return nil
}

func (s *OTLPSink) Close() error {
	s.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.provider.Shutdown(ctx)
}
