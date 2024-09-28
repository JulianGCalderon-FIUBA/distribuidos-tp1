package main

import (
	"fmt"
	"sync"
)

type gateway struct {
	config config
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
	var wg sync.WaitGroup

	wg.Add(2)
	go g.startConnectionHandler()
	go g.startDataHandler()

	wg.Wait()
	fmt.Println("All goroutines have completed.")
}
