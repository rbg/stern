//   Copyright 2025 Robert B Gordon <rbg@openrbg.com>
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ExporterConfig holds configuration for the OTel exporter
type ExporterConfig struct {
	Endpoint      string
	Protocol      string // "grpc" or "http"
	Insecure      bool
	BatchSize     int
	ExportTimeout time.Duration
	Headers       map[string]string
}

// Exporter wraps the OTel SDK components
type Exporter struct {
	loggerProvider *sdklog.LoggerProvider
	logger         log.Logger
	config         *ExporterConfig
}

// NewExporter creates a new OTel exporter with the given configuration
func NewExporter(ctx context.Context, config *ExporterConfig, res *resource.Resource) (*Exporter, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("OTel endpoint is required")
	}

	var logExporter sdklog.Exporter
	var err error

	switch config.Protocol {
	case "grpc":
		logExporter, err = newGRPCExporter(ctx, config)
	case "http":
		logExporter, err = newHTTPExporter(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s (must be 'grpc' or 'http')", config.Protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create OTel log exporter: %w", err)
	}

	// Create batch processor
	batchProcessor := sdklog.NewBatchProcessor(
		logExporter,
		sdklog.WithMaxQueueSize(config.BatchSize*2),
		sdklog.WithExportMaxBatchSize(config.BatchSize),
		sdklog.WithExportTimeout(config.ExportTimeout),
	)

	// Create logger provider
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(batchProcessor),
	)

	logger := loggerProvider.Logger("stern")

	return &Exporter{
		loggerProvider: loggerProvider,
		logger:         logger,
		config:         config,
	}, nil
}

// newGRPCExporter creates a gRPC OTLP log exporter
func newGRPCExporter(ctx context.Context, config *ExporterConfig) (sdklog.Exporter, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(config.Endpoint),
	}

	if config.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
		opts = append(opts, otlploggrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	if len(config.Headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(config.Headers))
	}

	return otlploggrpc.New(ctx, opts...)
}

// newHTTPExporter creates an HTTP OTLP log exporter
func newHTTPExporter(ctx context.Context, config *ExporterConfig) (sdklog.Exporter, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(config.Endpoint),
	}

	if config.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	if len(config.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(config.Headers))
	}

	return otlploghttp.New(ctx, opts...)
}

// Logger returns the OTel logger instance
func (e *Exporter) Logger() log.Logger {
	return e.logger
}

// Shutdown gracefully shuts down the exporter, flushing any pending logs
func (e *Exporter) Shutdown(ctx context.Context) error {
	if e.loggerProvider != nil {
		return e.loggerProvider.Shutdown(ctx)
	}
	return nil
}

// ForceFlush immediately exports all pending logs
func (e *Exporter) ForceFlush(ctx context.Context) error {
	if e.loggerProvider != nil {
		return e.loggerProvider.ForceFlush(ctx)
	}
	return nil
}
