package middleware

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Init(conn *amqp.Connection) {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to bind rabbit connection: %v", err)
	}
	err = initExchange(ch)
	if err != nil {
		log.Fatalf("failed to initialize exchanges %v", err)
	}
	err = initQueue(ch)
	if err != nil {
		log.Fatalf("failed to initialize queues %v", err)
	}
}

func initExchange(ch *amqp.Channel) error {

	err := ch.ExchangeDeclare(
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
	err = ch.ExchangeDeclare(
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

func initQueue(ch *amqp.Channel) error {
	q, err := ch.QueueDeclare(GamesPartitionerQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not declare games-partitioner queue: %w", err)
	}

	err = ch.QueueBind(
		q.Name,
		"",
		GamesExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not bind to games-partitioner queue: %w", err)
	}

	q, err = ch.QueueDeclare(GamesFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not declare games-partitioner queue: %w", err)
	}

	err = ch.QueueBind(
		q.Name,
		"",
		GamesExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not bind to games-partitioner queue: %w", err)
	}

	q, err = ch.QueueDeclare(ReviewsFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not declare games-partitioner queue: %w", err)
	}

	err = ch.QueueBind(
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
