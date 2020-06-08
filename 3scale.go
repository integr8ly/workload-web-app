package main

import (
	"crypto/tls"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

const threeScaleService = "3scale_service"

type ThreeScaleChecks struct {
	url      string
	interval time.Duration
}

func (t *ThreeScaleChecks) runForever() {
	// Create client
	tc := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: tc}
	client := &http.Client{Transport: tr}

	initCounters(threeScaleService, t.url)

	// Start make requests
	t.makeRequests(client)
}

func (t *ThreeScaleChecks) makeRequests(client *http.Client) {
	for {
		// Make Request
		r, err := http.Get(t.url)
		serviceTotalRequestsCounter.WithLabelValues(threeScaleService, t.url).Inc()
		if err != nil {
			updateErrorMetricsForService(threeScaleService, t.url, err.Error(), t.interval.Seconds())
			log.Warnf("3scale: request failed with error: %v", err)
		} else if r.StatusCode != http.StatusOK {
			updateErrorMetricsForService(threeScaleService, t.url, strconv.Itoa(r.StatusCode), t.interval.Seconds())
			log.Warnf("3scale: request failed with status code: %d", r.StatusCode)
		} else {
			updateSuccessMetricsForService(threeScaleService, t.url)
		}

		// Wait intervall
		time.Sleep(t.interval)
	}
}


