package checks

import (
	"fmt"
	"net/http"
	"time"

	"github.com/integr8ly/workload-web-app/pkg/counters"
	"github.com/integr8ly/workload-web-app/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const amqConsoleService = "amqconsole_service"

type AMQConsoleChecks struct {
	ConsoleURL string
	Username   string
	Password   string
	Interval   time.Duration
}

func (c *AMQConsoleChecks) run() error {
	client, err := utils.NewOAuthHTTPClient()
	if err != nil {
		return err
	}

	err = utils.AuthenticateClientThroughProxyOAuth(client, c.ConsoleURL, c.Username, c.Password)
	if err != nil {
		return err
	}
	log.Debug("amq_console: successfully logged-in to AMQ Console")

	response, err := client.Get(c.ConsoleURL)
	if err != nil {
		return fmt.Errorf("Request to %s failed with error: %s", c.ConsoleURL, err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Request to %s retured status code %d", c.ConsoleURL, response.StatusCode)
	}
	log.Debug("amq_console: successfully requested AMQ Console")
	return nil
}

func (c *AMQConsoleChecks) RunForever() {
	counters.InitCounters(amqConsoleService, c.ConsoleURL)
	for {
		err := c.run()
		counters.ServiceTotalRequestsCounter.WithLabelValues(amqConsoleService, c.ConsoleURL).Inc()
		if err != nil {
			log.Errorf("Failed to login to AMQ Console with error, %v", err)
			counters.UpdateErrorMetricsForService(amqConsoleService, c.ConsoleURL, err.Error(), c.Interval.Seconds())
		} else {
			counters.UpdateSuccessMetricsForService(amqConsoleService, c.ConsoleURL)
		}

		time.Sleep(c.Interval)
	}
}
