package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"log"
	"net"
)

const ReviewExchange string = "reviews"
const GamesExchange string = "games"

func (g *gateway) startDataHandler() {
	address := fmt.Sprintf(":%v", g.config.dataEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to bind socket: %v", err)
	}

	g.initRabbit()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()
			err := g.handleClientData(conn)
			if err != nil {
				log.Printf("error while handling client: %v", err)
			}
		}(conn)
	}
}

func (g *gateway) initRabbit() {
	rabbitChan, err := g.rabbitConn.Channel()
	if err != nil {
		log.Fatalf("failed to bind rabbit connection: %v", err)
	}

	err = rabbitChan.ExchangeDeclare(
		ReviewExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare reviews exchange: %v", err)
	}
	err = rabbitChan.ExchangeDeclare(
		GamesExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to bind games exchange: %v", err)
	}
}

func (g *gateway) handleClientData(conn net.Conn) error {
	m := protocol.NewMarshaller(conn)
	unm := protocol.NewUnmarshaller(conn)

	anyMsg, err := unm.ReceiveMessage()
	if err != nil {
		return err
	}
	helloMsg, ok := anyMsg.(*protocol.DataHello)
	if !ok {
		return fmt.Errorf("expected DataHello message, received %T", anyMsg)
	}

	// todo: validate client id
	_ = helloMsg

	err = m.SendMessage(&protocol.DataAccept{})
	if err != nil {
		return err
	}

	anyMsg, err = unm.ReceiveMessage()
	if err != nil {
		return err
	}
	_, ok = anyMsg.(*protocol.Prepare)
	if !ok {
		return fmt.Errorf("expected Prepare message, received %T", anyMsg)
	}

gameLoop:
	for {
		msg, err := unm.ReceiveMessage()
		if err != nil {
			return err
		}

		switch msg.(type) {
		case *protocol.Batch:
		case *protocol.Finish:
			break gameLoop
		default:
			return fmt.Errorf("expected Batch or Finish message, received %T", msg)
		}
	}

	anyMsg, err = unm.ReceiveMessage()
	if err != nil {
		return err
	}
	_, ok = anyMsg.(*protocol.Prepare)
	if !ok {
		return fmt.Errorf("expected Prepare message, received %T", anyMsg)
	}

reviewLoop:
	for {
		msg, err := unm.ReceiveMessage()
		if err != nil {
			return err
		}

		switch msg.(type) {
		case *protocol.Batch:
		case *protocol.Finish:
			break reviewLoop
		default:
			return fmt.Errorf("expected Batch or Finish message, received %T", msg)
		}
	}

	return nil
}
