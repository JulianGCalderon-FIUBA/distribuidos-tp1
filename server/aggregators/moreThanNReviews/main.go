package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"maps"
	"slices"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP string
	N        int
	Input    string
}

type handler struct {
	N       int
	results map[uint64]middleware.ReviewsPerGame
}

func (h handler) Aggregate(r middleware.ReviewsPerGame) error {
	if r.Reviews > h.N {
		h.results[r.AppID] = r
	}
	return nil
}

func (h handler) Conclude() ([]any, error) {
	results := slices.Collect(maps.Values(h.results))
	r := make([]any, 0)
	for i, res := range results {
		var p any = protocol.Q4Results{
			Name: res.Name,
			EOF:  i == len(results)-1,
		}
		r = append(r, &p)
	}
	return r, nil
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", 5000)

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("N", "N_REVIEWS")
	_ = v.BindEnv("Input", "INPUT")

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
		Input:    cfg.Input,
		Output:   middleware.ResultsQueue,
	}

	h := handler{
		N:       cfg.N,
		results: make(map[uint64]middleware.ReviewsPerGame),
	}

	agg, err := aggregator.NewAggregator(aggCfg, h)
	utils.Expect(err, "Failed to create more than n reviews node")

	err = agg.Run(context.Background())
	utils.Expect(err, "Failed to run more than n reviews node")
}