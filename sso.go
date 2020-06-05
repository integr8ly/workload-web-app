package main

import (
	"crypto/tls"
	"time"

	"github.com/Nerzal/gocloak/v3"
	log "github.com/sirupsen/logrus"
)

const ssoService = "sso_service"

type SSOChecks struct {
	serverURL string
	user      string
	password  string
	realmName string
	interval  time.Duration
}

func (s *SSOChecks) run() {
	//create client
	client := gocloak.NewClient(s.serverURL)
	restyClient := client.RestyClient()
	restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	_, err := client.LoginAdmin(s.user, s.password, s.realmName)
	serviceTotalRequestsCounter.WithLabelValues(ssoService, s.serverURL).Inc()
	if err != nil {
		updateErrorMetricsForService(ssoService, s.serverURL, err.Error(), s.interval.Seconds())
		log.Warnf("Login failed with error  :%v", err)
	} else {
		updateSuccessMetricsForService(ssoService, s.serverURL)
		log.Info("Login succeeded!!")
	}
}

func (s *SSOChecks) runForever() {
	initCounters(ssoService, s.serverURL)
	for {
		s.run()
		time.Sleep(s.interval)
	}
}
