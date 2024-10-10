package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"maps"
	"os/signal"
	"slices"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP string
	N        int
}

type handler struct {
	N       int
	results map[uint64]middleware.ReviewsPerGame
}

func (h *handler) Aggregate(_ *middleware.Channel, batch middleware.Batch[middleware.ReviewsPerGame]) error {
	for _, r := range batch.Data {
		if int(r.Reviews) > h.N {
			h.results[r.AppID] = r
		}
	}
	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	results := slices.Collect(maps.Values(h.results))
	for i, res := range results {
		p := protocol.Q4Results{
			Name: res.Name,
			EOF:  i == len(results)-1,
		}

		err := ch.SendAny(p, "", middleware.ResultsQueue)
		if err != nil {
			return err
		}
	}
	return nil
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", 5000)

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("N", "N_REVIEWS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q4Results{})

	aggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    middleware.MoreThanNReviewsCalculator,
		Output:   middleware.ResultsQueue,
	}

	h := handler{
		N:       cfg.N,
		results: make(map[uint64]middleware.ReviewsPerGame),
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	agg, err := aggregator.NewAggregator(aggCfg, &h)
	utils.Expect(err, "Failed to create more than n reviews node")

	err = agg.Run(ctx)
	utils.Expect(err, "Failed to run more than n reviews node")
}
