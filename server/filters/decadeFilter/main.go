package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/utils"
	"strconv"
	"strings"

	"github.com/op/go-logging"
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

func (h handler) Filter(g middleware.Game) filter.RoutingKey {
	mask := strconv.Itoa(h.decade)[0:3]
	releaseYear := strconv.Itoa(int(g.ReleaseYear))

	if strings.Contains(releaseYear, mask) {
		return filter.RoutingKey(middleware.EmptyKey)
	}

	return filter.RoutingKey(middleware.EmptyKey)
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	filterCfg := filter.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    middleware.GamesQueue,
		Exchange: middleware.DecadeExchange,
		Output: map[filter.RoutingKey][]filter.QueueName{
			filter.RoutingKey(middleware.EmptyKey): {
				filter.QueueName(middleware.TopNHistoricAvgPQueue),
			},
		},
	}

	h := handler{
		decade: cfg.Decade,
	}
	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(context.Background())
	utils.Expect(err, "Failed to run filter")
}
