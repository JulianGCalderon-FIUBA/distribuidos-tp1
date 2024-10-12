package main

import (
	"context"
	"os/signal"
	"syscall"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

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
		log.Fatalf("Failed to read config: %v", err)
	}
	_ = cfg.DataEndpointPort

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	gateway := newGateway(cfg)
	err = gateway.start(ctx)
	if err != nil {
		log.Fatalf("Failed to run gateway: %v", err)
	}
}
