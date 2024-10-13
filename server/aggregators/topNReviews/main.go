package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/middleware/aggregator"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"os/signal"
	"slices"
	"sort"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP    string
	PartitionID int
	N           int
}

type handler struct {
	output string
	sorted []middleware.GameStat
	N      int
}

func (h *handler) Aggregate(_ *middleware.Channel, batch middleware.Batch[middleware.GameStat]) error {
	for _, r := range batch.Data {
		i := sort.Search(len(h.sorted), func(i int) bool { return h.sorted[i].Stat >= r.Stat })

		g := middleware.GameStat{
			AppID: r.AppID,
			Name:  r.Name,
			Stat:  r.Stat,
		}

		h.sorted = append(h.sorted, middleware.GameStat{})
		copy(h.sorted[i+1:], h.sorted[i:])
		h.sorted[i] = g
	}

	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	index := max(0, len(h.sorted)-h.N)
	top := h.sorted[index:]
	slices.Reverse(top)

	return ch.Send(top, "", h.output)
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", "5")
	v.SetDefault("PartitionID", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("N", "N")
	_ = v.BindEnv("PartitionID", "PARTITION_ID")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q3Results{})

	input := middleware.Cat(middleware.GroupedQ3, cfg.PartitionID)
	aggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    input,
		Output:   middleware.PartialQ3,
	}

	h := handler{
		output: middleware.PartialQ3,
		sorted: make([]middleware.GameStat, 0),
		N:      cfg.N,
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	agg, err := aggregator.NewAggregator(aggCfg, &h)
	utils.Expect(err, "Failed to create top n reviews node")

	err = agg.Run(ctx)
	utils.Expect(err, "Failed to run top n reviews node")
}
