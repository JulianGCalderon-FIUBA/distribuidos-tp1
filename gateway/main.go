package main

import (
	"log"
)

type config struct {
	// aca pueden ir los parametros
	// de configuracion inicial
}

func getConfig() (config, error) {
	// aca ser deberia al config de algun archivo
	// lo dejo vacio como placeholder
	return config{}, nil
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	_ = config

	gateway := newGateway()
	gateway.start()
}
