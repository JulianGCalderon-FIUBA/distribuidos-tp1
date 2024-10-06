package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
	"sort"
)

type Aggregator struct {
	config         config
	m              *middleware.Middleware
	receivingQueue string
	games          []middleware.Game
}

type Batch middleware.Batch[middleware.Game]

func NewAggregator(config config) *Aggregator {
	m, err := middleware.NewMiddleware(config.RabbitIP)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}

	if err = m.InitTopNHistoricAvg(config.PartitionNumber); err != nil {
		log.Fatalf("failed to initialize middleware: %v", err)
	}
	return &Aggregator{
		config:         config,
		m:              m,
		receivingQueue: fmt.Sprintf("%v-%v", middleware.TopNHistoricAvgQueue, config.PartitionNumber),
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

		a.games = append(a.games, batch.Data...)

		if batch.EOF {
			log.Infof("Finished receiving batches for client %v", batch.ClientID)

			sort.Slice(a.games, func(i, j int) bool {
				return a.games[i].AveragePlaytimeForever > a.games[j].AveragePlaytimeForever
			})

			for _, game := range a.games[:a.config.TopN] {
				log.Infof("%v: %v", game.Name, game.AveragePlaytimeForever)
			}
		}
		_ = d.Ack(false)
	}

	return nil
}
