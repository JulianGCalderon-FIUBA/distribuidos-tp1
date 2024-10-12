package main

import (
	"container/heap"
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/middleware/joiner"
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
	_ = v.BindEnv("Input", "INPUT")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	topN      int
	topNGames utils.GameHeap
}

func (h *handler) Aggregate(_ *middleware.Channel, partial []middleware.GameStat) error {
	for _, g := range partial {
		if h.topNGames.Len() < h.topN {
			heap.Push(&h.topNGames, g)
		} else if g.Stat > h.topNGames.Peek().(middleware.GameStat).Stat {
			heap.Pop(&h.topNGames)
			heap.Push(&h.topNGames, g)
		}
	}

	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	sortedGames := make([]middleware.GameStat, 0, h.topNGames.Len())
	for h.topNGames.Len() > 0 {
		sortedGames = append(sortedGames, heap.Pop(&h.topNGames).(middleware.GameStat))
	}

	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].Stat > sortedGames[j].Stat
	})

	topNNames := make([]string, 0, h.topN)
	for _, g := range sortedGames {
		topNNames = append(topNNames, g.Name)
	}

	log.Infof("Top N games in historic average playtime: %v", topNNames)

	result := protocol.Q2Results{TopN: topNNames}
	return ch.SendAny(result, "", middleware.Results)
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q2Results{})

	joinCfg := joiner.Config{
		RabbitIP:   cfg.RabbitIP,
		Input:      cfg.Input,
		Output:     middleware.Results,
		Partitions: cfg.Partitions,
	}

	h := handler{
		topN:      cfg.TopN,
		topNGames: make([]middleware.GameStat, 0, cfg.TopN),
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	join, err := joiner.NewJoiner(joinCfg, &h)
	utils.Expect(err, "Failed to create partitioner")

	err = join.Run(ctx)
	utils.Expect(err, "Failed to run partitioner")
}
