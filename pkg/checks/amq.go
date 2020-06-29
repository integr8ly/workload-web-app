package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/integr8ly/workload-web-app/pkg/counters"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const amqSenderService = "amq_sender"
const amqReceiverService = "amq_receiver"

type AMQChecks struct {
	Address     string
	QueueName   string
	SendTimeout time.Duration
	Interval    time.Duration
}

func (a *AMQChecks) url() string {
	return fmt.Sprintf("%s%s", a.Address, a.QueueName)
}

func (a *AMQChecks) run(ctx context.Context) error {
	// Create client
	t := &tls.Config{InsecureSkipVerify: true}
	opts := amqp.ConnTLSConfig(t)
	client, err := amqp.Dial(a.Address, opts, amqp.ConnSASLAnonymous())
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", a.Address, err)
	}
	defer client.Close()

	// Create session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %v", err)
	}

	// Create a sender
	sender, err := session.NewSender(
		amqp.LinkTargetAddress(a.QueueName),
	)
	if err != nil {
		return fmt.Errorf("failed to create a new sender: %v", err)
	}

	// Create a receiver
	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress(a.QueueName),
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
		ctx, cancel := context.WithTimeout(ctx, a.SendTimeout)
		// Send message
		err := sender.Send(ctx, amqp.NewMessage([]byte("Hello!")))
		cancel()
		counters.ServiceTotalRequestsCounter.WithLabelValues(amqSenderService, a.url()).Inc()
		if err != nil {
			counters.UpdateErrorMetricsForService(amqSenderService, a.url(), err.Error(), a.Interval.Seconds())
			return err
		} else {
			counters.UpdateSuccessMetricsForService(amqSenderService, a.url())
		}
		time.Sleep(a.Interval)
	}
}

func (a *AMQChecks) receiveMessages(ctx context.Context, receiver *amqp.Receiver) error {
	for {
		// Receive next message
		msg, err := receiver.Receive(ctx)
		counters.ServiceTotalRequestsCounter.WithLabelValues(amqReceiverService, a.url()).Inc()
		if err != nil {
			counters.UpdateErrorMetricsForService(amqReceiverService, a.url(), err.Error(), a.Interval.Seconds())
			return err
		}
		// Accept message
		if msg != nil {
			counters.UpdateSuccessMetricsForService(amqReceiverService, a.url())
			msg.Accept()
			log.WithField("message", string(msg.GetData())).Debug("Message received")
		}
		time.Sleep(a.Interval)
	}
}

func (a *AMQChecks) RunForever() {
	counters.InitCounters(amqSenderService, a.url())
	counters.InitCounters(amqReceiverService, a.url())
	for {
		err := a.run(context.Background())
		if err != nil {
			log.WithField("error", err).Warnf("error occured")
			time.Sleep(a.Interval)
		}
	}
}
