package main

import (
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"errors"
	"fmt"
	"io"
	"net"
)

const MAX_RESULTS = 5

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

	err = g.m.InitResultsQueue()
	if err != nil {
		log.Errorf("Failed to initialize results queue: %v", err)
		return
	}

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
		err = g.receiveResults(conn)
		if err != nil {
			return err
		}
	}
}

func (g *gateway) receiveResults(conn *protocol.Conn) error {
	deliveryCh, err := g.m.ReceiveFromQueue(middleware.ResultsQueue)
	if err != nil {
		return err
	}

	results := 0

	for d := range deliveryCh {
		recv, err := middleware.Deserialize[any](d.Body)
		if err != nil {
			nackErr := d.Nack(false, false)
			return errors.Join(err, nackErr)
		}
		switch r := recv.(type) {
		case protocol.Q1Results:
			results += 1
		case protocol.Q2Results:
			results += 1
		case protocol.Q3Results:
			results += 1
		case protocol.Q4Results:
			if r.EOF {
				results += 1
			}
		case protocol.Q5Results:
			results += 1
		}

		err = d.Ack(false)
		if err != nil {
			return fmt.Errorf("failed to ack result: %v", err)
		}
		err = conn.SendAny(recv)
		if err != nil {
			return fmt.Errorf("failed to send result: %v", err)
		}

		if results == MAX_RESULTS {
			log.Infof("Sent all results to client")
			break
		}
	}
	return nil
}
