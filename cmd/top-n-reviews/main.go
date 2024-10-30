package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
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

type handler struct {
	output    string
	sorted    []middleware.GameStat
	N         int
	sequencer *utils.Sequencer
}

func (h *handler) handleBatch(ch *middleware.Channel, data []byte) error {

	batch, err := middleware.Deserialize[middleware.Batch[middleware.GameStat]](data)
	if err != nil {
		return err
	}

	h.sequencer.Mark(batch.BatchID, batch.EOF)

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

	if h.sequencer.EOF() {
		index := max(0, len(h.sorted)-h.N)
		top := h.sorted[index:]
		slices.Reverse(top)

		err := ch.Send(top, "", h.output)
		if err != nil {
			return err
		}
		ch.Finish()
	}

	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q3Result{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.Cat(middleware.GroupedQ3, cfg.PartitionID)
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: middleware.PartialQ3},
		},
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[handler]{
		Builder: func(clientID int) handler {
			return handler{
				output:    middleware.PartialQ3,
				sorted:    make([]middleware.GameStat, 0),
				N:         cfg.N,
				sequencer: utils.NewSequencer(),
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[handler]{
			qInput: (*handler).handleBatch,
		},
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
