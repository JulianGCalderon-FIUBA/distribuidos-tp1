package middleware

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config[T Handler] struct {
	// For each client, the builder is called to initialize a new builder
	Builder HandlerBuilder[T]
	// Each queue is registered to a particular HandlerFunc
	Endpoints map[string]HandlerFunc[T]
}

type Node[T Handler] struct {
	config  Config[T]
	rabbit  *amqp.Connection
	ch      *amqp.Channel
	clients map[int]T
}

func NewNode[T Handler](config Config[T], rabbit *amqp.Connection) (*Node[T], error) {
	ch, err := rabbit.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.Qos(1000, 0, false)
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
			for clientId, h := range n.clients {
				n.freeResources(clientId, h)
			}
			return nil
		}
	}
}

func (n *Node[T]) processDelivery(d Delivery) error {
	clientID := int(d.Headers["clientID"].(int32))

	h, ok := n.clients[clientID]
	if !ok {
		log.Infof("Building handler for client %v", clientID)
		h = n.config.Builder(clientID)
		n.clients[clientID] = h
	}

	ch := &Channel{
		Ch:         n.ch,
		ClientID:   clientID,
		FinishFlag: false,
	}

	err := n.config.Endpoints[d.Queue](h, ch, d.Body)
	// todo: persistir cuales clientes terminaron
	// todo: free resources
	// if ch.FinishFlag {
	// n.freeResources(clientID, h)
	// }

	if err != nil {
		log.Errorf("Failed to handle message %v", err)
		err = d.Nack(false, false)
		if err != nil {
			return err
		}
	}

	return d.Ack(false)
}

func (n *Node[T]) freeResources(clientID int, h T) {
	log.Infof("Cleaning resources for client %v", clientID)
	err := h.Free()
	if err != nil {
		log.Errorf("Error freeing handler files: %v", err)
	}
	delete(n.clients, clientID)
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
