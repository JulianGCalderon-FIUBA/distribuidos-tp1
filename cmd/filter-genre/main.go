package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"slices"
	"syscall"

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
	v.SetDefault("Address", "genre-filter-1:7000")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("BatchSize", "BATCH_SIZE")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func Filter(g middleware.Game) []string {
	var rks []string
	if slices.Contains(g.Genres, middleware.IndieGenre) {
		rks = append(rks, middleware.IndieKey)
	}
	if slices.Contains(g.Genres, middleware.ActionGenre) {
		rks = append(rks, middleware.ActionKey)
	}
	return rks
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	filterCfg := middleware.FilterConfig{
		RabbitIP: cfg.RabbitIP,
		Queue:    middleware.GamesGenre,
		Exchange: middleware.ExchangeGenre,
		QueuesByKey: map[string][]string{
			middleware.IndieKey: {
				middleware.GamesDecade,
				middleware.GamesQ3,
			},
			middleware.ActionKey: {
				middleware.GamesQ4,
				middleware.GamesQ5,
			},
		},
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	p, err := middleware.NewFilter(filterCfg, Filter)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
