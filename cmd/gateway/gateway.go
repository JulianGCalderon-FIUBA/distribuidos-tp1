package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"errors"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type gateway struct {
	config   config
	rabbit   *amqp.Connection
	rabbitCh *amqp.Channel
	mu       *sync.Mutex
	clients  map[int]chan protocol.Results
}

func newGateway(config config) *gateway {
	protocol.Register()
	return &gateway{
		config:  config,
		clients: make(map[int]chan protocol.Results),
		mu:      &sync.Mutex{},
	}
}

func (g *gateway) start(ctx context.Context) error {
	conn, ch, err := middleware.Dial(g.config.RabbitIP)
	if err != nil {
		return err
	}
	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()
	g.rabbit = conn
	g.rabbitCh = ch

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
