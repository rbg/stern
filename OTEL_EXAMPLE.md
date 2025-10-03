# OpenTelemetry Integration Example

This example demonstrates how to export Kubernetes pod logs from stern to an OpenTelemetry collector.

## Quick Start

### 1. Start OpenTelemetry Collector

Create a configuration file:

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

  # Add resource attributes
  resource:
    attributes:
      - key: deployment.environment
        value: development
        action: insert

exporters:
  # Console output for debugging
  logging:
    loglevel: info

  # File export
  file:
    path: /tmp/stern-logs.json
    rotation:
      max_megabytes: 10
      max_backups: 3

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [logging, file]
```

Start the collector:

```bash
docker run -d --name otel-collector \
  -p 4317:4317 \
  -p 4318:4318 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otel/config.yaml \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel/config.yaml
```

### 2. Run Stern with OTel Export

```bash
# Basic usage
stern . --all-namespaces --otel-enabled

# With custom endpoint
stern my-app --otel-enabled --otel-endpoint=collector:4317

# Using HTTP instead of gRPC
stern my-app --otel-enabled --otel-protocol=http --otel-endpoint=localhost:4318

# With custom batch size for high volume
stern . -A --otel-enabled --otel-batch-size=2048
```

### 3. View Exported Logs

```bash
# Check collector logs
docker logs -f otel-collector

# View exported file
tail -f /tmp/stern-logs.json | jq '.'
```

## Advanced Example: Full Observability Stack

### Docker Compose Setup

```yaml
# docker-compose.yaml
version: '3.8'

services:
  # OpenTelemetry Collector
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel/config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel/config.yaml
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "8888:8888"   # Prometheus metrics
      - "13133:13133" # Health check

  # Grafana Loki for log storage
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml

  # Grafana for visualization
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    volumes:
      - ./grafana-datasources.yaml:/etc/grafana/provisioning/datasources/datasources.yaml

  # Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yaml:/etc/prometheus/prometheus.yml
```

### OTel Collector Configuration with Loki

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

  # Transform attributes for Loki
  attributes:
    actions:
      - key: loki.attribute.labels
        action: insert
        value: k8s.namespace.name, k8s.pod.name, k8s.container.name

exporters:
  loki:
    endpoint: http://loki:3100/loki/api/v1/push
    labels:
      resource:
        k8s.namespace.name: "namespace"
        k8s.pod.name: "pod"
        k8s.container.name: "container"
      attributes:
        k8s.node.name: "node"

  prometheusremotewrite:
    endpoint: http://prometheus:9090/api/v1/write

  logging:
    loglevel: debug

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch, attributes]
      exporters: [loki, logging]
```

### Grafana Data Source

```yaml
# grafana-datasources.yaml
apiVersion: 1

datasources:
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    isDefault: true
```

### Running the Stack

```bash
# Start all services
docker-compose up -d

# Run stern with OTel
stern . --all-namespaces --otel-enabled --otel-endpoint=localhost:4317

# Open Grafana
open http://localhost:3000

# Query logs in Grafana Explore:
# {namespace="default", pod=~"my-app.*"}
```

## Example: Filtering and Enrichment

### Filter Logs Before Export

```bash
# Only export ERROR logs
stern my-app --otel-enabled --include="ERROR"

# Exclude noise
stern my-app --otel-enabled --exclude="health check"

# Specific containers
stern my-app --otel-enabled -c="^app$" -E="istio-proxy"
```

### Collector-Side Processing

```yaml
processors:
  # Add custom attributes
  resource:
    attributes:
      - key: environment
        value: production
        action: insert
      - key: team
        value: platform
        action: insert

  # Filter by log level
  filter:
    logs:
      include:
        match_type: regexp
        bodies:
          - 'level=(error|warn|fatal)'

  # Extract structured fields from JSON logs
  transform:
    log_statements:
      - context: log
        statements:
          - set(attributes["extracted.level"], ExtractGrokPatterns(body, "%{LOGLEVEL:level}"))
          - set(attributes["extracted.message"], ExtractGrokPatterns(body, "message=(?P<message>.*)$"))
```

## Example: Production Configuration

```bash
# Secure TLS connection
stern . --all-namespaces \
  --otel-enabled \
  --otel-endpoint=collector.prod.example.com:4317 \
  --otel-insecure=false \
  --otel-batch-size=2048 \
  --otel-export-timeout=60s

# With authentication headers (via environment)
export OTEL_EXPORTER_OTLP_HEADERS="authorization=Bearer ${API_TOKEN}"
stern . -A --otel-enabled
```

## Querying Logs in Grafana

### LogQL Examples

```logql
# All logs from a namespace
{namespace="default"}

# Logs from specific pod
{namespace="default", pod="my-app-7d8f9c-xyz"}

# Filter by log content
{namespace="default"} |= "error"

# Regex filter
{namespace="default"} |~ "status=[45]\\d{2}"

# Rate of errors
rate({namespace="default"} |= "ERROR" [5m])

# Top 10 pods by log volume
topk(10, sum by (pod) (rate({namespace="default"}[5m])))
```

## Troubleshooting

### Enable Debug Logging

```bash
# Verbose stern output
stern . --otel-enabled --verbosity=6

# Collector debug logs
# In otel-collector-config.yaml:
exporters:
  logging:
    loglevel: debug
```

### Test Collector Connectivity

```bash
# Test gRPC endpoint
grpcurl -plaintext localhost:4317 list

# Test HTTP endpoint
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{"resourceLogs":[]}'
```

### Verify Log Export

```bash
# Check if logs are being received
docker logs otel-collector | grep "LogsExporter"

# Monitor file export
watch -n 1 'wc -l /tmp/stern-logs.json'
```

## Performance Tuning

```bash
# High-volume cluster (thousands of pods)
stern . -A \
  --otel-enabled \
  --otel-batch-size=4096 \
  --otel-export-timeout=120s \
  --max-log-requests=100

# Low-latency requirements
stern . -A \
  --otel-enabled \
  --otel-batch-size=128 \
  --otel-export-timeout=5s
```

## References

- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [Grafana Loki](https://grafana.com/docs/loki/latest/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
