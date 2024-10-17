package main

import (
	"container/heap"
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"os/signal"
	"sort"
	"syscall"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	TopN       int
	Partitions int
	Input      string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("TopN", "10")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("TopN", "TOP_N")
	_ = v.BindEnv("Partitions", "PARTITIONS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	output     string
	topN       int
	topNGames  middleware.GameHeap
	partitions int
}

func (h *handler) handlePartialResult(ch *middleware.Channel, data []byte) error {
	partial, err := middleware.Deserialize[[]middleware.GameStat](data)
	h.partitions--

	if err != nil {
		return err
	}

	for _, g := range partial {
		if h.topNGames.Len() < h.topN {
			heap.Push(&h.topNGames, g)
		} else if g.Stat > h.topNGames.Peek().(middleware.GameStat).Stat {
			heap.Pop(&h.topNGames)
			heap.Push(&h.topNGames, g)
		}
	}

	if h.partitions == 0 {
		log.Infof("Received all partial results")
		return h.conclude(ch)
	}
	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	sortedGames := make([]middleware.GameStat, 0, h.topNGames.Len())
	for h.topNGames.Len() > 0 {
		sortedGames = append(sortedGames, heap.Pop(&h.topNGames).(middleware.GameStat))
	}

	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].Stat > sortedGames[j].Stat
	})

	result := protocol.Q3Results{TopN: sortedGames}
	return ch.SendAny(result, "", h.output)
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q3Results{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.PartialQ3
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: middleware.Results},
		},
	}.Declare(ch)

	if err != nil {
		utils.Expect(err, "Failed to declare topology")
	}

	nConfig := middleware.Config[handler]{
		Builder: func(clientID int) handler {
			return handler{
				output:     middleware.Results,
				topN:       cfg.TopN,
				topNGames:  make([]middleware.GameStat, 0, cfg.TopN),
				partitions: cfg.Partitions,
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[handler]{
			qInput: (*handler).handlePartialResult,
		},
	}

	node, err := middleware.NewNode(nConfig, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
