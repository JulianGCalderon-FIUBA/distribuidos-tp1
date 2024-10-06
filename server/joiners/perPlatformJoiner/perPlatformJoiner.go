package main

import (
	"distribuidos/tp1/server/middleware"
	"maps"
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

type Joiner struct {
	cfg   config
	m     middleware.Middleware
	count Count
}

func newJoiner(cfg config) (*Joiner, error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}
	err = m.InitPerPlatformJoiner()
	if err != nil {
		return nil, err
	}

	log.Infof("Initialized partitioner infrastructure")

	return &Joiner{
		cfg: cfg,
		m:   *m,
	}, nil
}

func (p *Joiner) run() error {
	dch, err := p.m.ReceiveFromQueue(middleware.GamesPerPlatformJoin)
	if err != nil {
		return err
	}

	_ = dch

	received := 0

	for msg := range dch {
		_ = msg
		count, err := middleware.Deserialize[Count](msg.Body)
		if err != nil {
			log.Errorf("Failed to deserialize")
			_ = msg.Nack(false, false)
			continue
		}

		p.count.Count = mergeCounts(p.count.Count, count.Count)

		received += 1

		if received == p.cfg.PartitionsNumber {
			break
		}
	}

	log.Infof("Received all partial results")
	log.Infof("Got %v games with linux support", p.count.Count[Linux])
	log.Infof("Got %v games with mac support", p.count.Count[Mac])
	log.Infof("Got %v games with windows support", p.count.Count[Windows])

	return nil
}

func mergeCounts(count1, count2 map[Platform]int) map[Platform]int {
	for k, v := range maps.All(count1) {
		count2[k] += v
	}

	return count2
}
