package main

import (
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	reviewFilter, err := newReviewFilter(cfg)
	if err != nil {
		log.Fatalf("Failed to create new review filter: %v", err)
	}
	err = reviewFilter.run()
	if err != nil {
		log.Fatalf("Failed to run review filter: %v", err)
	}
}
