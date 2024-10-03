package middleware

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) SendToExchange(msg any, exchange string) error {
	buf, err := Serialize(msg)
	if err != nil {
		return err
	}

	return m.ch.PublishWithContext(context.Background(),
		exchange,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        buf,
		},
	)
}
