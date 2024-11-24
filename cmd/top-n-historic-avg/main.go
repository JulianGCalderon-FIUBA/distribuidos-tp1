package main

import (
	"container/heap"
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"sort"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP    string
	TopN        int
	PartitionId int
	Input       string
	Output      string
	Address     string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("TopN", "10")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("TopN", "TOP_N")
	_ = v.BindEnv("PartitionId", "PARTITION_ID")
	_ = v.BindEnv("Input", "INPUT")
	_ = v.BindEnv("Address", "ADDRESS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	output    string
	topN      int
	results   middleware.GameHeap
	sequencer *utils.Sequencer
}

func (h *handler) handleBatch(ch *middleware.Channel, data []byte) error {
	batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](data)
	if err != nil {
		return err
	}

	h.sequencer.Mark(batch.BatchID, batch.EOF)

	for _, g := range batch.Data {
		if h.results.Len() < h.topN {
			heap.Push(&h.results, middleware.GameStat{
				AppID: g.AppID,
				Name:  g.Name,
				Stat:  g.AveragePlaytimeForever,
			})
		} else if g.AveragePlaytimeForever > h.results.Peek().(middleware.GameStat).Stat {
			heap.Pop(&h.results)
			heap.Push(&h.results, middleware.GameStat{
				AppID: g.AppID,
				Name:  g.Name,
				Stat:  g.AveragePlaytimeForever,
			})
		}
	}

	if h.sequencer.EOF() {
		err := h.conclude(ch)
		if err != nil {
			return err
		}

		ch.Finish()
	}
	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	sortedGames := make([]middleware.GameStat, 0, h.results.Len())
	for h.results.Len() > 0 {
		sortedGames = append(sortedGames, heap.Pop(&h.results).(middleware.GameStat))
	}

	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].Stat > sortedGames[j].Stat
	})

	err := ch.Send(sortedGames, "", h.output)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) Free() error {
	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.Cat(cfg.Input, "x", cfg.PartitionId)

	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: middleware.PartialQ2},
		},
	}.Declare(ch)

	if err != nil {
		utils.Expect(err, "Failed to declare topology")
	}

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			return &handler{
				output:    middleware.PartialQ2,
				topN:      cfg.TopN,
				results:   make(middleware.GameHeap, cfg.TopN),
				sequencer: utils.NewSequencer(),
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[*handler]{
			qInput: (*handler).handleBatch,
		},
		Address: cfg.Address,
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
