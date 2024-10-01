package main

import (
	"fmt"
	"sync"
)

type gateway struct {
	config        config
	activeClients int
	mu sync.Mutex
}

func newGateway(config config) *gateway {
	return &gateway{
		config:        config,
		activeClients: 0,
	}
}

func (g *gateway) start() {
	var wg sync.WaitGroup

	wg.Add(2)
	go g.startConnectionHandler()
	go g.startDataHandler()

	wg.Wait()
	fmt.Println("All goroutines have completed.")
}
