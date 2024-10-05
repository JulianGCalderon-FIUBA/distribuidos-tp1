package main

import (
	"distribuidos/tp1/utils"
	"errors"
	"fmt"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP         string
	InputQueue       string
	OutputExchange   string
	PartitionsNumber int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionsNumber", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionsNumber", "PARTITIONS_NUMBER")
	_ = v.BindEnv("InputQueue", "INPUT_QUEUE")
	_ = v.BindEnv("Type", "TYPE")

	var c config
	err := v.Unmarshal(&c)

	if c.InputQueue == "" {
		return c, errors.New("InputQueue should not be empty")
	}
	if c.OutputExchange == "" {
		c.OutputExchange = fmt.Sprintf("%v-x", c.InputQueue)
	}

	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	p, err := newPartitioner(cfg)
	utils.Expect(err, "Failed to create partitioner")

	err = p.run()
	utils.Expect(err, "Failed to run partitioner")
}
