package middleware

import (
	"context"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"net"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config[T Handler] struct {
	// For each client, the builder is called to initialize a new builder
	Builder HandlerBuilder[T]
	// Each queue is registered to a particular HandlerFunc
	Endpoints map[string]HandlerFunc[T]

	Port uint64
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

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		n.sendAlive(ctx)
	}()
	defer wg.Wait()

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
	if ch.FinishFlag {
		n.freeResources(clientID, h)
	}

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
	log.Infof("Freeing resources for client %v", clientID)
	err := h.Free()
	if err != nil {
		log.Errorf("Error freeing handler files: %v", err)
	}
	delete(n.clients, clientID)
}

func (n *Node[T]) sendAlive(ctx context.Context) {
	addr := net.UDPAddr{
		Port: int(n.config.Port),
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Errorf("failed to start listener on port %d: %w", n.config.Port, err)
	}

	log.Infof("Listening at %v", addr.String())

	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	for {
		buf := make([]byte, 1024)
		nRead, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Read error: %v", err)
			return
		}

		msg := buf[:nRead]
		log.Infof("Received from %v: %s", raddr, msg)

		go func(conn *net.UDPConn, raddr *net.UDPAddr, msg []byte) {
			_, err := conn.WriteToUDP([]byte("ACK"), raddr)
			if err != nil {
				fmt.Printf("Write err: %v", err)
			}
		}(conn, raddr, msg)
	}
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
