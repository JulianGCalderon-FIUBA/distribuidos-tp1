package main

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
	go g.startConnectionHandler()
	go g.startDataHandler()
}
