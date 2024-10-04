package main

import (
	"distribuidos/tp1/server/middleware"
	"strconv"
)

type DecadeFilter struct {
	config config
	m      *middleware.Middleware
}

type Batch middleware.Batch[middleware.Game]

func NewDecadeFilter(config config) *DecadeFilter {
	m, err := middleware.NewMiddleware(config.RabbitIP)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}

	if err = m.InitDecadeFilter(); err != nil {
		log.Fatalf("failed to initialize middleware: %v", err)
	}
	return &DecadeFilter{
		config: config,
		m:      m,
	}
}

func (df *DecadeFilter) start() error {
	log.Infof("Decade filter started")
	err := df.receive()
	if err != nil {
		log.Errorf("Failed to receive batches: %v", err)
	}

	return nil
}

func (df *DecadeFilter) receive() error {
	deliveryCh, err := df.m.ReceiveFromQueue(middleware.DecadeQueue)
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

		log.Infof("Amount of games received: %#+v\n", len(batch))

		_ = df.filterByDecade(batch, df.config.Decade)

		_ = d.Ack(false)
	}

	return nil
}

func (df *DecadeFilter) filterByDecade(batch Batch, decade int) Batch {
	var decadeGames Batch
	mask := strconv.Itoa(decade)[0:3]
	log.Info(mask)

	// for _, game := range batch {
	// 	if slices.Contains(strconv.Itoa(game.ReleaseYear), mask) {
	// 		decadeGames = append(decadeGames, game)
	// 	}
	// }

	return decadeGames
}
