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
	languageFilter, err := newLanguageFilter(cfg)
	if err != nil {
		log.Fatalf("Failed to create new language filter: %v", err)
	}
	err = languageFilter.run()
	if err != nil {
		log.Fatalf("Failed to run language filter: %v", err)
	}
}
