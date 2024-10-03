package main

import (
	"distribuidos/tp1/server/middleware"
	"log"
	"github.com/spf13/viper"
)

type config struct {
	RabbitIP               string
	BatchSize              int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("BatchSize", "100")

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

	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}
	err = m.Init(); if err != nil {
		log.Fatalf("failed to initialize middleware: %v", err)
	}

	err = filterGames(m, cfg); if err != nil {
		log.Fatalf("failed to filter games: %v", err)
	}
}