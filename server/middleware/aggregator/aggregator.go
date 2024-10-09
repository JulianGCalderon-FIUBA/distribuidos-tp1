package aggregator

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"errors"
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
	// Result is sent to Output queue
	Conclude() ([]any, error)
}

type Config struct {
	RabbitIP string
	// Name of the queue to read from
	Input string
	// Name of the queue to send result to
	Output string
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

	_, err = c.QueueDeclare(
		cfg.Output,
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

	var latestBatchID int
	missingBatchIDs := make(map[int]struct{})
	seen := make(map[int]struct{})
	fakeEof := false

loop:
	for d := range dch {
		batch, err := middleware.Deserialize[middleware.Batch[T]](d.Body)

		if _, ok := seen[batch.BatchID]; ok {
			log.Infof("Already received %v", batch.BatchID)
		} else {
			seen[batch.BatchID] = struct{}{}
		}

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

		delete(missingBatchIDs, batch.BatchID)
		for i := latestBatchID + 1; i < batch.BatchID; i++ {
			missingBatchIDs[i] = struct{}{}
		}

		latestBatchID = max(batch.BatchID, latestBatchID)
		if batch.EOF {
			fakeEof = true
		}
		if fakeEof && len(missingBatchIDs) == 0 {
			results, err := f.handler.Conclude()
			if err != nil {
				nackErr := d.Nack(false, false)
				return errors.Join(err, nackErr)
			}
			for _, result := range results {
				buf, err := middleware.Serialize(result)
				if err != nil {
					nackErr := d.Nack(false, false)
					return errors.Join(err, nackErr)
				}

				err = f.rabbitCh.Publish(
					"",
					f.cfg.Output,
					false,
					false,
					amqp.Publishing{
						ContentType: "text/plain",
						Body:        buf,
					},
				)
				if err != nil {
					nackErr := d.Nack(false, false)
					return errors.Join(err, nackErr)
				}
			}
		}

		err = d.Ack(false)
		if err != nil {
			return err
		}
	}

	return nil
}
