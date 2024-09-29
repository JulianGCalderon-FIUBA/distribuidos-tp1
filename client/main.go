package main

import (
	"log"
)

type config struct {
	connectionEndpointAddress string
	dataEndpointAddress       string
	packageSize               int
}

func getConfig() (config, error) {
	// todo: read from file
	return config{
		connectionEndpointAddress: "127.0.0.1:9001",
		dataEndpointAddress:       "127.0.0.1:9002",
		packageSize:               100,
	}, nil
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	client := newClient(config)
	client.start()
}
