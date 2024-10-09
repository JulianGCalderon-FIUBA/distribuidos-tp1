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

func (m *Middleware) Init(exchanges map[string]string, queues []queueConfig) error {
	err := m.initExchanges(exchanges)
	if err != nil {
		return fmt.Errorf("failed to initialize exchanges %w", err)
	}
	err = m.initQueues(queues)
	if err != nil {
		return fmt.Errorf("failed to initialize queues %w", err)
	}

	return nil
}

func (m *Middleware) initExchanges(exchanges map[string]string) error {
	for exchange, kind := range exchanges {
		err := m.ch.ExchangeDeclare(
			exchange,
			kind,
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to bind exchange %s: %w", exchange, err)
		}
	}

	return nil
}

func (m *Middleware) initQueues(queues []queueConfig) error {
	for _, queue := range queues {
		q, err := m.ch.QueueDeclare(queue.name,
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
			queue.routingKey,
			queue.exchange,
			false,
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Middleware) InitResultsQueue() error {
	_, err := m.ch.QueueDeclare(ResultsQueue,
		false,
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
