package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Azure/go-amqp"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"time"
)

var (
	messagesSentGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "amq_messages_sent_success",
	})
	messagesReceivedGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "amq_messages_received_success",
	})
)

func init() {
	prometheus.MustRegister(messagesSentGauge)
	prometheus.MustRegister(messagesReceivedGauge)
}

type AMQChecks struct {
	address     string
	queueName   string
	sendTimeout time.Duration
	interval    time.Duration
}

func (a *AMQChecks) run(ctx context.Context) error {
	// Create client
	t := &tls.Config{InsecureSkipVerify: true}
	opts := amqp.ConnTLSConfig(t)
	client, err := amqp.Dial(a.address, opts)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", a.address, err)
	}
	defer client.Close()

	// Create session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %v", err)
	}

	// Create a sender
	sender, err := session.NewSender(
		amqp.LinkTargetAddress(a.queueName),
	)
	if err != nil {
		return fmt.Errorf("failed to create a new sender: %v", err)
	}

	// Create a receiver
	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress(a.queueName),
		amqp.LinkCredit(10),
	)
	if err != nil {
		return fmt.Errorf("failed to create a new receiver: %v", err)
	}

	// This will cancel all goroutines if any of them returns a non-nil error
	g, ctx := errgroup.WithContext(ctx)
	// Start receiving messages
	g.Go(func() error {
		return a.receiveMessages(ctx, receiver)
	})
	// Start sending messages
	g.Go(func() error {
		return a.sendMessages(ctx, sender)
	})

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (a *AMQChecks) sendMessages(ctx context.Context, sender *amqp.Sender) error {
	for {
		ctx, cancel := context.WithTimeout(ctx, a.sendTimeout)
		// Send message
		err := sender.Send(ctx, amqp.NewMessage([]byte("Hello!")))
		cancel()
		if err != nil {
			messagesSentGauge.Set(0)
			return err
		} else {
			messagesSentGauge.Set(1)
		}
		time.Sleep(a.interval)
	}
}

func (a *AMQChecks) receiveMessages(ctx context.Context, receiver *amqp.Receiver) error {
	for {
		// Receive next message
		msg, err := receiver.Receive(ctx)
		if err != nil {
			messagesReceivedGauge.Set(0)
			return err
		}
		// Accept message
		if msg != nil {
			messagesReceivedGauge.Set(1)
			msg.Accept()
			log.WithField("message", string(msg.GetData())).Debug("Message received")
		}
		time.Sleep(a.interval)
	}
}

func (a *AMQChecks) runForever() {
	for {
		err := a.run(context.Background())
		if err != nil {
			log.WithField("error", err).Warnf("error occured")
			time.Sleep(a.interval)
		}
	}
}
