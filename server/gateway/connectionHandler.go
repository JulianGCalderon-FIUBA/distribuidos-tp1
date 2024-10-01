package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"io"
	"log"
	"net"
)

func (g *gateway) getActiveClients() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.activeClients
}

func (g *gateway) incrementActiveClients() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.activeClients++
}

func (g *gateway) startConnectionHandler() {
	address := fmt.Sprintf(":%d", g.config.ConnectionEndpointPort)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatalf("failed to bind socket: %v", err)
	}
	defer listener.Close()

	fmt.Println("Gateway is listening on port ", g.config.ConnectionEndpointPort)

	for {
		conn, err := listener.Accept()
		g.incrementActiveClients()

		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		fmt.Println("Client connected: ", conn.RemoteAddr().String())

		go func() {
			if err := g.handleClient(conn); err != nil {
				log.Printf("Error handling client: %v", err)
			}
		}()
	}
}

func (g *gateway) handleClient(conn net.Conn) error {
	defer conn.Close()

	m := protocol.NewMarshaller(conn)
	unm := protocol.NewUnmarshaller(conn)

	for {
		msg, err := unm.ReceiveMessage()
		if err != nil {
			if err == io.EOF {
				log.Printf("Client disconnected")
				return nil

			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		request, ok := msg.(*protocol.RequestHello)
		if !ok {
			return fmt.Errorf("expected RequestHello message, received: %T", msg)
		}

		log.Printf("Game size: %d\n", request.GameSize)
		log.Printf("Review size: %d\n", request.ReviewSize)

		clientId := g.getActiveClients()

		err = m.SendMessage(&protocol.AcceptRequest{
			ClientID: uint64(clientId),
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %v", err)
		}
	}
}
