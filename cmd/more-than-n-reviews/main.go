package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"maps"
	"os/signal"
	"slices"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP string
	N        int
	Address  string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", 5000)

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("N", "N_REVIEWS")
	_ = v.BindEnv("Address", "ADDRESS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	output    string
	N         int
	results   map[uint64]middleware.GameStat
	sequencer *utils.Sequencer
}

func (h *handler) handleBatch(ch *middleware.Channel, data []byte) error {

	batch, err := middleware.Deserialize[middleware.Batch[middleware.GameStat]](data)
	if err != nil {
		return err
	}

	h.sequencer.Mark(batch.BatchID, batch.EOF)

	for _, r := range batch.Data {
		if int(r.Stat) > h.N {
			h.results[r.AppID] = r
		}
	}

	if h.sequencer.EOF() {
		return h.conclude(ch)
	}

	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	results := slices.Collect(maps.Values(h.results))
	if len(results) == 0 {
		p := protocol.Q4Result{
			EOF: true,
		}

		return ch.SendAny(p, "", middleware.Results)
	}

	for i, res := range results {
		p := protocol.Q4Result{
			Games: []middleware.GameStat{res},
			EOF:   i == len(results)-1,
		}

		err := ch.SendAny(p, "", h.output)
		if err != nil {
			return err
		}
	}
	ch.Finish()
	return nil
}

func (h *handler) Free() error {
	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q4Result{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.GroupedQ4Filter
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: middleware.Results},
		},
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			return &handler{
				output:    middleware.Results,
				N:         cfg.N,
				results:   make(map[uint64]middleware.GameStat),
				sequencer: utils.NewSequencer(),
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[*handler]{
			middleware.GroupedQ4Filter: (*handler).handleBatch,
		},
		Address: cfg.Address,
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
