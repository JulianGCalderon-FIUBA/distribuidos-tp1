package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"fmt"
	"sync"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP         string
	PartitionsNumber int
	Input            string
	Output           string
}

type sharedHandler struct {
	lastBatchId     int
	eofReceived     int
	totalPartitions int
	m               *sync.Mutex
}

type handler struct {
	id int
	sh *sharedHandler
}

func (h handler) Aggregate(r middleware.Batch[middleware.ReviewsPerGame]) error {
	h.sh.m.Lock()
	defer h.sh.m.Unlock()

	h.sh.lastBatchId += 1
	b := middleware.Batch[middleware.ReviewsPerGame]{
		Data:     r.Data,
		ClientID: r.ClientID,
		BatchID:  h.sh.lastBatchId,
		EOF:      false,
	}

	// mandar b
	return nil
}

func (h handler) Conclude() ([]any, error) {
	h.sh.m.Lock()
	defer h.sh.m.Unlock()

	h.sh.eofReceived += 1
	b := middleware.Batch[middleware.ReviewsPerGame]{}

	if h.sh.eofReceived == h.sh.totalPartitions {
		b.EOF = true
	}

	// mandar b

	return nil, nil
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionsNumber", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionsNumber", "PARTITIONS")
	_ = v.BindEnv("Input", "INPUT")
	_ = v.BindEnv("Output", "OUTPUT")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	shared := sharedHandler{
		lastBatchId:     0,
		eofReceived:     0,
		totalPartitions: cfg.PartitionsNumber,
		m:               &sync.Mutex{},
	}

	wg := sync.WaitGroup{}
	wg.Add(cfg.PartitionsNumber)

	for i := range cfg.PartitionsNumber {
		h := handler{
			id: i + 1,
			sh: &shared,
		}
		qname := fmt.Sprintf("%v-%v", cfg.Input, i+1)
		hcfg := aggregator.Config{
			RabbitIP: cfg.RabbitIP,
			Input:    qname,
			Output:   cfg.Output,
		}
		go func() {
			defer wg.Done()
			agg, err := aggregator.NewAggregator(hcfg, h)
			utils.Expect(err, "Failed to create game aggregator")
			err = agg.Run(context.Background())
			utils.Expect(err, "Failed to run game aggregator")
		}()
	}
}
