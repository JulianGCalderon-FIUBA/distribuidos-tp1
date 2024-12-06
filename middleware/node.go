package middleware

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/restarter-protocol"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const MAX_PACKAGE_SIZE = 1024

type Config[T Handler] struct {
	// For each client, the builder is called to initialize a new builder
	Builder HandlerBuilder[T]
	// Each queue is registered to a particular HandlerFunc
	Endpoints map[string]HandlerFunc[T]
	// Registers all node outputs
	OutputConfig Output
}

type Node[T Handler] struct {
	config         Config[T]
	rabbit         *amqp.Connection
	ch             *amqp.Channel
	clients        map[int]T
	db             *database.Database
	doneClientsSet *DiskSet
}

func NewNode[T Handler](config Config[T], rabbit *amqp.Connection) (*Node[T], error) {
	ch, err := rabbit.Channel()
	if err != nil {
		return nil, err
	}
	err = ch.Confirm(false)
	if err != nil {
		return nil, err
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, err
	}

	db, err := database.NewDatabase("node")
	utils.Expect(err, "unrecoverable error")

	doneClientsSet := NewSetDisk("ids")
	err = doneClientsSet.LoadDisk(db)
	utils.Expect(err, "unrecoverable error")

	cleanAllClients(doneClientsSet)

	return &Node[T]{
		config:         config,
		rabbit:         rabbit,
		ch:             ch,
		clients:        make(map[int]T),
		db:             db,
		doneClientsSet: doneClientsSet,
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
	cleanAction := int(d.Headers["cleanAction"].(int32))

	if cleanAction != NotClean {
		err := n.notifyFallenNode(clientID, cleanAction)
		if err != nil {
			return err
		}
		return d.Ack(false)
	}

	if n.doneClientsSet.Seen(clientID) {
		return d.Ack(false)
	}

	h, ok := n.clients[clientID]
	if !ok {
		log.Infof("Building handler for client %v", clientID)
		var err error
		h, err = n.config.Builder(clientID)
		if err != nil {
			log.Errorf("Error building handler: %v", err)
			return d.Reject(false)
		}
		n.clients[clientID] = h
	}

	ch := &Channel{
		Ch:          n.ch,
		ClientID:    clientID,
		FinishFlag:  false,
		CleanAction: NotClean,
	}

	err := n.config.Endpoints[d.Queue](h, ch, d.Body)

	if ch.FinishFlag {
		utils.MaybeExit(0.2)
		err = n.freeResources(clientID, h)
		if err != nil {
			return err
		}
		utils.MaybeExit(0.2)
	}

	if err != nil {
		log.Errorf("Failed to handle message %v", err)
		err = d.Nack(false, false)
		if err != nil {
			return err
		}
	}

	utils.MaybeExit(0.0002)

	return d.Ack(false)
}

func (n *Node[T]) notifyFallenNode(clientID int, cleanAction int) error {
	switch cleanAction {
	case CleanAll:
		if len(n.clients) == 0 {
			break
		}
		log.Infof("Cleaning all system resources")
		for i := clientID; i > 0; i-- {
			if h, ok := n.clients[i]; ok {
				err := n.freeResources(i, h)
				if err != nil {
					log.Errorf("Error freeing resources for client %v: %v", i, err)
				}
			}
		}

	case CleanId:
		if h, ok := n.clients[clientID]; ok {
			log.Infof("Client %v disconnected, cleaning its resources", clientID)
			err := n.freeResources(clientID, h)
			if err != nil {
				log.Errorf("Error freeing resources for client %v: %v", clientID, err)
			}
		}
	}

	n.propagateFallenNode(clientID, cleanAction)
	return nil
}

func (n *Node[T]) propagateFallenNode(clientID int, cleanAction int) {
	for _, key := range n.config.OutputConfig.Keys {
		ch := &Channel{
			Ch:          n.ch,
			ClientID:    clientID,
			CleanAction: cleanAction,
		}
		err := ch.Send([]byte{}, n.config.OutputConfig.Exchange, key)
		if err != nil {
			log.Error("Failed to propagate to pipeline: %v", err)
		}
	}
}

func (n *Node[T]) freeResources(clientID int, h Handler) error {
	log.Infof("Freeing resources for client %v", clientID)
	snapshot, err := n.db.NewSnapshot()
	if err != nil {
		return err
	}

	err = n.doneClientsSet.MarkDisk(snapshot, clientID)
	if err != nil {
		cerr := snapshot.Abort()
		utils.Expect(cerr, "unrecoverable error")

		return err
	}
	cerr := snapshot.Commit()
	utils.Expect(cerr, "unrecoverable error")

	err = h.Free()
	if err != nil {
		return err
	}
	delete(n.clients, clientID)

	return nil
}

func cleanAllClients(clientsSet *DiskSet) {
	for client := range clientsSet.ids {
		os.RemoveAll(fmt.Sprintf("client-%v", client))
	}
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
