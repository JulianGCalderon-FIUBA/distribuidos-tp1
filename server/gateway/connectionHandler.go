package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"io"
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
		log.Fatalf("Failed to bind socket: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			continue
		}
		g.incrementActiveClients()

		go func() {
			if err := g.handleClient(conn); err != nil {
				log.Errorf("Error while handling client: %v", err)
			}
		}()
	}
}

func (g *gateway) handleClient(netConn net.Conn) error {
	conn := protocol.NewConn(netConn)
	defer conn.Close()

	clientId := g.getActiveClients()
	log.Infof("Client connected with id: %v", clientId)

	for {
		var hello protocol.RequestHello
		err := conn.Recv(&hello)
		if err != nil {
			if err == io.EOF {
				log.Infof("Client disconnected")
				return nil

			}
			return err
		}

		log.Infof("Received game size: %v", hello.GameSize)
		log.Infof("Received review size: %v", hello.ReviewSize)

		err = conn.Send(&protocol.AcceptRequest{
			ClientID: uint64(clientId),
		})
		if err != nil {
			return err
		}
	}
}
