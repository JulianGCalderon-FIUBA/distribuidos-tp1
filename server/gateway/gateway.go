package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type gateway struct {
	config  config
	rabbit  *amqp.Connection
	mu      *sync.Mutex
	clients map[int]*protocol.Conn
}

func newGateway(config config) *gateway {
	protocol.Register()
	return &gateway{
		config:  config,
		clients: make(map[int]*protocol.Conn),
		mu:      &sync.Mutex{},
	}
}

func (g *gateway) start(ctx context.Context) error {
	addr := fmt.Sprintf("amqp://guest:guest@%v:5672/", g.config.RabbitIP)
	conn, err := amqp.Dial(addr)
	if err != nil {
		return err
	}
	g.rabbit = conn
	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		err := g.startRequestEndpoint(ctx)
		if err != nil {
			log.Errorf("Failed to run request endpoint: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := g.startDataEndpoint(ctx)
		if err != nil {
			log.Errorf("Failed to run data endpoint: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := g.startResultsEndpoint(ctx)
		if err != nil {
			log.Errorf("Failed to run results handler: %v", err)
		}
	}()
	wg.Wait()

	return nil
}
