package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

func (g *gateway) updateClientCounter(clientCounter uint64) error {
	snapshot, err := g.db.NewSnapshot()
	if err != nil {
		return err
	}

	defer func() {
		switch err {
		case nil:
			cerr := snapshot.Commit()
			utils.Expect(cerr, "unrecoverable error")
		default:
			cerr := snapshot.Abort()
			utils.Expect(cerr, "unrecoverable error")
		}
	}()

	counterFile, err := snapshot.Update("client-counter")
	if err != nil {
		return err
	}

	_, err = counterFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	err = binary.Write(counterFile, binary.LittleEndian, clientCounter)
	if err != nil {
		return err
	}

	return nil
}

func (g *gateway) loadClientCounter() (uint64, error) {
	exists, err := g.db.Exists("client-counter")
	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, nil
	}

	counterFile, err := g.db.Get("client-counter")
	if err != nil {
		return 0, err
	}

	defer counterFile.Close()
	var clientCounter uint64

	err = binary.Read(counterFile, binary.LittleEndian, &clientCounter)
	if err != nil {
		return 0, err
	}

	return clientCounter, nil
}

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

	clientCounter, err := g.loadClientCounter()
	if err != nil {
		return fmt.Errorf("Failed to load client counter: %v", err)
	}

	g.clientCounter = clientCounter

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("Failed to accept connection: %v", err)
		}
		g.clientCounter += 1
		err = g.updateClientCounter(g.clientCounter)
		if err != nil {
			return fmt.Errorf("Failed to update client counter: %v", err)
		}

		wg.Add(1)
		clientID := g.clientCounter
		go func(clientID uint64) {
			defer wg.Done()
			err := g.handleClient(ctx, conn, int(clientID))
			if err != nil {
				log.Errorf("Error while handling client: %v", err)
			}
		}(clientID)
	}
}

func (g *gateway) handleClient(ctx context.Context, netConn net.Conn, clientID int) error {
	var err error
	conn := protocol.NewConn(netConn)

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

	ch := make(chan protocol.Result)

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
