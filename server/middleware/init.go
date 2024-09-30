package middleware

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Init(conn *amqp.Connection) {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to bind rabbit connection: %v", err)
	}

	err = ch.ExchangeDeclare(
		ReviewExchange,
		amqp.ExchangeFanout,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare reviews exchange: %v", err)
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
		log.Fatalf("failed to bind games exchange: %v", err)
	}
}
