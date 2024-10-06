package filter

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"errors"
	"fmt"

	logging "github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
)

var log = logging.MustGetLogger("log")

type (
	RoutingKey string
	QueueName  string
	ClientID   uint64
)

// This interface contains business logic
type Handler[T any] interface {
	// Given a record T, returns which routing key
	// to send it to.
	Filter(record T) RoutingKey
}

type Config struct {
	RabbitIP string
	// Name of the node, used for the declared exchange
	Name string
	// Name of the queue to read from
	Input string
	// Output configuration: contains routing keys
	// and queues binded to them
	Output map[RoutingKey][]QueueName
}

// Filter structure, abstracting away queue system details
// Receives an input queue, a name, and a set of output queues,
// and builds the following topology.
//
// :              |--> Q1(K1)
// : [Q] --> [X]--|--> Q2(K2)
// :              |--> Q3(K2)
// :              |--> Q4(K3)
//
// Dispatches every row from Q to exchange X with routing key
// according to Filter function from Handler.
type Filter[T any] struct {
	cfg        Config
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	handler    Handler[T]
}

func NewFilter[T any](cfg Config, h Handler[T]) (*Filter[T], error) {
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

	err = c.ExchangeDeclare(
		cfg.Name,
		amqp.ExchangeDirect,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	for q, ks := range transpose(cfg.Output) {
		_, err = c.QueueDeclare(
			string(q),
			false,
			false,
			false,
			false,
			nil)
		if err != nil {
			return nil, err
		}

		for _, k := range ks {
			err = c.QueueBind(
				string(q),
				string(k),
				cfg.Name,
				false,
				nil,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	return &Filter[T]{
		cfg:        cfg,
		rabbitConn: r,
		rabbitCh:   c,
		handler:    h,
	}, nil
}

func (f *Filter[T]) Run(ctx context.Context) error {
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

	partitions := make(map[RoutingKey]middleware.Batch[T])
	statistics := make(map[RoutingKey]int)

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
			rk := f.handler.Filter(record)

			entry := partitions[rk]
			entry.Data = append(entry.Data, record)
			partitions[rk] = entry
		}

		for rk, partition := range partitions {
			partition.BatchID = batch.BatchID
			partition.ClientID = batch.ClientID
			partition.EOF = batch.EOF

			statistics[rk] += len(partition.Data)

			buf, err := middleware.Serialize(partition)
			if err != nil {
				nackErr := d.Nack(false, false)
				return errors.Join(err, nackErr)
			}

			err = f.rabbitCh.Publish(
				f.cfg.Name,
				string(rk),
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

			partition.Data = partition.Data[:0]
			partitions[rk] = partition
		}

		err = d.Ack(false)
		if err != nil {
			return err
		}

		// todo: handle disordered batches
		if batch.EOF {
			log.Info("Received EOF from client")
			for rk, stats := range statistics {
				log.Infof("Sent %v records to key %v", stats, rk)
			}
		}
	}

	return nil
}

func transpose(m map[RoutingKey][]QueueName) map[QueueName][]RoutingKey {
	t := make(map[QueueName][]RoutingKey)

	for k, qs := range m {
		for _, q := range qs {
			t[q] = append(t[q], k)
		}
	}

	return t
}
