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
	evnVarPort             = "PORT"
	envVarAMQAddress       = "AMQ_ADDRESS"
	envVarAMQQueue         = "AMQ_QUEUE"
	envVarEnvironment      = "ENVIRONMENT"
	envVarRequestInterval  = "REQUEST_INTERVAL"
	envVarURL              = "RHSSO_SERVER_URL"
	envVarUser             = "RHSSO_USER"
	envVarPassword         = "RHSSO_PWD"
	envVarThreeScaleURL    = "THREE_SCALE_URL"
	envVarAMQCRUDNamespace = "AMQ_CRUD_NAMESPACE"

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
		if val, err := strconv.Atoi(os.Getenv(envVarRequestInterval)); err != nil {
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

func startAMQChecks() {
	addr := os.Getenv(envVarAMQAddress)
	q := os.Getenv(envVarAMQQueue)
	if addr != "" && q != "" {
		log.WithFields(log.Fields{
			"address":  addr,
			"queue":    q,
			"interval": counters.RequestInterval,
		}).Info("Start AMQ checks")
		c := &checks.AMQChecks{
			Address:     addr,
			QueueName:   q,
			SendTimeout: 2 * time.Second,
			Interval:    counters.RequestInterval,
		}
		c.RunForever()
	} else {
		log.Warnf("AMQ Checks are not started as env vars %s, %s are not set", envVarAMQAddress, envVarAMQQueue)
	}
}

func startAMQCRUDChecks() {
	namespace := os.Getenv(envVarAMQCRUDNamespace)
	if namespace != "" {
		log.Info("Start AMQ CRUD checks")
		c, err := checks.NewAMQCRUDChecks(namespace)
		if err != nil {
			log.Warnf("Failed to start AMQ CRUD Checks with error: %s", err)
			return
		}

		c.RunForever()
	} else {
		log.Warnf("AMQ CRUD Checks are not started as env var %s is not set", envVarAMQCRUDNamespace)
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
	go startAMQChecks()
	go startSSOChecks()
	go startThreeScaleChecks()
	go startAMQCRUDChecks()
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
