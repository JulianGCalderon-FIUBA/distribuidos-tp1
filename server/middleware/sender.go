package middleware

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m *Middleware) SendBatchGame(batchGame BatchGame, routingKey string) error {
	serialized, err := batchGame.Serialize()
	if err != nil {
		return fmt.Errorf("could not serialize batch game: %w", err)
	}

	return m.sendBatch(serialized, GamesExchange, routingKey)
}

func (m *Middleware) SendBatchReview(batchReview BatchReview) error {
	serialized, err := batchReview.Serialize()
	if err != nil {
		return fmt.Errorf("could not serialize batch review: %w", err)
	}

	return m.sendBatch(serialized, ReviewExchange, "")
}

func (m *Middleware) sendBatch(batch []byte, exchange string, routingKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.ch.PublishWithContext(ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        batch,
		},
	)

	if err != nil {
		return fmt.Errorf("could not publish batch review: %w", err)
	}

	return nil
}
