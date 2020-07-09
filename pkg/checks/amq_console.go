package checks

import (
	"net/http"
	"strconv"
	"time"

	"github.com/integr8ly/workload-web-app/pkg/counters"
	"github.com/integr8ly/workload-web-app/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const amqConsoleService = "amqconsole_service"

type AMQConsoleChecks struct {
	ConsoleURL string
	Interval   time.Duration
}

func (c *AMQConsoleChecks) run() {
	//Get the config and use the bearerToken to pass through openshift auth-proxy
	config, err := utils.GetClusterConfig()
	if err != nil {
		log.Info("An error has occured : %v, err")
		return
	}

	//Create new request using http
	req, err := http.NewRequest("GET", c.ConsoleURL, nil)

	//Add authorization header to the req
	req.Header.Add("Authorization", config.BearerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	counters.ServiceTotalRequestsCounter.WithLabelValues(amqConsoleService, c.ConsoleURL).Inc()
	if err != nil {
		counters.UpdateErrorMetricsForService(amqConsoleService, c.ConsoleURL, err.Error(), c.Interval.Seconds())
		log.Warnf("AMQ Console is not reachable with error, %v", err)

	} else if resp.StatusCode != http.StatusOK {
		counters.UpdateErrorMetricsForService(amqConsoleService, c.ConsoleURL, strconv.Itoa(resp.StatusCode), c.Interval.Seconds())
		log.Warnf("AMQ Console is not reachable with status code: %d", resp.StatusCode)
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
