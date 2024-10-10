package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/utils"
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

func (g *gateway) startConnectionHandler(ctx context.Context) (err error) {
	address := fmt.Sprintf(":%d", g.config.ConnectionEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("Failed to bind socket: %v", err)
	}

	closer := utils.SpawnCloser(ctx, listener)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	err = g.m.InitResultsQueue()
	if err != nil {
		return fmt.Errorf("Failed to initialize results queue: %v", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("Failed to accept connection: %v", err)
		}
		g.incrementActiveClients()

		go func() {
			if err := g.handleClient(ctx, conn); err != nil {
				log.Errorf("Error while handling client: %v", err)
			}
		}()
	}
}

func (g *gateway) handleClient(ctx context.Context, netConn net.Conn) error {
	var err error
	conn := protocol.NewConn(netConn)

	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

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

		err = conn.Send(protocol.AcceptRequest{
			ClientID: uint64(clientId),
		})
		if err != nil {
			return err
		}
		err = g.receiveResults(ctx, conn)
		if err != nil {
			return err
		}
	}
}

func (g *gateway) receiveResults(ctx context.Context, conn *protocol.Conn) error {
	deliveryCh, err := g.m.ReceiveFromQueue(middleware.Results)
	if err != nil {
		return err
	}

	results := 0

	for {
		select {
		case d := <-deliveryCh:
			recv, err := middleware.Deserialize[any](d.Body)
			if err != nil {
				log.Errorf("Failed to deserialize: %v", err)
				err = d.Nack(false, false)
				if err != nil {
					return err
				}
			}
			log.Infof("Received results")
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
		case <-ctx.Done():
			return nil
		}
	}
}
