package middleware

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Middleware struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewMiddleware(ip string) (*Middleware, error) {
	addr := fmt.Sprintf("amqp://guest:guest@%v:5672/", ip)
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbit: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to bind rabbit connection: %w", err)
	}
	return &Middleware{
		conn: conn,
		ch:   ch,
	}, nil
}

func (m *Middleware) Init() error {
	err := m.initExchange()
	if err != nil {
		return fmt.Errorf("failed to initialize exchanges %w", err)
	}
	err = m.initQueue()
	if err != nil {
		return fmt.Errorf("failed to initialize queues %w", err)
	}

	return nil
}

func (m *Middleware) initExchange() error {
	err := m.ch.ExchangeDeclare(
		ReviewExchange,
		amqp.ExchangeFanout,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	err = m.ch.ExchangeDeclare(
		GamesExchange,
		amqp.ExchangeFanout,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (m *Middleware) initQueue() error {
	q, err := m.ch.QueueDeclare(GamesPartitionerQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.QueueBind(
		q.Name,
		"",
		GamesExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	q, err = m.ch.QueueDeclare(GamesFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.QueueBind(
		q.Name,
		"",
		GamesExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	q, err = m.ch.QueueDeclare(ReviewsFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.QueueBind(
		q.Name,
		"",
		ReviewExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (m *Middleware) InitReviewFilter() error {
	// Receiving queue
	q, err := m.ch.QueueDeclare(ReviewsFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.QueueBind(
		q.Name,
		"",
		ReviewExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Sending exchanges
	err = m.ch.ExchangeDeclare(
		ReviewsScoreFilterExchange,
		amqp.ExchangeDirect,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Sending queues
	q, err = m.ch.QueueDeclare(FiftyThReviewsQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.QueueBind(
		q.Name,
		PositiveReviews,
		ReviewsScoreFilterExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	q, err = m.ch.QueueDeclare(LanguageReviewsFilterQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.QueueBind(
		q.Name,
		NegativeReviews,
		ReviewsScoreFilterExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	q, err = m.ch.QueueDeclare(NinetyPercentileReviewsQueue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.QueueBind(
		q.Name,
		NegativeReviews,
		ReviewsScoreFilterExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}
