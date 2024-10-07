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

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP string
	N        int
	Input    string
}

type handler struct {
	N       int
	results map[uint64]protocol.Q4Results
}

func (h handler) Aggregate(r middleware.ReviewsPerGame) error {
	if r.Reviews > h.N {
		res := protocol.Q4Results{
			Name: r.Name,
			EOF:  false,
		}
		h.results[r.AppID] = res
	}
	return nil
}

func (h handler) Conclude() ([]any, error) {
	results := slices.Collect(maps.Values(h.results))
	r := make([]any, 0)
	eof := protocol.Q4Results{
		Name: "",
		EOF:  true,
	}

	for _, res := range results {
		r = append(r, &res)
	}
	r = append(r, &eof)
	log.Infof("sending: %v", r)
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
		results: make(map[uint64]protocol.Q4Results),
	}

	agg, err := aggregator.NewAggregator(aggCfg, h)
	utils.Expect(err, "Failed to create more than n reviews node")

	err = agg.Run(context.Background())
	utils.Expect(err, "Failed to run more than n reviews node")
}
