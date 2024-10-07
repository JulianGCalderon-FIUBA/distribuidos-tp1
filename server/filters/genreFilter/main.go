package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/utils"
	"slices"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP  string
	BatchSize int
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

type handler struct{}

func (h handler) Filter(g middleware.Game) filter.RoutingKey {
	if slices.Contains(g.Genres, middleware.IndieGenre) {
		return filter.RoutingKey(middleware.IndieGameKeys)
	}

	if slices.Contains(g.Genres, middleware.ActionGenre) {
		return filter.RoutingKey(middleware.ActionGameKeys)
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
		Exchange: middleware.GenresExchange,
		Output: map[filter.RoutingKey][]filter.QueueName{
			filter.RoutingKey(middleware.IndieGameKeys): {
				filter.QueueName(middleware.DecadeQueue),
				filter.QueueName(middleware.TopNAmountReviewsGamesQueue),
			},
			filter.RoutingKey(middleware.ActionGameKeys): {
				filter.QueueName(middleware.MoreThanNReviewsGamesQueue),
				filter.QueueName(middleware.NinetyPercentileGamesQueue),
			},
		},
	}

	h := handler{}
	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(context.Background())
	utils.Expect(err, "Failed to run filter")
}
