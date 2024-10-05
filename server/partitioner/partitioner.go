package main

import (
	"distribuidos/tp1/server/middleware"
	"strconv"
)

type Partitionable interface {
	PartitionId(partitionsNumber int) uint64
}

type Partitioner[T Partitionable] struct {
	cfg config
	m   middleware.Middleware
}

func newPartitioner[T Partitionable](cfg config) (*Partitioner[T], error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}
	err = m.InitPartitioner(cfg.InputQueue, cfg.OutputExchange, cfg.PartitionsNumber)
	if err != nil {
		return nil, err
	}

	log.Infof("Initialized partitioner infrastructure")

	return &Partitioner[T]{
		cfg: cfg,
		m:   *m,
	}, nil
}

func (p *Partitioner[T]) run() error {
	dch, err := p.m.Consume(p.cfg.InputQueue)
	if err != nil {
		return err
	}

	for d := range dch {
		batch, err := middleware.Deserialize[middleware.Batch[T]](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch %v", err)

			err = d.Nack(false, false)
			if err != nil {
				return err
			}

			continue
		}

		partitions := make([]middleware.Batch[T], p.cfg.PartitionsNumber)
		for _, game := range batch {
			partitionId := game.PartitionId(p.cfg.PartitionsNumber)
			partitions[partitionId] = append(partitions[partitionId], game)
		}

		for partitionId, partition := range partitions {
			err = p.m.Publish(partition, p.cfg.OutputExchange, strconv.Itoa(partitionId))
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
