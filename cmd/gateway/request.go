package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"net"
	"sync"
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

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("Failed to accept connection: %v", err)
		}
		clientCounter += 1
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			err := g.handleClient(ctx, conn, clientID)
			if err != nil {
				log.Errorf("Error while handling client: %v", err)
			}
		}(clientCounter)
	}
}

func (g *gateway) handleClient(ctx context.Context, netConn net.Conn, clientID int) error {
	var err error
	conn := protocol.NewConn(netConn)
	// no estoy muy segura si este close alcanza, acepto sugerencias
	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	var hello protocol.RequestHello
	err = conn.Recv(&hello)
	if err != nil {
		return err
	}

	log.Infof("Received client hello: %v", clientID)

	ch := make(chan protocol.Results)

	g.mu.Lock()
	g.clients[clientID] = ch
	g.mu.Unlock()

	err = conn.Send(protocol.AcceptRequest{
		ClientID: uint64(clientID),
	})
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case result, more := <-ch:
			if !more {
				log.Infof("Sent all results to client %v, closing connection", clientID)
				g.mu.Lock()
				delete(g.clients, clientID)
				g.mu.Unlock()
				return nil
			}
			err := conn.SendAny(result)
			if err != nil {
				log.Infof("Failed to send result to client")
			}
		}

	}
}
