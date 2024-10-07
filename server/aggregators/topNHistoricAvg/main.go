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
	v.SetDefault("PartitionId", "0")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("TopN", "TOP_N")
	_ = v.BindEnv("PartitionId", "PARTITION_ID")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type GameHeap []middleware.AvgPlaytimeGame

func (g GameHeap) Len() int { return len(g) }
func (g GameHeap) Less(i, j int) bool {
	return g[i].AveragePlaytimeForever < g[j].AveragePlaytimeForever
}
func (g GameHeap) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
func (g *GameHeap) Push(x any) {
	*g = append(*g, x.(middleware.AvgPlaytimeGame))
}

func (g *GameHeap) Pop() any {
	old := *g
	n := len(old)
	x := old[n-1]
	*g = old[0 : n-1]
	return x
}

type handler struct {
	topN    int
	results GameHeap
}

func (h handler) Aggregate(g middleware.Game) error {
	if h.results.Len() < h.topN {
		heap.Push(&h.results, middleware.AvgPlaytimeGame{
			Name:                   g.Name,
			AveragePlaytimeForever: g.AveragePlaytimeForever,
		})
	} else if g.AveragePlaytimeForever > h.results[0].AveragePlaytimeForever {
		heap.Pop(&h.results)
		heap.Push(&h.results, middleware.AvgPlaytimeGame{
			Name:                   g.Name,
			AveragePlaytimeForever: g.AveragePlaytimeForever,
		})
	}
	return nil
}

func (h handler) Conclude() ([]any, error) {
	sortedGames := make([]middleware.AvgPlaytimeGame, 0, h.results.Len())
	for h.results.Len() > 0 {
		sortedGames = append(sortedGames, heap.Pop(&h.results).(middleware.AvgPlaytimeGame))
	}

	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].AveragePlaytimeForever > sortedGames[j].AveragePlaytimeForever
	})

	for _, g := range sortedGames {
		log.Infof("Game %v: %v", g.Name, g.AveragePlaytimeForever)
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
		results: make(GameHeap, cfg.TopN),
	}

	agg, err := aggregator.NewAggregator(aggCfg, h)
	utils.Expect(err, "Failed to create partitioner")

	err = agg.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
}
