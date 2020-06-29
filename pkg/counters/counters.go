package counters

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricsPrefix           = "workload_app"
	defaultRequestsInterval = 10 * time.Second
)

var (
	ServiceUpGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_service_up", metricsPrefix),
	}, []string{"name", "service"})
	ServiceTotalRequestsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_service_requests_total", metricsPrefix),
	}, []string{"name", "service"})
	ServiceTotalErrorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_service_requests_errors_total", metricsPrefix),
	}, []string{"name", "service", "cause"})
	ServiceTotalDowntimeCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_service_downtime_seconds", metricsPrefix),
	}, []string{"name", "service"})
	RequestInterval = defaultRequestsInterval
)

func InitCounters(name string, service string) {
	// counters with dynamic labels need to explicitly set initial value
	ServiceTotalRequestsCounter.WithLabelValues(name, service).Add(0)
	ServiceTotalErrorsCounter.WithLabelValues(name, service, "").Add(0)
	ServiceTotalDowntimeCounter.WithLabelValues(name, service).Add(0)
}

func UpdateErrorMetricsForService(name string, service string, cause string, downtime float64) {
	ServiceUpGauge.WithLabelValues(name, service).Set(0)
	ServiceTotalErrorsCounter.WithLabelValues(name, service, cause).Inc()
	ServiceTotalDowntimeCounter.WithLabelValues(name, service).Add(downtime)
}

func UpdateSuccessMetricsForService(name string, service string) {
	ServiceUpGauge.WithLabelValues(name, service).Set(1)
}
