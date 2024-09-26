package main

type gateway struct {
	// aca pueden ir los campos en comun
	// entre data handler y connection handler
}

func newGateway() *gateway {
	return &gateway{}
}

func (g *gateway) start() {
	go g.startConnectionHandler()
	go g.startDataHandler()
}

func (g *gateway) startConnectionHandler() {
	// aca iniciaria el hilo del connection handler
	// lo dejo como funcion pero podria ser una estructura (como gateway)
	// y estar en su propio archivo (o carpeta)
}

func (g *gateway) startDataHandler() {
	// aca iniciaria el hilo del data handler
	// lo dejo como funcion pero podria ser una estructura (como gateway)
	// y estar en su propio archivo (o carpeta)
}
