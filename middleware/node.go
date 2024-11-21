package middleware

import (
	"context"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const MAX_PACKAGE_SIZE = 1024

type Config[T Handler] struct {
	// For each client, the builder is called to initialize a new builder
	Builder HandlerBuilder[T]
	// Each queue is registered to a particular HandlerFunc
	Endpoints map[string]HandlerFunc[T]

	Address string
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
	udpAddr, err := net.ResolveUDPAddr("udp", n.config.Address)
	if err != nil {
		log.Errorf("Failed to resolve address: %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Errorf("failed to start listener on address %d: %w", n.config.Address, err)
	}

	log.Infof("Listening in %v", udpAddr)

	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	for {
		buf := make([]byte, MAX_PACKAGE_SIZE)
		_, rAddr, err := conn.ReadFromUDP(buf)

		if err != nil {
			log.Errorf("Read error: %v", err)
			return
		}

		header := utils.MsgHeader{}

		_, err = binary.Decode(buf, binary.LittleEndian, &header)
		if err != nil {
			log.Errorf("Failed to decode header: %v", err)
			continue
		}
		log.Infof("Received KeepAlive from: %v", rAddr)

		err = n.send(conn, rAddr)
		if err != nil {
			log.Errorf("Failed to send ACK: %v", err)
		}
	}
}

func (n *Node[T]) send(conn *net.UDPConn, rAddr *net.UDPAddr) error {
	nodeName := strings.Split(n.config.Address, ":")[0]
	msg, err := binary.Append(nil, binary.LittleEndian, utils.MsgHeader{Ty: utils.Ack})
	if err != nil {
		return err
	}

	msg = append(msg, nodeName...)

	_, err = conn.WriteToUDP(msg, rAddr)
	if err != nil {
		return err
	}

	log.Infof("Sent Ack")
	return nil
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
