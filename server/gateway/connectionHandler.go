package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"log"
	"net"
)

func (g *gateway) startConnectionHandler() {
	address := fmt.Sprintf(":%d", g.config.connectionEndpointPort)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatalf("failed to bind socket: %v", err)
	}
	defer listener.Close()

	fmt.Println("Gateway is listening on port ", g.config.connectionEndpointPort)

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		fmt.Println("Client connected: ", conn.RemoteAddr().String())

		go func() {
			err := g.handleClient(conn)
			if err != nil {
				log.Printf("Error handling client: %v", err)
			}
		}()
	}
}

func (g *gateway) handleClient(conn net.Conn) error {
	defer conn.Close()
	g.activeClients++

	m := protocol.NewMarshaller(conn)
	unm := protocol.NewUnmarshaller(conn)

	for {
		msg, err := unm.ReceiveMessage()
		if err != nil {
			if err.Error() == "EOF" {
				log.Printf("Client disconnected")
				return nil

			}
			return fmt.Errorf("failed to read message: %v", err)
		}

		request, ok := msg.(*protocol.RequestHello)
		if !ok {
			return fmt.Errorf("expected RequestHello message, received: %T", msg)
		}

		log.Printf("Game size: %d\n", request.GameSize)
		log.Printf("Review size: %d\n", request.ReviewSize)

		err = m.SendMessage(&protocol.AcceptRequest{
			ClientID: uint64(g.activeClients),
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %v", err)
		}
	}
}
