# Monitoring

## Prometheus :simple-prometheus:

[:octicons-tag-24: 1.0.0](https://github.com/DeepSpace2/plugnpin/releases/tag/v1.0.0){ .md-tag target="_blank" }

PlugNPiN optionally exposes runtime and application metrics in Prometheus format  
by using the `METRICS` environment variable ([Configuration → Environment Variables](./configuration.md#optional)).

By default the metrics endpoint is available at `http://<plugnpin-host>:9100/metrics`  
and the port can be set with the `METRICS_SERVER_PORT` environment variable.

### Example Prometheus Scrape Configuration

```yaml
scrape_configs:
  - job_name: "plugnpin"
    static_configs:
      - targets: ["<plugnpin-host>:9100"]
```

