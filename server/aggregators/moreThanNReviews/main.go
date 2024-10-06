package main

import (
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
	N        int
	ID       int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", 5000)
	v.SetDefault("ID", 0)

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("N", "N_REVIEWS")
	_ = v.BindEnv("ID", "ID")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	joiner, err := newAggregator(config)
	if err != nil {
		log.Fatalf("Failed to create new aggregator: %v", err)
	}

	joiner.run()
}
