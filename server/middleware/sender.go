package middleware

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) SendToExchange(msg any, exchange, key string) error {
	buf, err := Serialize(msg)
	if err != nil {
		return err
	}

	return m.ch.PublishWithContext(context.Background(),
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

func (m *Middleware) SendToPartition(msg any, exchange string, partitionId int) error {
	queue := fmt.Sprintf("%v-%v", exchange, partitionId)
	return m.SendToExchange(msg, exchange, queue)
}
