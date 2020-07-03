package checks

import (
	"net/http"
	"time"

	"github.com/integr8ly/workload-web-app/pkg/counters"
	log "github.com/sirupsen/logrus"
)

const amqConsoleService = "amqconsole_service"

type AMQConsoleChecks struct {
	ConsoleURL string
	Interval   time.Duration
}

func (c *AMQConsoleChecks) run() {
	//TODO
	//Get the config and use the bearerToken to pass through openshift auth-proxy

	//Access the AMQ console
	_, err := http.Get(c.ConsoleURL)
	counters.ServiceTotalRequestsCounter.WithLabelValues(amqConsoleService, c.ConsoleURL).Inc()
	if err != nil {
		log.Errorf("An error has occured, %v", err)
		counters.UpdateErrorMetricsForService(amqConsoleService, c.ConsoleURL, err.Error(), c.Interval.Seconds())
	} else {
		counters.UpdateSuccessMetricsForService(amqConsoleService, c.ConsoleURL)
	}
}

func (c *AMQConsoleChecks) RunForever() {
	counters.InitCounters(amqConsoleService, c.ConsoleURL)
	for {
		c.run()
		time.Sleep(c.Interval)
	}
}
