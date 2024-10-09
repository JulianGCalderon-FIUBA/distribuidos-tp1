package main

import (
	"container/heap"
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"fmt"
	"sort"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP    string
	TopN        int
	PartitionId int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("TopN", "10")
	v.SetDefault("PartitionId", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("TopN", "TOP_N")
	_ = v.BindEnv("PartitionId", "PARTITION_ID")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	topN    int
	results utils.GameHeap
}

func (h handler) Aggregate(g middleware.Game) error {
	if h.results.Len() < h.topN {
		heap.Push(&h.results, middleware.GameStat{
			Name: g.Name,
			Stat: g.AveragePlaytimeForever,
		})
	} else if g.AveragePlaytimeForever > h.results.Peek().(middleware.GameStat).Stat {
		heap.Pop(&h.results)
		heap.Push(&h.results, middleware.GameStat{
			Name: g.Name,
			Stat: g.AveragePlaytimeForever,
		})
	}
	return nil
}

func (h handler) Conclude() ([]any, error) {
	sortedGames := make([]middleware.GameStat, 0, h.results.Len())
	for h.results.Len() > 0 {
		sortedGames = append(sortedGames, heap.Pop(&h.results).(middleware.GameStat))
	}

	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].Stat > sortedGames[j].Stat
	})

	for _, g := range sortedGames {
		log.Infof("Game %v: %v", g.Name, g.Stat)
	}

	return []any{sortedGames}, nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	qName := fmt.Sprintf("%v-%v", middleware.TopNHistoricAvgQueue, cfg.PartitionId)
	aggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    qName,
		Output:   middleware.TopNHistoricAvgJQueue,
	}

	h := handler{
		topN:    cfg.TopN,
		results: make(utils.GameHeap, cfg.TopN),
	}

	agg, err := aggregator.NewAggregator(aggCfg, h)
	utils.Expect(err, "Failed to create partitioner")

	err = agg.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
}