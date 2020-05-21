package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Azure/go-amqp"
	"github.com/prometheus/client_golang/prometheus"
	"log"
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

func init()  {
	prometheus.MustRegister(messagesSentGauge)
	prometheus.MustRegister(messagesReceivedGauge)
}

type AMQChecks struct {
	address string
	queueName string
	sendTimeout time.Duration
	interval time.Duration
}

func (a *AMQChecks) runForever() error {
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

	// Start receiving messages
	go func(receiver *amqp.Receiver) {
		a.receiveMessages(receiver)
	}(receiver)

	//start send messages
	a.sendMessages(sender)
	return nil
}

func (a *AMQChecks) sendMessages(sender *amqp.Sender) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), a.sendTimeout)
		// Send message
		err := sender.Send(ctx, amqp.NewMessage([]byte("Hello!")))
		cancel()
		if err != nil {
			messagesSentGauge.Set(0)
			log.Printf("Error when send message: %v", err)
		} else {
			messagesSentGauge.Set(1)
		}
		time.Sleep(a.interval)
	}
}

func (a *AMQChecks) receiveMessages(receiver *amqp.Receiver) error {
	for {
		ctx := context.Background()
		// Receive next message
		msg, err := receiver.Receive(ctx)
		if err != nil {
			messagesReceivedGauge.Set(0)
			log.Printf("Error when read message from AMQP: %v", err)
		}
		// Accept message
		if msg != nil {
			messagesReceivedGauge.Set(1)
			msg.Accept()
			log.Printf("Message received: %s", msg.GetData())
		}
		time.Sleep(a.interval)
	}
}

//func main() {
//	c := &AMQChecks{
//		address:     "amqps://test:test@localhost",
//		queueName:   "/queue-requests",
//		sendTimeout: 1*time.Second,
//		interval:    1*time.Second,
//	}
//	c.runForever()
//}