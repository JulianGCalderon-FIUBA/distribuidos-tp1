package main

import (
	"fmt"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type gateway struct {
	config        config
	rabbitConn    *amqp.Connection
	activeClients int
	mu            sync.Mutex
}

func newGateway(config config) *gateway {
	return &gateway{
		config:        config,
		activeClients: 0,
	}
}

func (g *gateway) start() {
	rabbitAddress := fmt.Sprintf("amqp://guest:guest@%v:5672/", g.config.RabbitIP)
	rabbitConn, err := amqp.Dial(rabbitAddress)
	if err != nil {
		log.Fatalf("failed to connect to rabbit: %v", err)
	}
	g.rabbitConn = rabbitConn

	var wg sync.WaitGroup

	wg.Add(2)
	go g.startConnectionHandler()
	go g.startDataHandler()

	wg.Wait()
	fmt.Println("All goroutines have completed.")
}
