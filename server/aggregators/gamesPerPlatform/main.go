package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"fmt"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP    string
	PartitionID int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionID", "0")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionID", "PARTITION_ID")

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

func (h handler) Aggregate(g middleware.Game) error {
	if g.Windows {
		h.count[Windows] += 1
	}
	if g.Linux {
		h.count[Linux] += 1
	}
	if g.Mac {
		h.count[Mac] += 1
	}
	return nil
}

func (h handler) Conclude() (any, error) {
	for k, v := range h.count {
		log.Infof("Found %v games with %v support", v, string(k))
	}
	return h.count, nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	qName := fmt.Sprintf("%v-x-%v", middleware.GamesPerPlatformQueue, cfg.PartitionID)
	aggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    qName,
		Output:   middleware.GamesPerPlatformJoin,
	}

	h := handler{
		count: make(map[Platform]int),
	}

	agg, err := aggregator.NewAggregator(aggCfg, h)
	utils.Expect(err, "Failed to create partitioner")

	err = agg.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
}
