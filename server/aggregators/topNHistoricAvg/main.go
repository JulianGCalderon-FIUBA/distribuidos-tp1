package main

import (
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP        string
	TopN            int
	PartitionNumber int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", "10")
	v.SetDefault("PartitionNumber", "0")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("TopN", "TOP_N")
	_ = v.BindEnv("PartitionNumber", "PARTITION_NUMBER")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	a := NewAggregator(cfg)

	err = a.run()
	if err != nil {
		log.Fatalf("failed to get top %v historic average games: %v", cfg.TopN, err)
	}
}
