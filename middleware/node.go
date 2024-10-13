package middleware

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Each client should be handled completely indepently. Therefore,
// our middleware can hide this detail to the business layer.
// As some handlers require to keep state, we can't use the same instance for
// each client. Therefore, we need a HandlerBuilder function that creates
// a new handler when a new client comes.
type HandlerBuilder[T any] func(clientID int) T

// Some nodes need to listen from multiple queues. To allow this, we define
// HandlerFunc, which represents the handling of a message of a particular
// queue. If the user needs to handle multiple queues, it must define
// multiple HandlerFuncs
type HandlerFunc[T any] func(h *T, ch *Channel, data []byte) error

type Config[T any] struct {
	// For each client, the builder is called to initialize a new builder
	Builder HandlerBuilder[T]
	// Each queue is registered to a particular HandlerFunc
	Endpoints map[string]HandlerFunc[T]
}

type Node[T any] struct {
	config  Config[T]
	rabbit  *amqp.Connection
	ch      *amqp.Channel
	clients map[int]T
}

func NewNode[T any](config Config[T], rabbit *amqp.Connection) (*Node[T], error) {
	ch, err := rabbit.Channel()
	if err != nil {
		return nil, err
	}

	return &Node[T]{
		config:  config,
		rabbit:  rabbit,
		ch:      ch,
		clients: make(map[int]T),
	}, nil
}

func (n *Node[T]) Run(ctx context.Context) error {
	defer n.rabbit.Close()

	dch := make(chan Delivery)
	for queue := range n.config.Endpoints {
		err := n.Consume(ctx, queue, dch)
		if err != nil {
			return err
		}
	}

	for {
		select {
		case d := <-dch:
			err := n.processDelivery(d)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (n *Node[T]) processDelivery(d Delivery) error {
	// todo: get correct client handler
	clientID := 1

	h, ok := n.clients[clientID]
	if !ok {
		h = n.config.Builder(clientID)
		n.clients[clientID] = h
	}
	ch := &Channel{
		Ch:       n.ch,
		ClientID: clientID,
	}

	err := n.config.Endpoints[d.Queue](&h, ch, d.Body)
	if err != nil {
		log.Errorf("Failed to handle message %v", err)
		err = d.Nack(false, false)
		if err != nil {
			return err
		}
	}

	return d.Ack(false)
}

type Delivery struct {
	Queue string
	amqp.Delivery
}

func (n *Node[T]) Consume(ctx context.Context, queue string, deliveries chan<- Delivery) error {
	dch, err := n.ch.ConsumeWithContext(ctx, queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range dch {
			deliveries <- Delivery{
				Queue:    queue,
				Delivery: d,
			}
		}
	}()

	return nil
}
