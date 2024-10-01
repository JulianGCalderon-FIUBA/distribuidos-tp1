package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type gateway struct {
	config config
	m      *middleware.Middleware
	// aca pueden ir los campos en comun
	// entre data handler y connection handler
	// ej: clientes activos?
}

func newGateway(config config) *gateway {
	return &gateway{
		config: config,
	}
}

func (g *gateway) start() {
	rabbitAddress := fmt.Sprintf("amqp://guest:guest@%v:5672/", g.config.RabbitIP)
	rabbitConn, err := amqp.Dial(rabbitAddress)
	if err != nil {
		log.Fatalf("failed to connect to rabbit: %v", err)
	}

	m := middleware.NewMiddleware(rabbitConn)
	g.m = m

	var wg sync.WaitGroup

	wg.Add(2)
	go g.startConnectionHandler()
	go g.startDataHandler()

	wg.Wait()
	fmt.Println("All goroutines have completed.")
}
