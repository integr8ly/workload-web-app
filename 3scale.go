package main

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	apiCallsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "three_scale_api_requests_success",
	})
)

func init() {
	prometheus.MustRegister(apiCallsGauge)
}

type ThreeScaleChecks struct {
	url      string
	interval time.Duration
}

func (t *ThreeScaleChecks) runForever() {
	// Create client
	tc := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: tc}
	client := &http.Client{Transport: tr}

	// Start make requests
	t.makeRequests(client)
}

func (t *ThreeScaleChecks) makeRequests(client *http.Client) {
	for {
		// Make Request
		r, err := http.Get(t.url)
		if err != nil {
			apiCallsGauge.Set(0)
			log.Warnf("3scale: request failed with error: %v", err)
		} else if r.StatusCode != http.StatusOK {
			apiCallsGauge.Set(0)
			log.Warnf("3scale: request failed with status code: %d", r.StatusCode)
		} else {
			apiCallsGauge.Set(1)
		}

		// Wait intervall
		time.Sleep(t.interval)
	}
}
