package main

import (
	"container/heap"
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/joiner"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"sort"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	TopN       int
	Partitions int
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
	topN      int
	topNGames utils.GameHeap
}

func (h handler) Aggregate(partial []middleware.AvgPlaytimeGame) error {
	for _, g := range partial {
		if h.topNGames.Len() < h.topN {
			heap.Push(&h.topNGames, g)
		} else if g.AveragePlaytimeForever > h.topNGames.Peek().(middleware.AvgPlaytimeGame).AveragePlaytimeForever {
			heap.Pop(&h.topNGames)
			heap.Push(&h.topNGames, g)
		}
	}

	return nil
}

func (h handler) Conclude() (any, error) {
	log.Infof("Heap length: %v", h.topNGames.Len())
	sortedGames := make([]middleware.AvgPlaytimeGame, 0, h.topNGames.Len())
	for h.topNGames.Len() > 0 {
		sortedGames = append(sortedGames, heap.Pop(&h.topNGames).(middleware.AvgPlaytimeGame))
	}

	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].AveragePlaytimeForever > sortedGames[j].AveragePlaytimeForever
	})

	topNNames := make([]string, 0, h.topN)
	for _, g := range sortedGames {
		topNNames = append(topNNames, g.Name)
	}

	log.Infof("Q2 Results: %v", topNNames)

	return &protocol.Q2Results{TopN: topNNames}, nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q2Results{})

	joinCfg := joiner.Config{
		RabbitIP:         cfg.RabbitIP,
		Input:            middleware.TopNHistoricAvgJQueue,
		Output:           middleware.ResultsQueue,
		PartitionsNumber: cfg.Partitions,
	}

	h := handler{
		topN:      cfg.TopN,
		topNGames: make([]middleware.AvgPlaytimeGame, 0, cfg.TopN),
	}

	join, err := joiner.NewJoiner(joinCfg, h)
	utils.Expect(err, "Failed to create partitioner")

	err = join.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
}
