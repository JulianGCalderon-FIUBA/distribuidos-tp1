package main

import (
	"distribuidos/tp1/middleware"
	"fmt"
	"log"
	"net"
)

const PACKET_SIZE = 4

func (g *gateway) StartConnectionHandler() {
	address := fmt.Sprintf(":%d", g.config.connectionEndpointPort)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatalf("failed to bind socket: %v", err)
	}
	defer listener.Close()

	fmt.Println("Gateway is listening on port ", g.config.connectionEndpointPort)

	for {
		conn, err := listener.Accept()

		fmt.Println("Client connected: ", conn.RemoteAddr().String())

		if err != nil {
			log.Fatalf("Error:", err)
			continue
		}

		go g.handleClient(conn)
	}
}

func (g *gateway) handleClient(conn net.Conn) {
	defer conn.Close()

	m := middleware.NewMessageHandler(conn)

	for {
		msg, err := m.ReceiveMessage()
		if err != nil {
			log.Fatalf("Failed to read message: %v", err)
			return
		}

		fmt.Printf("Received: %s\n", msg)
	}
}
