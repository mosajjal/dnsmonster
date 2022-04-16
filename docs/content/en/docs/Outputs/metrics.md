---
title: "Metrics"
linkTitle: "Dnsmonster Metrics"
weight: 400
---

Each enabled input and output comes with a set of metrics in order to monitor performance and troubleshoot your running instance. `dnsmonster` uses the [go-metrics](https://github.com/rcrowley/go-metrics) library which makes it easy to register metrics on the fly and in a modular way. 

currently, three metric outputs are supported:
- `stderr`
- `statsd`
- `prometheus`

## Configuration parameters

```ini
[metric]
; Metric Endpoint Service. Choices: stderr, statsd, prometheus
MetricEndpointType = stderr

; Statsd endpoint. Example: 127.0.0.1:8125 
MetricStatsdAgent =

; Prometheus Registry endpoint. Example: http://0.0.0.0:2112/metric
MetricPrometheusEndpoint =

; Interval between sending results to Metric Endpoint
MetricFlushInterval = 10s
```
