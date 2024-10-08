package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"slices"
	"sort"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP string
	Input    string
	Output   string
	N        int
}

type handler struct {
	sorted []middleware.GameStat
	N      int
}

func (h *handler) Aggregate(r middleware.ReviewsPerGame) error {
	i := sort.Search(len(h.sorted), func(i int) bool { return h.sorted[i].Stat >= r.Reviews })

	g := middleware.GameStat{
		AppID: r.AppID,
		Name:  r.Name,
		Stat:  r.Reviews,
	}

	h.sorted = append(h.sorted, middleware.GameStat{})
	copy(h.sorted[i+1:], h.sorted[i:])
	h.sorted[i] = g

	return nil
}

func (h *handler) Conclude() ([]any, error) {
	index := max(0, len(h.sorted)-h.N)
	top := h.sorted[index:]
	slices.Reverse(top)

	return []any{top}, nil
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", "5")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Input", "INPUT")
	_ = v.BindEnv("Output", "OUTPUT")
	_ = v.BindEnv("N", "N")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q3Results{})

	aggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    cfg.Input,
		Output:   cfg.Output,
	}

	h := handler{
		sorted: make([]middleware.GameStat, 0),
		N:      cfg.N,
	}

	agg, err := aggregator.NewAggregator(aggCfg, &h)
	utils.Expect(err, "Failed to create top n reviews node")

	err = agg.Run(context.Background())
	utils.Expect(err, "Failed to run top n reviews node")
}
