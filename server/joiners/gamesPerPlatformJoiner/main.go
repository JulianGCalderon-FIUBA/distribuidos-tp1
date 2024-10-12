package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/middleware/joiner"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"os/signal"
	"syscall"

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

func (h handler) Aggregate(_ *middleware.Channel, c map[Platform]int) error {
	for k, v := range c {
		h.count[k] += v
	}

	return nil
}

func (h handler) Conclude(ch *middleware.Channel) error {
	for k, v := range h.count {
		log.Infof("Found %v games with %v support", v, string(k))
	}

	result := protocol.Q1Results{
		Windows: h.count[Windows],
		Linux:   h.count[Linux],
		Mac:     h.count[Mac],
	}

	return ch.SendAny(result, "", middleware.Results)
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q1Results{})

	joinCfg := joiner.Config{
		RabbitIP:   cfg.RabbitIP,
		Input:      middleware.PartialQ1,
		Output:     middleware.Results,
		Partitions: cfg.Partitions,
	}

	h := handler{
		count: make(map[Platform]int),
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	join, err := joiner.NewJoiner(joinCfg, h)
	utils.Expect(err, "Failed to create partitioner")

	err = join.Run(ctx)
	utils.Expect(err, "Failed to run partitioner")
}
