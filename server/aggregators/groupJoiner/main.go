package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/middleware/aggregator"
	"distribuidos/tp1/utils"
	"os/signal"
	"sync"
	"syscall"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	Partitions int
	Input      string
	Output     string
}

type sharedHandler struct {
	lastBatchId     int
	eofReceived     int
	totalPartitions int
	m               *sync.Mutex
}

type handler struct {
	id     int
	sh     *sharedHandler
	output string
}

func (h handler) Aggregate(ch *middleware.Channel, batch middleware.Batch[middleware.ReviewsPerGame]) error {
	h.sh.m.Lock()
	defer h.sh.m.Unlock()

	h.sh.lastBatchId += 1
	b := middleware.Batch[middleware.ReviewsPerGame]{
		Data:     batch.Data,
		ClientID: batch.ClientID,
		BatchID:  h.sh.lastBatchId,
		EOF:      false,
	}

	return ch.Send(b, "", h.output)
}

func (h handler) Conclude(ch *middleware.Channel) error {
	h.sh.m.Lock()
	defer h.sh.m.Unlock()

	h.sh.eofReceived += 1
	b := middleware.Batch[middleware.ReviewsPerGame]{}

	if h.sh.eofReceived == h.sh.totalPartitions {
		b.EOF = true
		return ch.Send(b, "", h.output)
	}

	return nil
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Partitions", "PARTITIONS")
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
		totalPartitions: cfg.Partitions,
		m:               &sync.Mutex{},
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(cfg.Partitions)

	for i := 1; i <= cfg.Partitions; i++ {
		h := handler{
			id:     i,
			sh:     &shared,
			output: cfg.Output,
		}
		input := middleware.Cat(cfg.Input, i)
		hcfg := aggregator.Config{
			RabbitIP: cfg.RabbitIP,
			Input:    input,
			Output:   cfg.Output,
		}
		agg, err := aggregator.NewAggregator(hcfg, h)
		utils.Expect(err, "Failed to create game aggregator")
		go func() {
			defer wg.Done()
			log.Infof("Starting handler %v", i)
			err = agg.Run(ctx)
			utils.Expect(err, "Failed to run game aggregator")
		}()
	}

	wg.Wait()
}
