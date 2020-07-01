package checks

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	"github.com/integr8ly/workload-web-app/pkg/counters"
	log "github.com/sirupsen/logrus"
)

const threeScaleService = "3scale_service"

type ThreeScaleChecks struct {
	URL      string
	Interval time.Duration
}

func (t *ThreeScaleChecks) RunForever() {
	// Create client
	tc := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: tc}
	client := &http.Client{Transport: tr}

	counters.InitCounters(threeScaleService, t.URL)

	// Start make requests
	t.makeRequests(client)
}

func (t *ThreeScaleChecks) makeRequests(client *http.Client) {
	for {
		// Make Request
		r, err := http.Get(t.URL)
		counters.ServiceTotalRequestsCounter.WithLabelValues(threeScaleService, t.URL).Inc()
		if err != nil {
			counters.UpdateErrorMetricsForService(threeScaleService, t.URL, err.Error(), t.Interval.Seconds())
			log.Warnf("3scale: request failed with error: %v", err)
		} else if r.StatusCode != http.StatusOK {
			counters.UpdateErrorMetricsForService(threeScaleService, t.URL, strconv.Itoa(r.StatusCode), t.Interval.Seconds())
			log.Warnf("3scale: request failed with status code: %d", r.StatusCode)
		} else {
			counters.UpdateSuccessMetricsForService(threeScaleService, t.URL)
		}

		// Wait intervall
		time.Sleep(t.Interval)
	}
}
