package main

import (
	"log"

	"github.com/spf13/viper"
)

type config struct {
	ConnectionEndpointPort int
	DataEndpointPort       int
	RabbitIP               string
	BatchSize              int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("ConnectionEndpointPort", "9001")
	v.SetDefault("DataEndpointPort", "9002")
	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("BatchSize", "100")

	_ = v.BindEnv("ConnectionEndpointPort", "CONN_PORT")
	_ = v.BindEnv("DataEndpointPort", "DATA_PORT")
	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("BatchSize", "BATCH_SIZE")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	_ = cfg.DataEndpointPort

	gateway := newGateway(cfg)
	gateway.start()
}
