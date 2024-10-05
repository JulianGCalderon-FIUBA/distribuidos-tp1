package middleware

import (
	"distribuidos/tp1/utils"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) Send(msg any, exchange, key string) error {
	buf, err := Serialize(msg)
	if err != nil {
		return err
	}

	err = m.ch.Publish(
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        buf,
		},
	)
	utils.Expect(err, "failed to publish message")
	return nil
}
