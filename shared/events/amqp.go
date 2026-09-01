// Package events provides a thin RabbitMQ publisher/consumer over the topic
// exchange "askshop.events", keyed by the routing keys declared in
// shared/contracts/amqp.go (e.g. "order.event.placed").
//
// It degrades gracefully: if RABBITMQ_URL is unset or the broker is
// unreachable, Publish and Consume become no-ops (logged once) instead of
// failing service startup. Local dev without RabbitMQ still works; a real
// broker lights the wiring up.
package events

import (
	"askshop/shared/env"
	"context"
	"encoding/json"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "askshop.events"

// Publisher publishes events to the askshop.events topic exchange.
type Publisher struct {
	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewPublisher dials RABBITMQ_URL and declares the shared exchange. If the URL
// is unset or the broker is unreachable, it returns a Publisher that logs and
// no-ops on Publish rather than failing.
func NewPublisher() *Publisher {
	url := env.GetString("RABBITMQ_URL", "")
	if url == "" {
		log.Println("events: RABBITMQ_URL not set, publishing disabled (no-op)")
		return &Publisher{}
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("events: failed to connect to RabbitMQ, publishing disabled: %v", err)
		return &Publisher{}
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("events: failed to open channel, publishing disabled: %v", err)
		conn.Close()
		return &Publisher{}
	}

	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		log.Printf("events: failed to declare exchange, publishing disabled: %v", err)
		ch.Close()
		conn.Close()
		return &Publisher{}
	}

	log.Println("events: connected to RabbitMQ")
	return &Publisher{conn: conn, ch: ch}
}

// Publish sends payload as JSON to the given routing key (see shared/contracts/amqp.go).
// It's a no-op (returns nil) if the publisher has no live connection.
func (p *Publisher) Publish(ctx context.Context, routingKey string, payload interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch == nil {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(ctx, exchangeName, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Close releases the underlying channel/connection, if any.
func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

// Consume starts consuming messages matching routingKey on a dedicated queue
// named by queueName, invoking handler for each delivery. It blocks until ctx
// is cancelled. If RABBITMQ_URL is unset or unreachable, it logs and returns
// immediately (no-op).
func Consume(ctx context.Context, queueName, routingKey string, handler func(body []byte)) {
	url := env.GetString("RABBITMQ_URL", "")
	if url == "" {
		log.Println("events: RABBITMQ_URL not set, consuming disabled (no-op)")
		return
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("events: failed to connect to RabbitMQ, consuming disabled: %v", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("events: failed to open channel, consuming disabled: %v", err)
		return
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		log.Printf("events: failed to declare exchange, consuming disabled: %v", err)
		return
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Printf("events: failed to declare queue %s, consuming disabled: %v", queueName, err)
		return
	}

	if err := ch.QueueBind(q.Name, routingKey, exchangeName, false, nil); err != nil {
		log.Printf("events: failed to bind queue %s to %s, consuming disabled: %v", queueName, routingKey, err)
		return
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("events: failed to start consuming from %s, consuming disabled: %v", queueName, err)
		return
	}

	log.Printf("events: consuming queue=%s routingKey=%s", queueName, routingKey)
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			handler(d.Body)
		}
	}
}
