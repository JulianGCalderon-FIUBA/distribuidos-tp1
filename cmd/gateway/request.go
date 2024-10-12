package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"net"
)

const MAX_RESULTS = 5

func (g *gateway) startRequestEndpoint(ctx context.Context) (err error) {
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

	clientCounter := 0

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("Failed to accept connection: %v", err)
		}
		clientCounter += 1

		// todo: add wait group
		go func(clientID int) {
			err := g.handleClient(ctx, conn, clientID)
			if err != nil {
				log.Errorf("Error while handling client: %v", err)
			}
		}(clientCounter)
	}
}

func (g *gateway) handleClient(_ context.Context, netConn net.Conn, clientID int) error {
	var err error
	conn := protocol.NewConn(netConn)

	var hello protocol.RequestHello
	err = conn.Recv(&hello)
	if err != nil {
		return err
	}

	log.Infof("Received client hello: %v", clientID)

	{
		// register connection for the result handler
		// todo: consider receiving results and handle connection here
		g.mu.Lock()
		g.clients[clientID] = conn
		g.mu.Unlock()
	}

	err = conn.Send(protocol.AcceptRequest{
		ClientID: uint64(clientID),
	})
	if err != nil {
		return err
	}

	return nil
}
