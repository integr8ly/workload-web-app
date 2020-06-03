package main

import (
	"crypto/tls"
	"time"

	"github.com/Nerzal/gocloak/v3"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	ssoLoginGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sso_login_success",
	})
)

func init() {
	prometheus.MustRegister(ssoLoginGauge)
}

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
	if err != nil {
		ssoLoginGauge.Set(0)
		log.Warnf("Login failed with error  :%v", err)
	} else {
		ssoLoginGauge.Set(1)
		log.Info("Login succeeded!!")
	}
}

func (s *SSOChecks) runForever() {
	for {
		s.run()
		time.Sleep(s.interval)
	}
}
