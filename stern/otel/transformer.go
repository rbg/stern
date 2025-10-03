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
	"encoding/json"
	"strings"
	"time"

	"go.opentelemetry.io/otel/log"
)

// LogRecord represents a log entry with metadata
type LogRecord struct {
	Timestamp     time.Time
	Body          string
	Namespace     string
	PodName       string
	ContainerName string
	NodeName      string
	Labels        map[string]string
	Annotations   map[string]string
}

// deriveServiceName extracts service name from pod labels or falls back to pod name
func deriveServiceName(labels map[string]string, podName string) string {
	// Try standard Kubernetes service name labels in order of preference
	if serviceName, ok := labels["app.kubernetes.io/name"]; ok && serviceName != "" {
		return serviceName
	}
	if serviceName, ok := labels["app"]; ok && serviceName != "" {
		return serviceName
	}
	if serviceName, ok := labels["k8s-app"]; ok && serviceName != "" {
		return serviceName
	}
	// Fall back to pod name if no service label is found
	return podName
}

// parseStructuredLog attempts to parse the log body as JSON and extract structured fields
func parseStructuredLog(body string) (message string, severity string, structuredAttrs map[string]interface{}, isStructured bool) {
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, "{") {
		return body, "", nil, false
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body, "", nil, false
	}

	// Extract common logging fields
	// Try various common message field names
	for _, key := range []string{"msg", "message", "Message"} {
		if val, ok := parsed[key]; ok {
			if strVal, ok := val.(string); ok {
				message = strVal
				delete(parsed, key)
				break
			}
		}
	}

	// Extract severity/level
	for _, key := range []string{"level", "severity", "levelname"} {
		if val, ok := parsed[key]; ok {
			if strVal, ok := val.(string); ok {
				severity = strings.ToUpper(strVal)
				delete(parsed, key)
				break
			}
		}
	}

	// If we couldn't extract a message, use the whole JSON as the body
	if message == "" {
		message = body
	}

	return message, severity, parsed, true
}

// convertToLogKeyValue converts a Go value to an OTel log.Value
func convertToLogKeyValue(v interface{}) log.Value {
	switch val := v.(type) {
	case string:
		return log.StringValue(val)
	case float64:
		return log.Float64Value(val)
	case int:
		return log.Int64Value(int64(val))
	case int64:
		return log.Int64Value(val)
	case bool:
		return log.BoolValue(val)
	case map[string]interface{}:
		// For nested objects, convert to JSON string
		if jsonBytes, err := json.Marshal(val); err == nil {
			return log.StringValue(string(jsonBytes))
		}
		return log.StringValue("")
	case []interface{}:
		// For arrays, convert to JSON string
		if jsonBytes, err := json.Marshal(val); err == nil {
			return log.StringValue(string(jsonBytes))
		}
		return log.StringValue("")
	default:
		// Fallback: convert to string
		return log.StringValue("")
	}
}

// mapSeverityToOTel maps common log levels to OTel severity
func mapSeverityToOTel(severity string) log.Severity {
	switch strings.ToUpper(severity) {
	case "DEBUG":
		return log.SeverityDebug
	case "INFO":
		return log.SeverityInfo
	case "WARN", "WARNING":
		return log.SeverityWarn
	case "ERROR":
		return log.SeverityError
	case "FATAL", "CRITICAL":
		return log.SeverityFatal
	default:
		return log.SeverityUndefined
	}
}

// EmitLog emits a log record to the OTel logger with proper attributes
func EmitLog(ctx context.Context, logger log.Logger, record *LogRecord) {
	// Try to parse structured logs
	message, severity, structuredAttrs, isStructured := parseStructuredLog(record.Body)

	// Build log record with K8s semantic conventions
	var attrs []log.KeyValue

	// Service and host attributes (resource-level semantic conventions)
	// https://opentelemetry.io/docs/specs/semconv/resource/
	serviceName := deriveServiceName(record.Labels, record.PodName)
	attrs = append(attrs, log.String("service.name", serviceName))

	if record.NodeName != "" {
		attrs = append(attrs, log.String("host.name", record.NodeName))
	}

	// Core K8s attributes following semantic conventions
	// https://opentelemetry.io/docs/specs/semconv/resource/k8s/
	if record.Namespace != "" {
		attrs = append(attrs, log.String("k8s.namespace.name", record.Namespace))
	}
	if record.PodName != "" {
		attrs = append(attrs, log.String("k8s.pod.name", record.PodName))
	}
	if record.ContainerName != "" {
		attrs = append(attrs, log.String("k8s.container.name", record.ContainerName))
	}
	if record.NodeName != "" {
		attrs = append(attrs, log.String("k8s.node.name", record.NodeName))
	}

	// Add pod labels as attributes with prefix
	for key, value := range record.Labels {
		attrs = append(attrs, log.String("k8s.pod.label."+key, value))
	}

	// Add pod annotations as attributes with prefix
	for key, value := range record.Annotations {
		attrs = append(attrs, log.String("k8s.pod.annotation."+key, value))
	}

	// Add structured log fields as attributes
	if isStructured {
		for key, value := range structuredAttrs {
			attrs = append(attrs, log.KeyValue{
				Key:   key,
				Value: convertToLogKeyValue(value),
			})
		}
	}

	// Create and emit the log record using the builder pattern
	logRecord := log.Record{}
	logRecord.SetTimestamp(record.Timestamp)
	logRecord.SetObservedTimestamp(time.Now())
	logRecord.SetBody(log.StringValue(message))

	// Set severity if extracted from structured log
	if severity != "" {
		logRecord.SetSeverity(mapSeverityToOTel(severity))
	}

	logRecord.AddAttributes(attrs...)

	logger.Emit(ctx, logRecord)
}
