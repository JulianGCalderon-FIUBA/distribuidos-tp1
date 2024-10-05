package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) Consume(queue string) (<-chan amqp.Delivery, error) {
	return m.ch.Consume(queue, "", false, false, false, false, nil)
}
