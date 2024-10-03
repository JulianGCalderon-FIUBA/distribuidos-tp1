package main

import (
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

func (g *gateway) start() {
	m, err := middleware.NewMiddleware(g.config.RabbitIP)
	if err != nil {
		log.Fatalf("Failed to initialize middleware: %v", err)
	}
	g.m = m

	var wg sync.WaitGroup
	wg.Add(2)
	go g.startConnectionHandler()
	go g.startDataHandler()
	wg.Wait()
}
