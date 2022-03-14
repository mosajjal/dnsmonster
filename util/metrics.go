package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	statsd "github.com/syntaqx/go-metrics-datadog"
)

// the capture and output metrics and stats are handled here.

type MetricConfig struct {
	MetricEndpointType       string        `long:"metricEndpointType"       env:"DNSMONSTER_METRICENDPOINTTYPE"       default:"stderr" description:"Metric Endpoint Service"    choice:"statsd" choice:"prometheus" choice:"stderr"`
	MetricStatsdAgent        string        `long:"metricStatsdAgent"        env:"DNSMONSTER_METRICSTATSDAGENT"        default:""       description:"Statsd endpoint. Example: 127.0.0.1:8125 "`
	MetricPrometheusEndpoint string        `long:"metricPrometheusEndpoint" env:"DNSMONSTER_METRICPROMETHEUSENDPOINT" default:""       description:"Prometheus Registry endpoint. Example: http://0.0.0.0:2112/metric"`
	MetricFlushInterval      time.Duration `long:"metricFlushInterval"      env:"DNSMONSTER_METRICFLUSHINTERVAL"      default:"10s"    description:"Interval between sending results to Metric Endpoint"`
	// MetricProxy             string        `long:"metricProxy"              env:"DNSMONSTER_METRICPROXY"             default:""       description:"URI formatted proxy server to use for metric endpoint. Example: http://username:password@hostname:port"`
}

func (metricConfig MetricConfig) SetupMetrics() error {
	switch metricConfig.MetricEndpointType {
	case "statsd":
		if metricConfig.MetricStatsdAgent == "" {
			return fmt.Errorf("statsd Agent is required")
		}
		statsdOptions := []statsd.ReporterOption{
			statsd.UseFlushInterval(metricConfig.MetricFlushInterval),
			statsd.UsePercentiles([]float64{0.25, 0.99}),
		}
		reporter, err := statsd.NewReporter(metrics.DefaultRegistry, metricConfig.MetricStatsdAgent, statsdOptions...)
		if err != nil {
			return err
		}
		go reporter.Flush()

	case "prometheus":
		// log.Infof("Prometheus Metrics enabled")
		if metricConfig.MetricPrometheusEndpoint == "" {
			return fmt.Errorf("promethus Registry is required")
		}
		prometheusClient := prometheusmetrics.NewPrometheusProvider(metrics.DefaultRegistry, "dnsmonster", GeneralFlags.ServerName, prometheus.DefaultRegisterer, 1*time.Second)
		go prometheusClient.UpdatePrometheusMetrics()

		u, err := url.Parse(metricConfig.MetricPrometheusEndpoint)
		if err != nil || u.Path == "" {
			return fmt.Errorf("invalid URL for Prometheus")
		}
		go func() {
			http.Handle(u.Path, promhttp.Handler())
			http.ListenAndServe(u.Host, nil)
		}()

	case "stderr":
		// go metrics.Log(metrics.DefaultRegistry, metricConfig.MetricFlushInterval, log.StandardLogger())
		go func() {
			for range time.Tick(metricConfig.MetricFlushInterval) {
				metricsJson, _ := json.Marshal(metrics.DefaultRegistry.GetAll())
				log.Infof("metrics: %s", metricsJson)
			}
		}()

	default:
		return fmt.Errorf("endpoint Type %s is not supported", metricConfig.MetricEndpointType)
	}

	return nil
}
