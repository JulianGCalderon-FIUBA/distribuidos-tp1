package middleware

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Middleware struct {
	ch *amqp.Channel
}

func NewMiddleware(conn *amqp.Connection) *Middleware {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to bind rabbit connection: %v", err)
	}
	return &Middleware{
		ch: ch,
	}
}

func (m *Middleware) Init() {
	err := m.initExchange()
	if err != nil {
		log.Fatalf("failed to initialize exchanges %v", err)
	}
	err = m.initQueue()
	if err != nil {
		log.Fatalf("failed to initialize queues %v", err)
	}
}

func (m *Middleware) initExchange() error {
	err := m.ch.ExchangeDeclare(
		ReviewExchange,
		amqp.ExchangeFanout,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare reviews exchange: %w", err)
	}
	err = m.ch.ExchangeDeclare(
		GamesExchange,
		amqp.ExchangeFanout,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind games exchange: %w", err)
	}
	return nil
}

func (m *Middleware) initQueue() error {
	q, err := m.ch.QueueDeclare(GamesPartitionerQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not declare games-partitioner queue: %w", err)
	}

	err = m.ch.QueueBind(
		q.Name,
		"",
		GamesExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not bind to games-partitioner queue: %w", err)
	}

	q, err = m.ch.QueueDeclare(GamesFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not declare games-partitioner queue: %w", err)
	}

	err = m.ch.QueueBind(
		q.Name,
		"",
		GamesExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not bind to games-partitioner queue: %w", err)
	}

	q, err = m.ch.QueueDeclare(ReviewsFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not declare games-partitioner queue: %w", err)
	}

	err = m.ch.QueueBind(
		q.Name,
		"",
		ReviewExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not bind to games-partitioner queue: %w", err)
	}

	return nil
}
