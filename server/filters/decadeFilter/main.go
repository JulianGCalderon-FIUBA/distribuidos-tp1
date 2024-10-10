package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/server/middleware/node"
	"distribuidos/tp1/utils"
	"fmt"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
	Decade   int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Decade", "2010")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Decade", "DECADE")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	decade int
}

func (h handler) Filter(g middleware.Game) []string {
	mask := strconv.Itoa(h.decade)[0:3]
	releaseYear := strconv.Itoa(int(g.ReleaseYear))

	if strings.Contains(releaseYear, mask) {
		return []string{fmt.Sprintf("%v-%v", middleware.DecadeKey, h.decade)}
	}

	return nil
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	key := fmt.Sprintf("%v-%v", middleware.DecadeKey, cfg.Decade)
	filterCfg := filter.Config{
		RabbitIP: cfg.RabbitIP,
		Queue:    middleware.GamesDecade,
		Exchange: node.ExchangeConfig{
			Name: middleware.DecadeExchange,
			Type: amqp.ExchangeDirect,
			QueuesByKey: map[string][]string{
				key: {middleware.TopNHistoricAvgPQueue},
			},
		},
	}

	h := handler{
		decade: cfg.Decade,
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
