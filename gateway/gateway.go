package main

type gateway struct {
	config config
	// aca pueden ir los campos en comun
	// entre data handler y connection handler
	// ej: clientes activos?
}

func NewGateway(config config) *gateway {
	return &gateway{
		config: config,
	}
}

func (g *gateway) Start() {
	go g.StartConnectionHandler()
	go g.startDataHandler()
}
