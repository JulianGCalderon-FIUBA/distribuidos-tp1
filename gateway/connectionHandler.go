package main

import (
	"distribuidos/tp1/middleware"
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

		go g.handleClient(conn)
	}
}

func (g *gateway) handleClient(conn net.Conn) {
	defer conn.Close()

	m := middleware.NewMessageHandler(conn)

	for {
		msg, err := m.ReceiveMessage()
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Client disconnected")
				return
			}
			log.Fatalf("Failed to read message: %v", err)
			return
		}

		fmt.Printf("Received: %s\n", msg)
	}
}
