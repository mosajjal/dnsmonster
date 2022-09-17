package util

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	statsd "github.com/syntaqx/go-metrics-datadog"
	"golang.org/x/sync/errgroup"
)

// the capture and output metrics and stats are handled here.
type metricConfig struct {
	MetricEndpointType       string        `long:"metricendpointtype"       ini-name:"metricendpointtype"       env:"DNSMONSTER_METRICENDPOINTTYPE"       default:"stderr" description:"Metric Endpoint Service"                                           choice:"statsd" choice:"prometheus" choice:"stderr"`
	MetricStatsdAgent        string        `long:"metricstatsdagent"        ini-name:"metricstatsdagent"        env:"DNSMONSTER_METRICSTATSDAGENT"        default:""       description:"Statsd endpoint. Example: 127.0.0.1:8125 "`
	MetricPrometheusEndpoint string        `long:"metricprometheusendpoint" ini-name:"metricprometheusendpoint" env:"DNSMONSTER_METRICPROMETHEUSENDPOINT" default:""       description:"Prometheus Registry endpoint. Example: http://0.0.0.0:2112/metric"`
	MetricStderrFormat       string        `long:"metricstderrformat"       ini-name:"metricstderrformat"       env:"DNSMONSTER_METRICSTDERRFORMAT"       default:"json"   description:"Format for stderr output."                                         choice:"json"   choice:"kv"`
	MetricFlushInterval      time.Duration `long:"metricflushinterval"      ini-name:"metricflushinterval"      env:"DNSMONSTER_METRICFLUSHINTERVAL"      default:"10s"    description:"Interval between sending results to Metric Endpoint"`
	// MetricProxy             string        `long:"metricproxy" ini-name:"metricproxy"              env:"DNSMONSTER_METRICPROXY"             default:""       description:"URI formatted proxy server to use for metric endpoint. Example: http://username:password@hostname:port"`
}

func (c metricConfig) SetupMetrics(ctx context.Context) error {
	//todo: none of the below goroutines have a consumer for ctx.Done()
	g, gCtx := errgroup.WithContext(ctx)
	switch c.MetricEndpointType {
	case "statsd":
		if c.MetricStatsdAgent == "" {
			return fmt.Errorf("statsd Agent is required")
		}
		statsdOptions := []statsd.ReporterOption{
			statsd.UseFlushInterval(c.MetricFlushInterval),
			statsd.UsePercentiles([]float64{0.25, 0.99}),
		}
		reporter, err := statsd.NewReporter(metrics.DefaultRegistry, c.MetricStatsdAgent, statsdOptions...)
		if err != nil {
			return err
		}
		g.Go(func() error { reporter.Flush(); return nil })

	case "prometheus":
		log.Infof("Prometheus Metrics enabled")
		if c.MetricPrometheusEndpoint == "" {
			return fmt.Errorf("promethus Registry is required")
		}
		prometheusClient := prometheusmetrics.NewPrometheusProvider(metrics.DefaultRegistry, "dnsmonster", GeneralFlags.ServerName, prometheus.DefaultRegisterer, 1*time.Second)
		g.Go(func() error { prometheusClient.UpdatePrometheusMetrics(); return nil })

		u, err := url.Parse(c.MetricPrometheusEndpoint)
		if err != nil || u.Path == "" {
			return fmt.Errorf("invalid URL for Prometheus")
		}
		g.Go(func() error {
			http.Handle(u.Path, promhttp.Handler())
			return http.ListenAndServe(u.Host, nil)
		})

	case "stderr":
		// go metrics.Log(metrics.DefaultRegistry, metricConfig.MetricFlushInterval, log.StandardLogger())
		g.Go(func() error {
			for {
				select {
				case <-time.Tick(c.MetricFlushInterval):
					out := ""
					switch c.MetricStderrFormat {
					case "json":
						if jMetrics, err := json.Marshal(metrics.DefaultRegistry.GetAll()); err != nil {
							log.Warnf("failed to convert metrics to JSON.")
						} else {
							out = string(jMetrics)
						}
					case "kv":
						for k1, v := range metrics.DefaultRegistry.GetAll() {
							out += fmt.Sprintf("%s=%v ", k1, v[reflect.ValueOf(v).MapKeys()[0].String()])
						}
					}
					os.Stderr.WriteString(fmt.Sprintf("%s metrics: %s\n", time.Now().Format(time.RFC3339), out))
				case <-gCtx.Done():
					log.Debug("metric goroutine exiting") //todo:remove
					return nil
				}
			}
		})

	default:
		return fmt.Errorf("endpoint Type %s is not supported", c.MetricEndpointType)
	}

	return nil
}
