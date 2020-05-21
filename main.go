package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	evnVarPort = "PORT"
	envVarAMQAddress = "AMQ_ADDRESS"
	envVarAMQQueue = "AMQ_QUEUE"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func startAMQChecks() {
	addr := os.Getenv(envVarAMQAddress)
	q := os.Getenv(envVarAMQQueue)
	if addr != "" && q != "" {
		log.Printf("Start AMQ checks. Address: %s Queue: %s", addr, q)
		c := &AMQChecks{
			address:     addr,
			queueName:   q,
			sendTimeout: 2*time.Second,
			interval:    1*time.Second,
		}
		if err := c.runForever(); err != nil {
			log.Printf("Failed to run AMQ checks due to error : %v", err)
		}
	} else {
		log.Printf("AMQ Checks are not started as there are no environment variables set")
	}
}

func main() {
	go startAMQChecks()
	startHttpServer()
}

func startHttpServer() {
	p := os.Getenv(evnVarPort)
	if p == "" {
		p = "8080"
	}
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", handler)
	log.Printf("Starting HTTP server on port: %s", p)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", p), nil))
}
