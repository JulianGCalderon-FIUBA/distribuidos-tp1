package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/server/middleware/node"
	"distribuidos/tp1/utils"
	"os/signal"
	"slices"
	"syscall"

	"github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
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

func (h handler) Filter(g middleware.Game) []string {
	var rks []string
	if slices.Contains(g.Genres, middleware.IndieGenre) {
		rks = append(rks, middleware.IndieGameKey)
	}
	if slices.Contains(g.Genres, middleware.ActionGenre) {
		rks = append(rks, middleware.ActionGameKey)
	}
	return rks
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	filterCfg := filter.Config{
		RabbitIP: cfg.RabbitIP,
		Queue:    middleware.GamesQueue,
		Exchange: node.ExchangeConfig{
			Name: middleware.GenresExchange,
			Type: amqp.ExchangeDirect,
			QueuesByKey: map[string][]string{
				middleware.IndieGameKey: {
					middleware.DecadeQueue,
					middleware.TopNAmountReviewsGamesQueue,
				},
				middleware.ActionGameKey: {
					middleware.MoreThanNReviewsGamesQueue,
					middleware.NinetyPercentileGamesQueue,
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
