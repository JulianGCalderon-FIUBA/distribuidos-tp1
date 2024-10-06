package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
)

type Platform int

const (
	Mac Platform = iota
	Linux
	Windows
)

type Aggregator struct {
	cfg   config
	m     middleware.Middleware
	count map[Platform]int
}

func newAggregator(cfg config) (*Aggregator, error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}
	err = m.InitPerPlatform(cfg.PartitionID)
	if err != nil {
		return nil, err
	}

	log.Infof("Initialized per platform infrastructure")

	count := make(map[Platform]int)

	return &Aggregator{
		cfg:   cfg,
		m:     *m,
		count: count,
	}, nil
}

func (a *Aggregator) run() error {
	xName := fmt.Sprintf("%v-x", middleware.GamesPerPlatformQueue)
	qName := fmt.Sprintf("%v-%v", xName, a.cfg.PartitionID)
	dch, err := a.m.ReceiveFromQueue(qName)
	if err != nil {
		return err
	}

	for d := range dch {
		batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](d.Body)
		if err != nil {
			return err
		}

		for _, game := range batch.Data {
			if game.Windows {
				a.count[Windows] += 1
			}
			if game.Linux {
				a.count[Linux] += 1
			}
			if game.Mac {
				a.count[Mac] += 1
			}
		}

		if batch.EOF {
			log.Infof("Received EOF from client: %v", batch.ClientID)
			log.Infof("Got %v games with linux support", a.count[Linux])
			log.Infof("Got %v games with mac support", a.count[Mac])
			log.Infof("Got %v games with windows support", a.count[Windows])
		}
	}

	return nil
}
