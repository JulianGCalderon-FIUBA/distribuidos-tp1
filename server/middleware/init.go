package middleware

import (
	"fmt"
	"strconv"

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

func (m *Middleware) InitReviewFilter() error {
	err := m.ch.ExchangeDeclare(
		ReviewExchange,
		amqp.ExchangeFanout,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Receiving queue
	q, err := m.ch.QueueDeclare(ReviewsQueue,
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

	// Sending exchange
	err = m.ch.ExchangeDeclare(
		ReviewsScoreFilterExchange,
		amqp.ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Sending queues
	q, err = m.ch.QueueDeclare(TopNAmountReviewsQueue,
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
		PositiveReviewKey,
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
		NegativeReviewKey,
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
		NegativeReviewKey,
		ReviewsScoreFilterExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (m *Middleware) InitLanguageFilter() error {
	err := m.ch.ExchangeDeclare(
		ReviewsScoreFilterExchange,
		amqp.ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Receiving queue
	q, err := m.ch.QueueDeclare(LanguageReviewsFilterQueue,
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
		NegativeReviewKey,
		ReviewsScoreFilterExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Sending exchange
	err = m.ch.ExchangeDeclare(
		ReviewsEnglishFilterExchange,
		amqp.ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Sending queue
	q, err = m.ch.QueueDeclare(NThousandEnglishReviewsQueue,
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
		ReviewsEnglishFilterExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (m *Middleware) InitPartitioner(input string, output string, partitionsNum int) error {
	_, err := m.ch.QueueDeclare(input,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = m.ch.ExchangeDeclare(
		output,
		amqp.ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for i := 0; i < partitionsNum; i++ {
		q, err := m.ch.QueueDeclare(fmt.Sprintf("%v-%v", output, i),
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return err
		}

		err = m.ch.QueueBind(q.Name,
			strconv.Itoa(i),
			output,
			false,
			nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func (m *Middleware) InitTopNHistoricAvgJoiner() error {
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

	_, err = m.ch.QueueDeclare(TopNHistoricAvgJQueue,
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
