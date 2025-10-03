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
	"testing"
	"time"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// mockLogRecordExporter is a simple exporter for testing
type mockLogRecordExporter struct {
	records []sdklog.Record
}

func (m *mockLogRecordExporter) Export(ctx context.Context, records []sdklog.Record) error {
	m.records = append(m.records, records...)
	return nil
}

func (m *mockLogRecordExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockLogRecordExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func TestEmitLog(t *testing.T) {
	mockExporter := &mockLogRecordExporter{}
	processor := sdklog.NewSimpleProcessor(mockExporter)
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(processor))
	logger := provider.Logger("test")

	record := &LogRecord{
		Timestamp:     time.Now(),
		Body:          "test log message",
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "test-container",
		NodeName:      "test-node",
		Labels: map[string]string{
			"app": "test",
		},
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	}

	EmitLog(context.Background(), logger, record)

	// Force flush to ensure the record is exported
	provider.ForceFlush(context.Background())

	if len(mockExporter.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mockExporter.records))
	}

	exportedRecord := mockExporter.records[0]
	if exportedRecord.Body().String() != record.Body {
		t.Errorf("expected body %q, got %q", record.Body, exportedRecord.Body().String())
	}
}

func TestLogRecordAttributes(t *testing.T) {
	mockExporter := &mockLogRecordExporter{}
	processor := sdklog.NewSimpleProcessor(mockExporter)
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(processor))
	logger := provider.Logger("test")

	record := &LogRecord{
		Timestamp:     time.Now(),
		Body:          "test message",
		Namespace:     "kube-system",
		PodName:       "coredns-abc123",
		ContainerName: "coredns",
		NodeName:      "node-1",
		Labels: map[string]string{
			"k8s-app": "kube-dns",
		},
		Annotations: map[string]string{},
	}

	EmitLog(context.Background(), logger, record)
	provider.ForceFlush(context.Background())

	if len(mockExporter.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mockExporter.records))
	}

	exportedRecord := mockExporter.records[0]

	// Check that attributes are set correctly
	var foundNamespace, foundPod, foundContainer, foundNode, foundService, foundHost bool
	exportedRecord.WalkAttributes(func(kv log.KeyValue) bool {
		switch kv.Key {
		case "k8s.namespace.name":
			if kv.Value.AsString() == "kube-system" {
				foundNamespace = true
			}
		case "k8s.pod.name":
			if kv.Value.AsString() == "coredns-abc123" {
				foundPod = true
			}
		case "k8s.container.name":
			if kv.Value.AsString() == "coredns" {
				foundContainer = true
			}
		case "k8s.node.name":
			if kv.Value.AsString() == "node-1" {
				foundNode = true
			}
		case "service.name":
			if kv.Value.AsString() == "kube-dns" {
				foundService = true
			}
		case "host.name":
			if kv.Value.AsString() == "node-1" {
				foundHost = true
			}
		}
		return true
	})

	if !foundNamespace {
		t.Error("namespace attribute not found or incorrect")
	}
	if !foundPod {
		t.Error("pod name attribute not found or incorrect")
	}
	if !foundContainer {
		t.Error("container name attribute not found or incorrect")
	}
	if !foundNode {
		t.Error("node name attribute not found or incorrect")
	}
	if !foundService {
		t.Error("service.name attribute not found or incorrect (should be 'kube-dns' from k8s-app label)")
	}
	if !foundHost {
		t.Error("host.name attribute not found or incorrect")
	}
}

func TestDeriveServiceName(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		podName  string
		expected string
	}{
		{
			name: "app.kubernetes.io/name label",
			labels: map[string]string{
				"app.kubernetes.io/name": "my-service",
				"app":                    "fallback",
			},
			podName:  "my-pod-123",
			expected: "my-service",
		},
		{
			name: "app label",
			labels: map[string]string{
				"app":     "my-app",
				"k8s-app": "fallback",
			},
			podName:  "my-pod-456",
			expected: "my-app",
		},
		{
			name: "k8s-app label",
			labels: map[string]string{
				"k8s-app": "kube-dns",
			},
			podName:  "coredns-abc123",
			expected: "kube-dns",
		},
		{
			name: "no service labels - fallback to pod name",
			labels: map[string]string{
				"version": "v1.0",
			},
			podName:  "standalone-pod-789",
			expected: "standalone-pod-789",
		},
		{
			name:     "empty labels - fallback to pod name",
			labels:   map[string]string{},
			podName:  "test-pod",
			expected: "test-pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveServiceName(tt.labels, tt.podName)
			if result != tt.expected {
				t.Errorf("deriveServiceName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestParseStructuredLog(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		expectedMessage    string
		expectedSeverity   string
		expectedStructured bool
		checkAttrs         func(t *testing.T, attrs map[string]interface{})
	}{
		{
			name:               "Zap JSON log with all fields",
			body:               `{"level":"debug","ts":"2025-10-03T20:04:36.479Z","logger":"statler.server.boho-api","caller":"pylim/impl.go:370","msg":"Polling job status","resource":{"service.instance.id":"80866d5e-2c67-46d5-8686-a8dcf0aea518","service.name":"aibutter","service.version":"v0.2.0-2-g508f03417594203981"},"otelcol.component.id":"statler","otelcol.component.kind":"exporter","job_id":"666887f85-7131-91a6-43cb-adff-2f13f4c39e20-1759488300-1759488360"}`,
			expectedMessage:    "Polling job status",
			expectedSeverity:   "DEBUG",
			expectedStructured: true,
			checkAttrs: func(t *testing.T, attrs map[string]interface{}) {
				if ts, ok := attrs["ts"].(string); !ok || ts != "2025-10-03T20:04:36.479Z" {
					t.Errorf("expected ts='2025-10-03T20:04:36.479Z', got %v", attrs["ts"])
				}
				if logger, ok := attrs["logger"].(string); !ok || logger != "statler.server.boho-api" {
					t.Errorf("expected logger='statler.server.boho-api', got %v", attrs["logger"])
				}
				if caller, ok := attrs["caller"].(string); !ok || caller != "pylim/impl.go:370" {
					t.Errorf("expected caller='pylim/impl.go:370', got %v", attrs["caller"])
				}
				if _, ok := attrs["resource"]; !ok {
					t.Error("expected resource field to be present")
				}
			},
		},
		{
			name:               "Simple Zap log with msg",
			body:               `{"level":"info","msg":"Server started"}`,
			expectedMessage:    "Server started",
			expectedSeverity:   "INFO",
			expectedStructured: true,
			checkAttrs: func(t *testing.T, attrs map[string]interface{}) {
				// level and msg should be removed, no other fields expected
				if len(attrs) != 0 {
					t.Errorf("expected no additional attributes, got %v", attrs)
				}
			},
		},
		{
			name:               "Non-JSON plain text log",
			body:               "This is a plain text log message",
			expectedMessage:    "This is a plain text log message",
			expectedSeverity:   "",
			expectedStructured: false,
			checkAttrs: func(t *testing.T, attrs map[string]interface{}) {
				if attrs != nil {
					t.Errorf("expected nil attrs for plain text, got %v", attrs)
				}
			},
		},
		{
			name:               "JSON with message field instead of msg",
			body:               `{"level":"error","message":"Database connection failed","error":"connection timeout"}`,
			expectedMessage:    "Database connection failed",
			expectedSeverity:   "ERROR",
			expectedStructured: true,
			checkAttrs: func(t *testing.T, attrs map[string]interface{}) {
				if err, ok := attrs["error"].(string); !ok || err != "connection timeout" {
					t.Errorf("expected error='connection timeout', got %v", attrs["error"])
				}
			},
		},
		{
			name:               "JSON with warning level",
			body:               `{"level":"warn","msg":"High memory usage","memory_mb":1024}`,
			expectedMessage:    "High memory usage",
			expectedSeverity:   "WARN",
			expectedStructured: true,
			checkAttrs: func(t *testing.T, attrs map[string]interface{}) {
				if mem, ok := attrs["memory_mb"].(float64); !ok || mem != 1024 {
					t.Errorf("expected memory_mb=1024, got %v", attrs["memory_mb"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, severity, attrs, isStructured := parseStructuredLog(tt.body)

			if message != tt.expectedMessage {
				t.Errorf("message = %q, expected %q", message, tt.expectedMessage)
			}
			if severity != tt.expectedSeverity {
				t.Errorf("severity = %q, expected %q", severity, tt.expectedSeverity)
			}
			if isStructured != tt.expectedStructured {
				t.Errorf("isStructured = %v, expected %v", isStructured, tt.expectedStructured)
			}
			if tt.checkAttrs != nil {
				tt.checkAttrs(t, attrs)
			}
		})
	}
}

func TestMapSeverityToOTel(t *testing.T) {
	tests := []struct {
		input    string
		expected log.Severity
	}{
		{"DEBUG", log.SeverityDebug},
		{"debug", log.SeverityDebug},
		{"INFO", log.SeverityInfo},
		{"info", log.SeverityInfo},
		{"WARN", log.SeverityWarn},
		{"WARNING", log.SeverityWarn},
		{"warn", log.SeverityWarn},
		{"ERROR", log.SeverityError},
		{"error", log.SeverityError},
		{"FATAL", log.SeverityFatal},
		{"CRITICAL", log.SeverityFatal},
		{"unknown", log.SeverityUndefined},
		{"", log.SeverityUndefined},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapSeverityToOTel(tt.input)
			if result != tt.expected {
				t.Errorf("mapSeverityToOTel(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEmitStructuredLog(t *testing.T) {
	mockExporter := &mockLogRecordExporter{}
	processor := sdklog.NewSimpleProcessor(mockExporter)
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(processor))
	logger := provider.Logger("test")

	record := &LogRecord{
		Timestamp:     time.Now(),
		Body:          `{"level":"info","msg":"Test message","user_id":12345,"action":"login"}`,
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "test-container",
		NodeName:      "test-node",
		Labels:        map[string]string{"app": "test"},
		Annotations:   map[string]string{},
	}

	EmitLog(context.Background(), logger, record)
	provider.ForceFlush(context.Background())

	if len(mockExporter.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mockExporter.records))
	}

	exportedRecord := mockExporter.records[0]

	// Check that body is the extracted message, not the full JSON
	if exportedRecord.Body().String() != "Test message" {
		t.Errorf("expected body 'Test message', got %q", exportedRecord.Body().String())
	}

	// Check that severity was set
	if exportedRecord.Severity() != log.SeverityInfo {
		t.Errorf("expected severity Info, got %v", exportedRecord.Severity())
	}

	// Check that structured fields were added as attributes
	var foundUserId, foundAction bool
	exportedRecord.WalkAttributes(func(kv log.KeyValue) bool {
		switch kv.Key {
		case "user_id":
			// JSON numbers are parsed as float64
			if kv.Value.AsFloat64() == 12345.0 {
				foundUserId = true
			}
		case "action":
			if kv.Value.AsString() == "login" {
				foundAction = true
			}
		}
		return true
	})

	if !foundUserId {
		t.Error("user_id attribute not found or incorrect")
	}
	if !foundAction {
		t.Error("action attribute not found or incorrect")
	}
}
