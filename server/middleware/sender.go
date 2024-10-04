package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) Send(msg any, exchange, key string) error {
	buf, err := Serialize(msg)
	if err != nil {
		return err
	}

	return m.ch.Publish(
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        buf,
		},
	)
}
