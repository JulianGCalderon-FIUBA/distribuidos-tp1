package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/server/middleware/node"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	"github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
}

type handler struct{}

func (h handler) Filter(r middleware.Review) []string {
	if r.Score == middleware.PositiveScore {
		return []string{middleware.PositiveReviewKey}
	} else {
		return []string{middleware.NegativeReviewKey}
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
		Queue:    middleware.ReviewsScore,
		Exchange: node.ExchangeConfig{
			Name: middleware.ReviewsScoreFilterExchange,
			Type: amqp091.ExchangeDirect,
			QueuesByKey: map[string][]string{
				middleware.PositiveReviewKey: {
					middleware.TopNAmountReviewsQueue,
				},
				middleware.NegativeReviewKey: {
					middleware.NinetyPercentileReviewsQueue,
					middleware.LanguageReviewsFilterQueue,
				},
			},
		},
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	h := handler{}
	p, err := filter.NewFilter(filterCfg, h)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
