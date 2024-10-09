package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/utils"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
}

type handler struct{}

func (h handler) Filter(r middleware.Review) []filter.RoutingKey {
	if r.Score == middleware.PositiveScore {
		return []filter.RoutingKey{filter.RoutingKey(middleware.PositiveReviewKey)}
	} else {
		return []filter.RoutingKey{filter.RoutingKey(middleware.NegativeReviewKey)}
	}
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	filterCfg := filter.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    middleware.ReviewsQueue,
		Exchange: middleware.ReviewsScoreFilterExchange,
		Output: map[filter.RoutingKey][]filter.QueueName{
			filter.RoutingKey(middleware.PositiveReviewKey): {
				filter.QueueName(middleware.TopNAmountReviewsQueue),
			},
			filter.RoutingKey(middleware.NegativeReviewKey): {
				filter.QueueName(middleware.NinetyPercentileReviewsQueue),
				filter.QueueName(middleware.LanguageReviewsFilterQueue),
			},
		},
	}

	h := handler{}
	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(context.Background())
	utils.Expect(err, "Failed to run filter")
}
