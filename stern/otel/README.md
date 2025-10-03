# OpenTelemetry Integration for Stern

This package provides OpenTelemetry (OTel) export functionality for stern, allowing Kubernetes pod logs to be sent directly to an OpenTelemetry collector.

## Features

- **OTLP Export**: Supports both gRPC and HTTP protocols
- **Batch Processing**: Efficient batching of log records
- **Structured Log Parsing**: Automatically detects and parses JSON logs (Zap, Logrus, etc.)
- **K8s Semantic Conventions**: Follows OpenTelemetry semantic conventions for Kubernetes resources
- **Rich Metadata**: Includes pod labels, annotations, namespace, node, and container information
- **Timestamp Preservation**: Maintains original log timestamps from Kubernetes
- **Graceful Shutdown**: Ensures all logs are flushed on exit

## Usage

### Basic Example

```bash
# Export logs to OTel collector via gRPC (default)
stern my-app -o otel

# Export logs via HTTP
stern my-app -o otel --otel-protocol=http --otel-endpoint=localhost:4318

# Use secure TLS connection
stern my-app -o otel --otel-endpoint=collector.example.com:4317 --otel-insecure=false
```

### Configuration Options

Enable OTel export by setting `--output=otel` (or `-o otel`). The following flags configure the exporter:

| Flag | Default | Description |
|------|---------|-------------|
| `--output`, `-o` | `default` | Set to `otel` to enable OpenTelemetry log export |
| `--otel-endpoint` | `localhost:4317` | OpenTelemetry collector endpoint |
| `--otel-protocol` | `grpc` | Protocol to use (`grpc` or `http`) |
| `--otel-insecure` | `true` | Use insecure connection (no TLS) |
| `--otel-batch-size` | `512` | Maximum batch size for log export |
| `--otel-export-timeout` | `30s` | Timeout for export operations |

### Environment Variables

The OTel SDK also respects standard OpenTelemetry environment variables:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_HEADERS="api-key=secret123"
export OTEL_RESOURCE_ATTRIBUTES="service.name=stern,service.version=1.0"

stern my-app -o otel
```

## Log Record Structure

Each log record exported to OpenTelemetry includes:

### Body
- For plain text logs: The actual log message from the container
- For JSON logs: The extracted `msg` or `message` field

### Severity
- Automatically mapped from `level`/`severity` field in JSON logs (DEBUG, INFO, WARN, ERROR, FATAL)

### Timestamp
- Original timestamp from Kubernetes (preserved from pod logs)

### Structured Log Parsing

Stern automatically detects and parses structured JSON logs from popular frameworks like Zap, Logrus, and Bunyan:

**Input (JSON log from Zap)**:
```json
{"level":"info","ts":"2025-10-03T15:04:36.479Z","caller":"api/server.go:123","msg":"Request processed","user_id":12345,"duration_ms":45}
```

**Output to OTel**:
- **Body**: `"Request processed"`
- **Severity**: `INFO`
- **Attributes**: `ts`, `caller`, `user_id`, `duration_ms` (plus all K8s attributes below)

### Attributes (K8s Semantic Conventions)

All logs include these Kubernetes-specific attributes:

| Attribute | Example | Description |
|-----------|---------|-------------|
| `service.name` | `my-app` | Derived from pod labels (app.kubernetes.io/name, app, or k8s-app) |
| `host.name` | `node-1` | Node where pod is running |
| `k8s.namespace.name` | `default` | Kubernetes namespace |
| `k8s.pod.name` | `my-app-7d8f9c-xyz` | Pod name |
| `k8s.container.name` | `app` | Container name |
| `k8s.node.name` | `node-1` | Node where pod is running |
| `k8s.pod.label.<key>` | `k8s.pod.label.app=my-app` | Pod labels (all labels) |
| `k8s.pod.annotation.<key>` | `k8s.pod.annotation.version=1.0` | Pod annotations (all annotations) |

Plus any additional fields from structured JSON logs.

### Resource Attributes

| Attribute | Example | Description |
|-----------|---------|-------------|
| `service.name` | `stern` | Service identifier |
| `k8s.cluster.name` | `production` | Cluster context from kubeconfig |

## Example with OpenTelemetry Collector

### 1. Start the Collector

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024

exporters:
  logging:
    loglevel: debug
  file:
    path: /tmp/stern-logs.json

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging, file]
```

```bash
otelcol --config=otel-collector-config.yaml
```

### 2. Run Stern with OTel

```bash
stern . --all-namespaces -o otel
```

### 3. Query Logs

Logs will be exported to the collector and can be:
- Forwarded to any OTLP-compatible backend (Jaeger, Grafana Loki, Datadog, etc.)
- Stored in files for analysis
- Processed with additional metadata enrichment

## Integration Examples

### With Grafana Loki

```yaml
exporters:
  loki:
    endpoint: http://loki:3100/loki/api/v1/push
    labels:
      resource:
        k8s.namespace.name: "namespace"
        k8s.pod.name: "pod"
      attributes:
        k8s.container.name: "container"
```

### With Jaeger (for trace correlation)

```yaml
exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true
```

### With Elasticsearch

```yaml
exporters:
  elasticsearch:
    endpoints: [http://elasticsearch:9200]
    logs_index: stern-logs
```

## Architecture

```
Kubernetes Pod Logs
        ↓
    Stern Tail
        ↓
  OTel Transformer
        ↓
  Batch Processor
        ↓
   OTLP Exporter
        ↓
  OTel Collector
        ↓
   Backend (Loki, Jaeger, etc.)
```

## Performance Considerations

- **Batch Size**: Increase `--otel-batch-size` for high-volume scenarios
- **Network**: Use gRPC for better performance than HTTP
- **Buffering**: The batch processor queues logs, preventing backpressure
- **Graceful Shutdown**: Stern waits up to 30 seconds to flush pending logs on exit

## Troubleshooting

### Connection Refused

```bash
# Check if collector is running
curl http://localhost:4318/v1/logs

# Use insecure mode for local testing
stern . -o otel --otel-insecure
```

### No Logs Appearing

```bash
# Enable verbose logging in stern
stern . -o otel --verbosity=6

# Check collector logs for errors
docker logs otel-collector
```

### TLS Errors

```bash
# Disable TLS for testing
stern . -o otel --otel-insecure=true

# For production, ensure valid certificates
stern . -o otel --otel-insecure=false --otel-endpoint=collector.prod:4317
```

## Development

### Running Tests

```bash
go test ./stern/otel/...
```

### Adding Custom Attributes

Modify `transformer.go` to add additional attributes:

```go
attrs = append(attrs, log.String("custom.attribute", "value"))
```

## References

- [OpenTelemetry Logs Specification](https://opentelemetry.io/docs/specs/otel/logs/)
- [OTLP Protocol](https://opentelemetry.io/docs/specs/otlp/)
- [K8s Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/resource/k8s/)
