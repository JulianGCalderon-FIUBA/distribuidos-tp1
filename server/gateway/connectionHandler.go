package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"log"
	"net"
)

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

	m := protocol.NewMarshaller(conn)
	unm := protocol.NewUnmarshaller(conn)

	for {
		msg, err := unm.ReceiveMessage()
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Client disconnected")
				return
			}
			log.Fatalf("Failed to read message: %v", err)
			return
		}

		fmt.Printf("Received: %s\n", msg)

		err = m.SendMessage(&protocol.AcceptRequest{
			ClientID: 1,
		})
		if err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
	}
}
