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

	err = m.ch.ExchangeDeclare(
		GenresExchange,
		amqp.ExchangeDirect,
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

	q, err = m.ch.QueueDeclare(GamesQueue,
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

	q, err = m.ch.QueueDeclare(ReviewsQueue,
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

func (m *Middleware) InitGenresQueues(routingKeys []string) error {
	q, err := m.ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("could not declare genre queue: %w", err)
	}

	for _, genre := range routingKeys {
		log.Printf("Binding queue %s to exchange %s with routing key %s",
			q.Name, GenresExchange, genre)

		err = m.ch.QueueBind(
			q.Name,
			genre,
			GenresExchange,
			false,
			nil)

		if err != nil {
			return fmt.Errorf("could not bind genre queue: %w", err)
		}
	}
	return nil
}
