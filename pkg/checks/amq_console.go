package checks

import (
	"net/http"
	"os"
	"time"

	"github.com/integr8ly/workload-web-app/pkg/counters"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const amqConsoleService = "amqconsole_service"

type AMQConsoleChecks struct {
	ConsoleURL string
	Interval   time.Duration
}

func (c *AMQConsoleChecks) run() {
	//Get the config and use the bearerToken to pass through openshift auth-proxy
	config, err := rest.InClusterConfig()
	if err != nil {
		if err == rest.ErrNotInCluster {
			// fall back to kubeconfig
			kubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
			if kubeconfig == "" {
				// fall back to recommended kubeconfig location
				kubeconfig = clientcmd.RecommendedHomeFile
			}

			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				log.Errorf("Error occured, %v", err)
				return
			}
		}
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
