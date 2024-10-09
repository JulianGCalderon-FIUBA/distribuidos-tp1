package node

import (
	"context"
	"errors"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

var EOF = errors.New("EOF")

// This interface contains business logic
type Handler interface {
	// Function to apply to each message received.
	// If returns EOF, then the node will finish.
	// On any other error, the message is nacked.
	// If no errors, the message is acked.
	Apply(ch *amqp.Channel, data []byte) error
}

type ExchangeConfig struct {
	Name        string
	Type        string
	QueuesByKey map[string][]string
}

type Config struct {
	RabbitIP string
	// Exchanges to declare, and queues binded to them
	Exchanges []ExchangeConfig
	// Queue to read from
	Queue string
}

type Node struct {
	cfg     Config
	ch      *amqp.Channel
	handler Handler
}

func NewNode(cfg Config, h Handler) (*Node, error) {
	addr := fmt.Sprintf("amqp://guest:guest@%v:5672/", cfg.RabbitIP)
	conn, err := amqp.Dial(addr)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = declare(ch, cfg.Exchanges)
	if err != nil {
		return nil, err
	}

	return &Node{
		cfg:     cfg,
		ch:      ch,
		handler: h,
	}, nil
}

func (n *Node) Run(ctx context.Context) error {
	dch, applyErr := n.ch.Consume(
		n.cfg.Queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if applyErr != nil {
		return applyErr
	}

	for {
		select {
		case d := <-dch:
			applyErr = n.handler.Apply(n.ch, d.Body)
			switch applyErr {
			case EOF:
				return d.Ack(false)
			case nil:
				continue
			default:
				err := d.Nack(false, false)
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():
			return nil
		}

	}
}

func declare(ch *amqp.Channel, exchanges []ExchangeConfig) error {
	for _, exchange := range exchanges {
		err := ch.ExchangeDeclare(
			exchange.Name,
			exchange.Type,
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return err
		}

		for queue, keys := range transpose(exchange.QueuesByKey) {
			_, err = ch.QueueDeclare(
				queue,
				false,
				false,
				false,
				false,
				nil)
			if err != nil {
				return err
			}

			for _, key := range keys {
				err = ch.QueueBind(
					queue,
					key,
					exchange.Name,
					false,
					nil,
				)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func transpose(queuesByKey map[string][]string) (keysByQueue map[string][]string) {
	keysByQueue = make(map[string][]string)

	for k, qs := range queuesByKey {
		for _, q := range qs {
			keysByQueue[q] = append(keysByQueue[q], k)
		}
	}

	return keysByQueue
}
