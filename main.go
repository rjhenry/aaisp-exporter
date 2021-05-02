package main

import (
	"flag"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	refreshSecs       = 60
	aaControlUsername = os.Getenv("AAISP_CONTROL_USERNAME")
	aaControlPassword = os.Getenv("AAISP_CONTROL_PASSWORD")
	httpClient        = resty.New()
	gaugeLabels       = []string{
		"LineID",
		"Login",
		"Postcode",
	}
	upstreamGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "upstream_sync_rate",
		Help: "Raw upstream sync rate (bits/sec)",
	}, gaugeLabels)
	downstreamGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "downstream_sync_rate",
		Help: "Raw downstream sync rate (bits/sec)",
	}, gaugeLabels)
	adjustedDownstreamGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "adjusted_downstream_rate",
		Help: "Adjusted downstream rate after optional rate limiting (bits/sec)",
	}, gaugeLabels)
	monthlyQuotaGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "monthly_allowance",
		Help: "Monthly quota (bytes)",
	}, gaugeLabels)
	quotaRemainingGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "monthly_allowance_remaining",
		Help: "Quota remaining, may exceed monthly_allowance due to rollover of unused quota (bytes)",
	}, gaugeLabels)
	gauges = []prometheus.GaugeVec{
		*upstreamGauge, *downstreamGauge, *adjustedDownstreamGauge, *monthlyQuotaGauge, *quotaRemainingGauge,
	}
	registry = prometheus.NewRegistry()
)

type AaResponse struct {
	Info []map[string]string `json:"info"`
}

func GetUpdatedValues() {
	resp, err := httpClient.
		R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(map[string]string{
			"control_login":    aaControlUsername,
			"control_password": aaControlPassword,
		}).
		SetResult(AaResponse{}).
		Post("https://chaos2.aa.net.uk/broadband/info/json")

	if err != nil {
		if resp != nil {
			logrus.Error(resp.String())
		} else {
			logrus.Error("Unknown failure fetching updated data")
		}
	} else {
		data := resp.Result()
		logrus.WithField("data", data).Info("Update successful")
	}
}

func main() {
	flag.Parse()
	logrus.SetFormatter(&logrus.TextFormatter{})
	// Register the summary and the histogram with Prometheus's default registry.
	for _, gauge := range gauges {
		registry.MustRegister(gauge)
	}
	// Add Go module build info.
	registry.MustRegister(prometheus.NewBuildInfoCollector())

	// Update CHAOS data every 60 seconds.
	go func() {
		for {
			GetUpdatedValues()
			// generate fake data
			for _, gauge := range gauges {
				gauge.With(prometheus.Labels{
					"LineID":   "1234",
					"Login":    "dw121@a.255",
					"Postcode": "N7 7EL",
				}).Set(rand.NormFloat64())
			}

			time.Sleep(
				time.Duration(refreshSecs) * time.Second)
		}
	}()

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	logrus.Fatal(http.ListenAndServe(":2112", nil))
}
