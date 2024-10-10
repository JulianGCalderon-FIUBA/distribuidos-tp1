package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"sync"
)

type gateway struct {
	config        config
	m             *middleware.Middleware
	activeClients int
	mu            sync.Mutex
}

func newGateway(config config) *gateway {
	protocol.Register()
	return &gateway{
		config:        config,
		activeClients: 0,
	}
}

func (g *gateway) start(ctx context.Context) {
	m, err := middleware.NewMiddleware(g.config.RabbitIP)
	if err != nil {
		log.Fatalf("Failed to initialize middleware: %v", err)
	}
	g.m = m

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := g.startConnectionHandler(ctx)
		if err != nil {
			log.Errorf("Failed to start connection handler: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		err := g.startDataHandler(ctx)
		if err != nil {
			log.Errorf("Failed to start data handler: %v", err)
		}
	}()
	wg.Wait()
}
