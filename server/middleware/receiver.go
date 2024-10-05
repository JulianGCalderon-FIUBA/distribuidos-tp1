package middleware

import (
	"distribuidos/tp1/utils"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) Subscribe(queue string) <-chan amqp.Delivery {
	ch, err := m.ch.Consume(queue, "", false, false, false, false, nil)
	utils.Expect(err, "failed to consume from queue")
	return ch
}
