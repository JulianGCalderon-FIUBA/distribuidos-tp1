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

type Count struct {
	ClientID uint64
	Count    map[Platform]int
}

type Aggregator struct {
	cfg   config
	m     middleware.Middleware
	count Count
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

	count := Count{
		ClientID: 0,
		Count:    make(map[Platform]int),
	}

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
				a.count.Count[Windows] += 1
			}
			if game.Linux {
				a.count.Count[Linux] += 1
			}
			if game.Mac {
				a.count.Count[Mac] += 1
			}
		}

		if batch.EOF {
			log.Infof("Received EOF from client: %v", batch.ClientID)
			log.Infof("Got %v games with linux support", a.count.Count[Linux])
			log.Infof("Got %v games with mac support", a.count.Count[Mac])
			log.Infof("Got %v games with windows support", a.count.Count[Windows])

			err := a.m.Send(a.count, "", middleware.GamesPerPlatformJoin)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
