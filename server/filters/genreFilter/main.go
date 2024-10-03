package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
	"log"
	"github.com/spf13/viper"
	amqp "github.com/rabbitmq/amqp091-go"
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

	rabbitAddress := fmt.Sprintf("amqp://guest:guest@%v:5672/", cfg.RabbitIP)
	rabbitConn, err := amqp.Dial(rabbitAddress)
	if err != nil {
		log.Fatalf("failed to connect to rabbit: %v", err)
	}

	m := middleware.NewMiddleware(rabbitConn)
	m.Init()

	err = filterGames(m, cfg); if err != nil {
		log.Fatalf("failed to filter games: %v", err)
	}
}