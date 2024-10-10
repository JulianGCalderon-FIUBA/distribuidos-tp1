package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"os/signal"
	"syscall"

	"distribuidos/tp1/utils"
	"encoding/gob"
	"math"
	"sort"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP   string
	Percentile int
}

type handler struct {
	sorted     []middleware.ReviewsPerGame
	percentile float64
}

func (h *handler) Aggregate(_ *middleware.Channel, batch middleware.Batch[middleware.ReviewsPerGame]) error {
	for _, r := range batch.Data {
		i := sort.Search(len(h.sorted), func(i int) bool { return h.sorted[i].Reviews >= r.Reviews })
		h.sorted = append(h.sorted, middleware.ReviewsPerGame{})
		copy(h.sorted[i+1:], h.sorted[i:])
		h.sorted[i] = r
	}

	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	n := float64(len(h.sorted))
	index := max(0, int(math.Ceil(h.percentile/100.0*n))-1)
	results := h.sorted[index:]
	r := make([]string, 0)
	for _, res := range results {
		r = append(r, res.Name)
	}

	p := protocol.Q5Results{
		Percentile90: r,
	}

	return ch.SendAny(p, "", middleware.ResultsQueue)
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Percentile", 90)

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Percentile", "PERCENTILE")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q5Results{})

	aggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    middleware.NinetyPercentileCalculator,
		Output:   middleware.ResultsQueue,
	}

	h := handler{
		sorted: make([]middleware.ReviewsPerGame, 0),
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	agg, err := aggregator.NewAggregator(aggCfg, &h)
	utils.Expect(err, "Failed to create 90 percentile node")

	err = agg.Run(ctx)
	utils.Expect(err, "Failed to run 90 percentile node")
}
