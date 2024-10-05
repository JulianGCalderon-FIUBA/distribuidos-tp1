package main

import (
	"distribuidos/tp1/server/middleware"
	"strconv"
)

type Partitionable interface {
	PartitionId(partitionsNumber int) uint64
}

type Partitioner struct {
	cfg config
	m   middleware.Middleware
}

func newPartitioner(cfg config) (*Partitioner, error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}
	err = m.InitPartitioner(cfg.InputQueue, cfg.OutputExchange, cfg.PartitionsNumber)
	if err != nil {
		return nil, err
	}

	log.Infof("Initialized partitioner infrastructure")

	return &Partitioner{
		cfg: cfg,
		m:   *m,
	}, nil
}

func (p *Partitioner) run() error {
	dch, err := p.m.Consume(p.cfg.InputQueue)
	if err != nil {
		return err
	}

	for d := range dch {
		batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch %v", err)

			err = d.Nack(false, false)
			if err != nil {
				return err
			}

			continue
		}

		partitions := make([]middleware.Batch[middleware.Game], p.cfg.PartitionsNumber)
		for _, game := range batch {
			partitionId := game.AppID % uint64(p.cfg.PartitionsNumber)
			partitions[partitionId] = append(partitions[partitionId], game)
		}

		for partitionId, partition := range partitions {
			err = p.m.Send(partition, p.cfg.OutputExchange, strconv.Itoa(partitionId))
			if err != nil {
				log.Errorf("Failed to send batch: %v", err)
				continue
			}
		}

		err = d.Ack(false)
		if err != nil {
			return err
		}
	}

	return nil
}
