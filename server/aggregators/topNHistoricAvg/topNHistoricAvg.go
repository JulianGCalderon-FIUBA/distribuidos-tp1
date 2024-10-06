package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
)

type Aggregator struct {
	config config
	m      *middleware.Middleware
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
		config: config,
		m:      m,
	}
}

func (a *Aggregator) run() error {
	log.Infof("Top N historic avg started")
	log.Infof("Partition number: %v", a.config.PartitionNumber)

	err := a.receive()
	if err != nil {
		log.Errorf("Failed to receive batches: %v", err)
	}

	return nil
}

func (a *Aggregator) receive() error {
	receivingQueue := fmt.Sprintf("%v-%v", middleware.TopNHistoricAvgQueue, a.config.PartitionNumber)
	deliveryCh, err := a.m.ReceiveFromQueue(receivingQueue)
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

		log.Infof("Received batch with %v games", len(batch.Data))

		if batch.EOF {
			log.Infof("Received EOF")
		}

		// avg := a.calculateAvg(batch)
		// err = a.m.Send(avg, middleware.TopNAvgExchange, "")
		// if err != nil {
		// 	log.Errorf("Failed to send top N avg games batch: %v", err)
		// }
	}

	return nil
}

// func (a *Aggregator) calculateAvg(batch Batch) Batch {

// }
