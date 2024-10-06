package aggregator

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"fmt"

	logging "github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
)

var log = logging.MustGetLogger("log")

// This interface contains business logic
type Handler[T any] interface {
	// Given a record T, process it.
	Aggregate(record T) error
	// Called when EOF is reached.
	Conclude(ch *amqp.Channel)
}

type Config struct {
	RabbitIP string
	// Name of the queue to read from
	Input string
}

// Aggregator structure, abstracting away queue system details
// Receives an input queue, and proceses each row received fro mthe queue
//
// Processes each row of type T until EOF arrives
type Aggregator[T any] struct {
	cfg        Config
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	handler    Handler[T]
}

func NewAggregator[T any](cfg Config, h Handler[T]) (*Aggregator[T], error) {
	addr := fmt.Sprintf("amqp://guest:guest@%v:5672/", cfg.RabbitIP)
	r, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}

	c, err := r.Channel()
	if err != nil {
		return nil, err
	}

	_, err = c.QueueDeclare(
		cfg.Input,
		false,
		false,
		false,
		false,
		nil)
	if err != nil {
		return nil, err
	}

	return &Aggregator[T]{
		cfg:        cfg,
		rabbitConn: r,
		rabbitCh:   c,
		handler:    h,
	}, nil
}

func (f *Aggregator[T]) Run(ctx context.Context) error {
	dch, err := f.rabbitCh.Consume(
		f.cfg.Input,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

loop:
	for d := range dch {
		batch, err := middleware.Deserialize[middleware.Batch[T]](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch %v", err)

			err = d.Nack(false, false)
			if err != nil {
				return err
			}

			continue
		}

		for _, record := range batch.Data {
			err := f.handler.Aggregate(record)
			if err != nil {
				err = d.Nack(false, false)
				if err != nil {
					return err
				}
				continue loop
			}
		}

		err = d.Ack(false)
		if err != nil {
			return err
		}

		// todo: handle disordered batches
		if batch.EOF {
			log.Info("Received EOF from client")
			f.handler.Conclude(f.rabbitCh)
		}
	}

	return nil
}
