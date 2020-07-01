package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	evnVarPort            = "PORT"
	envVarAMQAddress      = "AMQ_ADDRESS"
	envVarAMQQueue        = "AMQ_QUEUE"
	envVarAMQConsoleURL   = "AMQ_CONSOLE_URL"
	envVarEnvironment     = "ENVIRONMENT"
	envVarRequestInterval = "REQUEST_INTERVAL"
	envVarURL             = "RHSSO_SERVER_URL"
	envVarUser            = "RHSSO_USER"
	envVarPassword        = "RHSSO_PWD"
	envVarThreeScaleURL   = "THREE_SCALE_URL"

	productionEnv = "production"

	metricsPrefix           = "workload_app"
	defaultRequestsInterval = 10 * time.Second
)

var (
	serviceUpGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_service_up", metricsPrefix),
	}, []string{"name", "service"})
	serviceTotalRequestsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_service_requests_total", metricsPrefix),
	}, []string{"name", "service"})
	serviceTotalErrorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_service_requests_errors_total", metricsPrefix),
	}, []string{"name", "service", "cause"})
	serviceTotalDowntimeCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_service_downtime_seconds", metricsPrefix),
	}, []string{"name", "service"})
	requestInterval = defaultRequestsInterval
)

func init() {
	log.SetOutput(os.Stdout)
	if strings.ToLower(os.Getenv(envVarEnvironment)) == productionEnv {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}
	if os.Getenv(envVarRequestInterval) != "" {
		if val, err := strconv.Atoi(os.Getenv(envVarRequestInterval)); err != nil {
			requestInterval = time.Duration(val) * time.Second
		}
	}
	prometheus.MustRegister(serviceUpGauge)
	prometheus.MustRegister(serviceTotalRequestsCounter)
	prometheus.MustRegister(serviceTotalErrorsCounter)
	prometheus.MustRegister(serviceTotalDowntimeCounter)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func startAMQChecks() {
	addr := os.Getenv(envVarAMQAddress)
	q := os.Getenv(envVarAMQQueue)
	console := os.Getenv(envVarAMQConsoleURL)
	if addr != "" && q != "" {
		log.WithFields(log.Fields{
			"address":    addr,
			"queue":      q,
			"consoleURL": console,
			"interval":   requestInterval,
		}).Info("Start AMQ checks")
		c := &AMQChecks{
			address:     addr,
			queueName:   q,
			consoleURL:  console,
			sendTimeout: 2 * time.Second,
			interval:    requestInterval,
		}
		c.runForever()
	} else {
		log.Warnf("AMQ Checks are not started as env vars %s, %s are not set", envVarAMQAddress, envVarAMQQueue)
	}
}

func startSSOChecks() {
	url := os.Getenv(envVarURL)
	user := os.Getenv(envVarUser)
	pwd := os.Getenv(envVarPassword)
	realm := "master"
	if url != "" && user != "" && pwd != "" && realm != "" {
		log.WithFields(log.Fields{
			"serverURL": url,
			"realmName": realm,
			"interval":  requestInterval,
		}).Info("Start SSO Checks")
		c := &SSOChecks{
			serverURL: url,
			user:      user,
			password:  pwd,
			realmName: realm,
			interval:  requestInterval,
		}
		c.runForever()
	} else {
		log.Warnf("SSO checks are not started as env vars are not set correctly!")
	}
}

func startThreeScaleChecks() {
	url := os.Getenv(envVarThreeScaleURL)
	if url != "" {
		log.WithFields(log.Fields{
			"url":      url,
			"interval": requestInterval,
		}).Info("Start 3scale checks")
		c := &ThreeScaleChecks{
			url:      url,
			interval: requestInterval,
		}
		c.runForever()
	} else {
		log.Warnf("3sclae: 3scale checks are not started because the environment variables %s is not set", envVarThreeScaleURL)
	}
}

func main() {
	go startAMQChecks()
	go startSSOChecks()
	go startThreeScaleChecks()
	startHttpServer()
}

func startHttpServer() {
	p := os.Getenv(evnVarPort)
	if p == "" {
		p = "8080"
	}
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", handler)
	log.WithField("port", p).Infof("Starting HTTP server")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", p), nil))
}

func initCounters(name string, service string) {
	// counters with dynamic labels need to explicitly set initial value
	serviceTotalRequestsCounter.WithLabelValues(name, service).Add(0)
	serviceTotalErrorsCounter.WithLabelValues(name, service, "").Add(0)
	serviceTotalDowntimeCounter.WithLabelValues(name, service).Add(0)
}

func updateErrorMetricsForService(name string, service string, cause string, downtime float64) {
	serviceUpGauge.WithLabelValues(name, service).Set(0)
	serviceTotalErrorsCounter.WithLabelValues(name, service, cause).Inc()
	serviceTotalDowntimeCounter.WithLabelValues(name, service).Add(downtime)
}

func updateSuccessMetricsForService(name string, service string) {
	serviceUpGauge.WithLabelValues(name, service).Set(1)
}
