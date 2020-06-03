package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	evnVarPort        = "PORT"
	envVarAMQAddress  = "AMQ_ADDRESS"
	envVarAMQQueue    = "AMQ_QUEUE"
	envVarEnvironment = "ENVIRONMENT"
	productionEnv     = "production"
	envVarURL         = "RHSSO_SERVER_URL"
	envVarUser        = "RHSSO_USER"
	envVarPassword    = "RHSSO_PWD"
)

func init() {
	log.SetOutput(os.Stdout)
	if strings.ToLower(os.Getenv(envVarEnvironment)) == productionEnv {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func startAMQChecks() {
	addr := os.Getenv(envVarAMQAddress)
	q := os.Getenv(envVarAMQQueue)
	if addr != "" && q != "" {
		log.WithFields(log.Fields{
			"address": addr,
			"queue":   q,
		}).Info("Start AMQ checks")
		c := &AMQChecks{
			address:     addr,
			queueName:   q,
			sendTimeout: 2 * time.Second,
			interval:    1 * time.Second,
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
		}).Info("Start SSO Checks")
		c := &SSOChecks{
			serverURL: url,
			user:      user,
			password:  pwd,
			realmName: realm,
			interval:  7 * time.Second,
		}
		c.runForever()
	} else {
		log.Warnf("SSO checks are not started as env vars are not set correctly!")
	}
}

func main() {
	go startAMQChecks()
	go startSSOChecks()
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
