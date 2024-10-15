package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/middleware/aggregator"
	"distribuidos/tp1/protocol"
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
	sorted     []middleware.GameStat
	percentile float64
}

func (h *handler) Aggregate(_ *middleware.Channel, batch middleware.Batch[middleware.GameStat]) error {
	for _, r := range batch.Data {
		i := sort.Search(len(h.sorted), func(i int) bool { return h.sorted[i].Stat >= r.Stat })
		h.sorted = append(h.sorted, middleware.GameStat{})
		copy(h.sorted[i+1:], h.sorted[i:])
		h.sorted[i] = r
	}

	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	n := float64(len(h.sorted))
	index := max(0, int(math.Ceil(h.percentile/100.0*n))-1)
	results := h.sorted[index:]

	p := protocol.Q5Results{
		Percentile90: results,
	}

	return ch.SendAny(p, "", middleware.Results)
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
		Input:    middleware.GroupedQ5Percentile,
		Output:   middleware.Results,
	}

	h := handler{
		sorted:     make([]middleware.GameStat, 0),
		percentile: float64(cfg.Percentile),
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	agg, err := aggregator.NewAggregator(aggCfg, &h)
	utils.Expect(err, "Failed to create 90 percentile node")

	err = agg.Run(ctx)
	utils.Expect(err, "Failed to run 90 percentile node")
}
