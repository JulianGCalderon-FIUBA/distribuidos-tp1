package main

import (
	"container/heap"
	"distribuidos/tp1/server/middleware"
	"fmt"
	"sort"
)

type Aggregator struct {
	config         config
	m              *middleware.Middleware
	receivingQueue string
	results        GameHeap
}

type Batch middleware.Batch[middleware.Game]
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

func NewAggregator(config config) *Aggregator {
	m, err := middleware.NewMiddleware(config.RabbitIP)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}

	if err = m.InitTopNHistoricAvg(config.PartitionId); err != nil {
		log.Fatalf("failed to initialize middleware: %v", err)
	}
	return &Aggregator{
		config:         config,
		m:              m,
		receivingQueue: fmt.Sprintf("%v-%v", middleware.TopNHistoricAvgQueue, config.PartitionId),
		results:        make([]middleware.AvgPlaytimeGame, config.TopN),
	}
}

func (a *Aggregator) run() error {
	log.Infof("Top %v historic avg started", a.config.TopN)

	err := a.receive()
	if err != nil {
		log.Errorf("Failed to receive batches: %v", err)
	}

	return nil
}

func (a *Aggregator) receive() error {
	deliveryCh, err := a.m.ReceiveFromQueue(a.receivingQueue)
	for d := range deliveryCh {
		if err != nil {
			_ = d.Nack(false, false)
			return err
		}

		batch, err := middleware.Deserialize[Batch](d.Body)
		if err != nil {
			_ = d.Nack(false, false)
			return err
		}

		for _, game := range batch.Data {
			if a.results.Len() < a.config.TopN {
				heap.Push(&a.results, middleware.AvgPlaytimeGame{
					Name:                   game.Name,
					AveragePlaytimeForever: game.AveragePlaytimeForever,
				})
			} else if game.AveragePlaytimeForever > a.results[0].AveragePlaytimeForever {
				heap.Pop(&a.results)
				heap.Push(&a.results, middleware.AvgPlaytimeGame{
					Name:                   game.Name,
					AveragePlaytimeForever: game.AveragePlaytimeForever,
				})
			}
		}

		if batch.EOF {
			log.Infof("Finished receiving batches for client %v", batch.ClientID)

			sorted := a.sortResult()

			for _, game := range sorted {
				log.Infof("%v, %v", game.Name, game.AveragePlaytimeForever)
			}

			// topNGames := Batch{
			// 	BatchID:  batch.BatchID,
			// 	ClientID: batch.ClientID,
			// 	EOF:      true,
			// 	Data:     sorted,
			// }

			// err = a.m.Send(topNGames, "", middleware.TopNHistoricAvgJQueue)

			// if err != nil {
			// 	log.Errorf("Failed to send top %v games: %v", a.config.TopN, err)
			// 	_ = d.Nack(false, false)
			// 	continue
			// }
		}
		_ = d.Ack(false)
	}

	return nil
}

func (a *Aggregator) sortResult() []middleware.AvgPlaytimeGame {
	sortedGames := make([]middleware.AvgPlaytimeGame, 0, a.results.Len())
	for a.results.Len() > 0 {
		sortedGames = append(sortedGames, heap.Pop(&a.results).(middleware.AvgPlaytimeGame))
	}

	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].AveragePlaytimeForever > sortedGames[j].AveragePlaytimeForever
	})

	return sortedGames
}
