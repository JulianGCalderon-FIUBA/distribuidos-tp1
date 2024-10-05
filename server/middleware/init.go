package middleware

import (
	"fmt"
	"log"
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

func (m *Middleware) Init(exchanges map[string]string, queues map[string]string) error {
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

func (m *Middleware) InitGenreFilter() error {
	err := m.initExchanges(GenreFilterExchanges)
	if err != nil {
		return fmt.Errorf("failed to initialize exchanges %w", err)
	}
	err = m.initQueues(GenreFilterQueues)
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
			true,
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

func (m *Middleware) initQueues(queues map[string]string) error {
	for queue, exchange := range queues {
		q, err := m.ch.QueueDeclare(queue,
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
			exchange,
			false,
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Middleware) InitGenresQueues(routingKeys []string) error {
	q, err := m.ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("could not declare genre queue: %w", err)
	}

	for _, genre := range routingKeys {
		log.Printf("Binding queue %s to exchange %s with routing key %s",
			q.Name, GenresExchange, genre)

		err = m.ch.QueueBind(
			q.Name,
			genre,
			GenresExchange,
			false,
			nil)

		if err != nil {
			return fmt.Errorf("could not bind genre queue: %w", err)
		}
	}
	return nil
}

func (m *Middleware) InitReviewFilter() error {
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
		true,
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
