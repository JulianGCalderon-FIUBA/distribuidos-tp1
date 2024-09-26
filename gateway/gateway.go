package main

type gateway struct {
	// aca pueden ir los campos en comun
	// entre data handler y connection handler
	// ej: clientes activos?
}

func newGateway() *gateway {
	return &gateway{}
}

func (g *gateway) start() {
	go g.startConnectionHandler()
	go g.startDataHandler()
}
