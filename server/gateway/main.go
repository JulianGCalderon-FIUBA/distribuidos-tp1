package main

import (
	"log"
)

type config struct {
	connectionEndpointPort int
	dataEndpointPort       int
	rabbitIP               string
}

func getConfig() (config, error) {
	// todo: read from file
	return config{
		connectionEndpointPort: 9001,
		dataEndpointPort:       9002,
		rabbitIP:               "localhost",
	}, nil
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	gateway := newGateway(cfg)
	gateway.start()
}
