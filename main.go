package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/integr8ly/workload-web-app/pkg/checks"
	"github.com/integr8ly/workload-web-app/pkg/counters"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	evnVarPort            = "PORT"
	envVarEnvironment     = "ENVIRONMENT"
	envVarRequestInterval = "REQUEST_INTERVAL"
	envVarURL             = "RHSSO_SERVER_URL"
	envVarUser            = "RHSSO_USER"
	envVarPassword        = "RHSSO_PWD"
	envVarThreeScaleURL   = "THREE_SCALE_URL"

	productionEnv = "production"
)

func init() {
	log.SetOutput(os.Stdout)
	if strings.ToLower(os.Getenv(envVarEnvironment)) == productionEnv {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}
	if os.Getenv(envVarRequestInterval) != "" {
		if val, err := strconv.Atoi(os.Getenv(envVarRequestInterval)); err == nil {
			counters.RequestInterval = time.Duration(val) * time.Second
		}
	}
	prometheus.MustRegister(counters.ServiceUpGauge)
	prometheus.MustRegister(counters.ServiceTotalRequestsCounter)
	prometheus.MustRegister(counters.ServiceTotalErrorsCounter)
	prometheus.MustRegister(counters.ServiceTotalDowntimeCounter)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
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
			"interval":  counters.RequestInterval,
		}).Info("Start SSO Checks")
		c := &checks.SSOChecks{
			ServerURL: url,
			User:      user,
			Password:  pwd,
			RealmName: realm,
			Interval:  counters.RequestInterval,
		}
		c.RunForever()
	} else {
		log.Warnf("SSO checks are not started as env vars are not set correctly!")
	}
}

func startThreeScaleChecks() {
	url := os.Getenv(envVarThreeScaleURL)
	if url != "" {
		log.WithFields(log.Fields{
			"url":      url,
			"interval": counters.RequestInterval,
		}).Info("Start 3scale checks")
		c := &checks.ThreeScaleChecks{
			URL:      url,
			Interval: counters.RequestInterval,
		}
		c.RunForever()
	} else {
		log.Warnf("3sclae: 3scale checks are not started because the environment variables %s is not set", envVarThreeScaleURL)
	}
}

func main() {
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
