package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type AaResponse struct {
	Info []AaLineData `json:"info"`
}

type AaLineData struct {
	LineID         string `json:"id"`
	QuotaMonthly   string `json:"quota_monthly"`
	QuotaRemaining string `json:"quota_remaining"`
	RxRate         string `json:"rx_rate"`
	TxRate         string `json:"tx_rate"`
	TxRateAdjusted string `json:"tx_rate_adjusted"`
}

type AaGauges struct {
	QuotaMonthly   prometheus.GaugeVec
	QuotaRemaining prometheus.GaugeVec
	RxRate         prometheus.GaugeVec
	TxRate         prometheus.GaugeVec
	TxRateAdjusted prometheus.GaugeVec
}

//nolint:funlen
func main() {
	flag.Parse()
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat:   "",
		DisableTimestamp:  false,
		DisableHTMLEscape: false,
		DataKey:           "",
		FieldMap:          nil,
		CallerPrettyfier:  nil,
		PrettyPrint:       false,
	})

	var (
		listenPort    = "9902"
		refreshSecs   = 60
		gaugeLabels   = []string{"LineID"}
		upstreamGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "", Subsystem: "", ConstLabels: nil,
			Name: "upstream_sync_rate",
			Help: "Raw upstream sync rate (bits/sec)",
		}, gaugeLabels)
		downstreamGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "", Subsystem: "", ConstLabels: nil,
			Name: "downstream_sync_rate",
			Help: "Raw downstream sync rate (bits/sec)",
		}, gaugeLabels)
		adjustedDownstreamGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "", Subsystem: "", ConstLabels: nil,
			Name: "downstream_rate_adjusted",
			Help: "Adjusted downstream rate after optional rate limiting (bits/sec)",
		}, gaugeLabels)
		monthlyQuotaGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "", Subsystem: "", ConstLabels: nil,
			Name: "monthly_allowance",
			Help: "Monthly quota (bytes)",
		}, gaugeLabels)
		quotaRemainingGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "", Subsystem: "", ConstLabels: nil,
			Name: "monthly_allowance_remaining",
			Help: "Quota remaining, may exceed monthly_allowance due to rollover of unused quota (bytes)",
		}, gaugeLabels)
		gauges = AaGauges{
			QuotaMonthly:   *monthlyQuotaGauge,
			QuotaRemaining: *quotaRemainingGauge,
			RxRate:         *upstreamGauge,
			TxRate:         *downstreamGauge,
			TxRateAdjusted: *adjustedDownstreamGauge,
		}
		registry = prometheus.NewRegistry()
	)

	// Register the gauges with Prometheus's default registry.
	for _, gauge := range []prometheus.GaugeVec{
		gauges.QuotaMonthly,
		gauges.QuotaRemaining,
		gauges.RxRate,
		gauges.TxRate,
		gauges.TxRateAdjusted,
	} {
		registry.MustRegister(gauge)
	}
	// Add Go module build info.
	registry.MustRegister(prometheus.NewBuildInfoCollector())

	// Update CHAOS data every 60 seconds.
	go ScheduleUpdates(gauges, refreshSecs)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			ErrorLog: nil, ErrorHandling: 0, Registry: nil, DisableCompression: false, MaxRequestsInFlight: 0, Timeout: 0,
			EnableOpenMetrics: true,
		},
	))

	customPort, isPresent := os.LookupEnv("AAISP_EXPORTER_PORT")
	if isPresent {
		listenPort = customPort
	}

	logrus.Infof("starting aaisp-exporter on port %s", listenPort)

	logrus.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil))
}

func ScheduleUpdates(gauges AaGauges, refreshSecs int) {
	for {
		vals, err := GetUpdatedValues()
		if err != nil {
			logrus.Error("scheduled update failed")
		} else {
			if len(vals.Info) < 1 {
				logrus.Error("no data returned from CHAOS API")
			} else {
				for _, lineVal := range vals.Info {
					UpdateGauge(lineVal.QuotaMonthly, lineVal.LineID, &gauges.QuotaMonthly)
					UpdateGauge(lineVal.QuotaRemaining, lineVal.LineID, &gauges.QuotaRemaining)
					UpdateGauge(lineVal.RxRate, lineVal.LineID, &gauges.RxRate)
					UpdateGauge(lineVal.TxRate, lineVal.LineID, &gauges.TxRate)
					UpdateGauge(lineVal.TxRateAdjusted, lineVal.LineID, &gauges.TxRateAdjusted)
				}
			}
		}

		time.Sleep(
			time.Duration(refreshSecs) * time.Second)
	}
}

func GetUpdatedValues() (AaResponse, error) {
	var (
		aaControlUsername = os.Getenv("AAISP_CONTROL_USERNAME")
		aaControlPassword = os.Getenv("AAISP_CONTROL_PASSWORD")
		httpClient        = resty.New()
	)

	resp, err := httpClient.
		R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(map[string]string{
			"control_login":    aaControlUsername,
			"control_password": aaControlPassword,
		}).
		SetResult(AaResponse{Info: nil}).
		Post("https://chaos2.aa.net.uk/broadband/info/json")
	if err != nil {
		if resp != nil {
			return AaResponse{}, fmt.Errorf("invalid response from API: %q", err.Error())
		}
		// else
		return AaResponse{}, fmt.Errorf("unknown failure fetching update: %q", err.Error())
	}
	// else
	return *resp.Result().(*AaResponse), nil
}

func UpdateGauge(valStr string, lineID string, gauge *prometheus.GaugeVec) {
	valFloat, err := strconv.ParseFloat(valStr, 64)
	if err == nil {
		gauge.With(prometheus.Labels{
			"LineID": lineID,
		}).Set(valFloat)
	}
}
