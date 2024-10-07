package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"math"
	"sort"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

const PERCENTILE float64 = 90.0

type config struct {
	RabbitIP string
	Input    string
}

type handler struct {
	sorted []middleware.ReviewsPerGame
}

func (h *handler) Aggregate(r middleware.ReviewsPerGame) error {
	i := sort.Search(len(h.sorted), func(i int) bool { return h.sorted[i].Reviews >= r.Reviews })

	h.sorted = append(h.sorted, middleware.ReviewsPerGame{})
	copy(h.sorted[i+1:], h.sorted[i:])
	h.sorted[i] = r

	return nil
}

func (h *handler) Conclude() ([]any, error) {
	log.Infof("Total games: %v", len(h.sorted))
	n := float64(len(h.sorted))
	index := int(math.Ceil(PERCENTILE/100.0*n)) - 1
	results := h.sorted[index:]
	r := make([]string, 0)
	for _, res := range results {
		r = append(r, res.Name)
	}

	log.Infof("90 percentile value and index: %v %v", h.sorted[index], index)

	var p any = protocol.Q5Results{
		Percentile90: r,
	}

	return []any{&p}, nil
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Input", "INPUT")

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
		Input:    cfg.Input,
		Output:   middleware.ResultsQueue,
	}

	h := handler{
		sorted: make([]middleware.ReviewsPerGame, 0),
	}

	agg, err := aggregator.NewAggregator(aggCfg, &h)
	utils.Expect(err, "Failed to create 90 percentile node")

	err = agg.Run(context.Background())
	utils.Expect(err, "Failed to run 90 percentile node")
}
