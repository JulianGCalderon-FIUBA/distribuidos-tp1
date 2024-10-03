package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) ReceiveFromQueue(name string) (<-chan amqp.Delivery, error) {
	return m.ch.Consume(name, "", true, false, false, false, nil)
}
