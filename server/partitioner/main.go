package main

import (
	"distribuidos/tp1/server/middleware"
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
	Type             string
}

const GameType string = "game"
const ReviewType string = "review"

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
	if c.Type != GameType && c.Type != ReviewType {
		return c, errors.New("Type should not be game or review")
	}
	if c.OutputExchange == "" {
		c.OutputExchange = fmt.Sprintf("%v-x", c.InputQueue)
	}

	return c, err
}

type Game middleware.Game
type Review middleware.Review

func (g Game) PartitionId(partitionsNumber int) uint64 {
	return g.AppID % uint64(partitionsNumber)
}
func (r Review) PartitionId(partitionsNumber int) uint64 {
	return r.AppID % uint64(partitionsNumber)
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	if cfg.Type == ReviewType {
		p, err := newPartitioner[Game](cfg)
		utils.Expect(err, "Failed to create partitioner")

		err = p.run()
		utils.Expect(err, "Failed to run partitioner")
	} else {
		p, err := newPartitioner[Review](cfg)
		utils.Expect(err, "Failed to create partitioner")

		err = p.run()
		utils.Expect(err, "Failed to run partitioner")
	}
}
