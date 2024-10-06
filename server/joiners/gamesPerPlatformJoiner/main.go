package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/joiner"
	"distribuidos/tp1/utils"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	Partitions int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Partitions", "PARTITIONS")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

type Platform string

const (
	Mac     Platform = "mac"
	Linux   Platform = "linux"
	Windows Platform = "windows"
)

type handler struct {
	count map[Platform]int
}

func (h handler) Aggregate(c map[Platform]int) error {
	for k, v := range c {
		c[k] += v
	}

	return nil
}

func (h handler) Conclude() (any, error) {
	for k, v := range h.count {
		log.Infof("Found %v games with $v support", v, string(k))
	}
	return h.count, nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	joinCfg := joiner.Config{
		RabbitIP:         cfg.RabbitIP,
		Input:            middleware.GamesPerPlatformJoin,
		Output:           "",
		PartitionsNumber: cfg.Partitions,
	}

	h := handler{
		count: make(map[Platform]int),
	}

	join, err := joiner.NewJoiner(joinCfg, h)
	utils.Expect(err, "Failed to create partitioner")

	err = join.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
}
