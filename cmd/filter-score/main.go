package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
}

func Filter(r middleware.Review) []string {
	if r.Score == middleware.PositiveScore {
		return []string{middleware.PositiveKey}
	} else {
		return []string{middleware.NegativeKey}
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

	filterCfg := middleware.FilterConfig{
		RabbitIP: cfg.RabbitIP,
		Queue:    middleware.ReviewsScore,
		Exchange: middleware.ExchangeScore,
		QueuesByKey: map[string][]string{
			middleware.PositiveKey: {
				middleware.ReviewsQ3,
			},
			middleware.NegativeKey: {
				middleware.ReviewsQ5,
				middleware.ReviewsLanguage,
			},
		},
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	p, err := middleware.NewFilter(filterCfg, Filter)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
