package main

import (
	"log"

	"github.com/spf13/viper"
)

type config struct {
	ConnectionEndpointAddress string
	DataEndpointAddress       string
	PackageSize               int
}

const KB int = 1 << 10

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("ConnectionEndpointAddress", "127.0.0.1:9001")
	v.SetDefault("DataEndpointAddress", "127.0.0.1:9002")
	v.SetDefault("BatchSize", 8*KB)

	_ = v.BindEnv("ConnectionEndpointAddress", "GATEWAY_CONN_ADDR")
	_ = v.BindEnv("DataEndpointAddress", "GATEWAY_DATA_ADDR")
	_ = v.BindEnv("BatchSize", "BATCH_SIZE")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	client := newClient(config)
	client.start()
}
