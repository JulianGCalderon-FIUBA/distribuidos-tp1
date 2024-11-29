package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP string
	N        int
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

type handler struct {
	N int
}

func (h handler) Filter(g middleware.GameStat) []string {
	if g.Stat > uint64(h.N) {
		return []string{middleware.KeyQ4}
	}
	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q4Result{})

	h := handler{
		N: cfg.N,
	}

	filterCfg := middleware.FilterConfig{
		RabbitIP: cfg.RabbitIP,
		Exchange: middleware.ExchangeQ4,
		Queue:    middleware.GroupedQ4Filter,
		QueuesByKey: map[string][]string{
			middleware.KeyQ4: {
				middleware.ResultsQ4,
			},
		},
		
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	p, err := middleware.NewFilter(filterCfg, h.Filter)
	utils.Expect(err, "Failed to create filter")
	err = p.Run(ctx)
	utils.Expect(err, "Failed to run filter")
}
