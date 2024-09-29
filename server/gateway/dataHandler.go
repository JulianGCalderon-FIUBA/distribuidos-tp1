package main

import (
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"fmt"
	"log"
	"net"
)

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
		middleware.ReviewExchange,
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
		middleware.GamesExchange,
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

	// todo: receive data

	return nil
}
