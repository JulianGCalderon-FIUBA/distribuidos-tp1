package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"math"
	"os/signal"
	"sort"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP   string
	Percentile int
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

type handler struct {
	input      string
	output     string
	sorted     []middleware.GameStat
	percentile float64
	sequencer  *utils.Sequencer
}

func (h *handler) handleBatch(ch *middleware.Channel, data []byte) error {

	batch, err := middleware.Deserialize[middleware.Batch[middleware.GameStat]](data)
	if err != nil {
		return err
	}

	h.sequencer.Mark(batch.BatchID, batch.EOF)

	for _, r := range batch.Data {
		i := sort.Search(len(h.sorted), func(i int) bool { return h.sorted[i].Stat >= r.Stat })
		h.sorted = append(h.sorted, middleware.GameStat{})
		copy(h.sorted[i+1:], h.sorted[i:])
		h.sorted[i] = r
	}

	if h.sequencer.EOF() {
		n := float64(len(h.sorted))
		index := max(0, int(math.Ceil(h.percentile/100.0*n))-1)
		results := h.sorted[index:]
		p := protocol.Q5Results{
			Percentile90: results,
		}

		err := ch.SendAny(p, "", h.output)
		if err != nil {
			return err
		}

		return ch.SendFinish("", h.input)
	}
	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q5Results{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.GroupedQ5Percentile

	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: middleware.Results},
		},
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[handler]{
		Builder: func(clientID int) handler {
			return handler{
				input:      qInput,
				output:     middleware.Results,
				sorted:     make([]middleware.GameStat, 0),
				percentile: float64(cfg.Percentile),
				sequencer:  utils.NewSequencer(),
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[handler]{
			middleware.GroupedQ5Percentile: (*handler).handleBatch,
		},
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
