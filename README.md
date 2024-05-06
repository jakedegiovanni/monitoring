# Monitoring

A monitoring stack for your personal projects. Instrument with Opentelemetry and send traces and metrics to localhost:4317 via gRPC.

Logs to ....

`docker compose up` to spin up the stack

## OpenTelemtry

- Collector agent
  - Exposes otel-collector:9091/metrics for metrics
  - pushes to <TODO> for traces
  - <TODO> logs...
- Push to agent :4317 for grpc instruments

## Prometheus

- Scrapes otel-collector:9091/metrics to gather app metrics
- Exposes them under localhost:9090

## Grafana

### Metrics

### Grafana Loki

### Grafana Tempo