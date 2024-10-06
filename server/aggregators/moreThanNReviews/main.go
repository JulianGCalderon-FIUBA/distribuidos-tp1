package main

import (
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
	N        int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", 5000)

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("N", "N_REVIEWS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	_, err := getConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
}
