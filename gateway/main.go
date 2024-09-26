package main

import (
	"log"
)

type config struct {
	connectionEndpointPort int
	dataEndpointPort       int
}

func getConfig() (config, error) {
	// todo: read from file
	return config{
		connectionEndpointPort: 9001,
		dataEndpointPort:       9002,
	}, nil
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	gateway := newGateway(config)
	gateway.start()
}
