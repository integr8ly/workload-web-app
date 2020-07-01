package checks

import (
	"crypto/tls"
	"time"

	"github.com/Nerzal/gocloak/v3"
	"github.com/integr8ly/workload-web-app/pkg/counters"
	log "github.com/sirupsen/logrus"
)

const ssoService = "sso_service"

type SSOChecks struct {
	ServerURL string
	User      string
	Password  string
	RealmName string
	Interval  time.Duration
}

func (s *SSOChecks) run() {
	//create client
	client := gocloak.NewClient(s.ServerURL)
	restyClient := client.RestyClient()
	restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	_, err := client.LoginAdmin(s.User, s.Password, s.RealmName)
	counters.ServiceTotalRequestsCounter.WithLabelValues(ssoService, s.ServerURL).Inc()
	if err != nil {
		counters.UpdateErrorMetricsForService(ssoService, s.ServerURL, err.Error(), s.Interval.Seconds())
		log.Warnf("Login failed with error  :%v", err)
	} else {
		counters.UpdateSuccessMetricsForService(ssoService, s.ServerURL)
		log.Info("Login succeeded!!")
	}
}

func (s *SSOChecks) RunForever() {
	counters.InitCounters(ssoService, s.ServerURL)
	for {
		s.run()
		time.Sleep(s.Interval)
	}
}
