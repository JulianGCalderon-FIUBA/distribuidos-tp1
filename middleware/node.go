package middleware

import (
	"context"
	"distribuidos/tp1/restarter-protocol"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"net"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const MAX_PACKAGE_SIZE = 1024

type Config[T Handler] struct {
	// For each client, the builder is called to initialize a new builder
	Builder HandlerBuilder[T]
	// Each queue is registered to a particular HandlerFunc
	Endpoints map[string]HandlerFunc[T]

	OutputConfig Output
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
		err := n.sendAlive(ctx)
		if err != nil {
			log.Errorf("%v", err)
		}
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
			return nil
		}
	}
}

func (n *Node[T]) processDelivery(d Delivery) error {
	clientID := int(d.Headers["clientID"].(int32))
	cleanFlag := d.Headers["clean"].(bool)

	if n.notifyFallenNode(clientID, cleanFlag) {
		return nil
	}

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

func (n *Node[T]) notifyFallenNode(clientID int, cleanFlag bool) bool {
	if clientID == -1 {
		if len(n.clients) > 0 {
			log.Infof("Cleaning all system resources")
			for c := range n.clients {
				n.freeResources(c)
			}
		}
	} else if cleanFlag {
		_, ok := n.clients[clientID]
		if ok {
			log.Infof("Client %v crashed, clean all resources for it", clientID)
			n.freeResources(clientID)
		}
	} else {
		return false
	}

	if len(n.config.OutputConfig.Exchange) == 0 && len(n.config.OutputConfig.Keys) == 0 { // reached results node
		log.Infof("Finished cleaning system resources")
		return true
	}

	for _, key := range n.config.OutputConfig.Keys {
		ch := &Channel{
			Ch:        n.ch,
			ClientID:  clientID,
			CleanFlag: cleanFlag,
		}
		err := ch.Send([]byte{}, n.config.OutputConfig.Exchange, key)
		if err != nil {
			log.Error("Failed to propagate to pipeline: %v", err)
		}
	}
	return true
}

func (n *Node[T]) freeResources(clientID int) {
	log.Infof("Freeing resources for client %v", clientID)

	h := n.clients[clientID]

	err := h.Free()
	if err != nil {
		log.Errorf("Error freeing handler files: %v", err)
	}
	delete(n.clients, clientID)
}

func (n *Node[T]) sendAlive(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", utils.NODE_UDP_ADDR)
	if err != nil {
		return fmt.Errorf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to start listener on address %v: %v", utils.NODE_UDP_ADDR, err)
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
			return fmt.Errorf("Read error: %v", err)
		}

		decoded, err := restarter.Decode(buf)
		if err != nil {
			return fmt.Errorf("Failed to decode message: %v", err)
		}
		// log.Infof("Received KeepAlive from: %v", rAddr)

		err = n.send(conn, rAddr, decoded.Id)
		if err != nil {
			return fmt.Errorf("Failed to send ACK: %v", err)
		}
	}
}

func (n *Node[T]) send(conn *net.UDPConn, rAddr *net.UDPAddr, msgId uint64) error {
	packet := restarter.Packet{
		Id:  msgId,
		Msg: restarter.Ack{}}

	encoded, err := packet.Encode()
	if err != nil {
		return err
	}

	_, err = conn.WriteToUDP(encoded, rAddr)
	if err != nil {
		return err
	}
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
