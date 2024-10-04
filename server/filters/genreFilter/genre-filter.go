package main

import (
	"distribuidos/tp1/server/middleware"
	"slices"
)

type GenreFilter struct {
	config config
	m      *middleware.Middleware
}

type Batch middleware.Batch[middleware.Game]

func NewGenreFilter(config config) *GenreFilter {
	m, err := middleware.NewMiddleware(config.RabbitIP)
	if err != nil {
		log.Fatalf("failed to create middleware: %v", err)
	}

	if err = m.InitGenreFilter(); err != nil {
		log.Fatalf("failed to initialize middleware: %v", err)
	}
	return &GenreFilter{
		config: config,
		m:      m,
	}
}

func (gf *GenreFilter) start() error {
	log.Infof("Genre filter started")
	err := gf.receive()
	if err != nil {
		log.Errorf("Failed to receive batches: %v", err)
	}

	return nil
}

func (gf *GenreFilter) receive() error {
	// lo dejo comentado para testear
	// indieGames := 0
	// actionGames := 0

	deliveryCh, err := gf.m.ReceiveFromQueue(middleware.GamesQueue)
	for d := range deliveryCh {
		if err != nil {
			_ = d.Nack(false, false)
			return err
		}

		batchGame, err := middleware.Deserialize[Batch](d.Body)
		if err != nil {
			_ = d.Nack(false, false)
			return err
		}

		// log.Infof("Amount of games received: %#+v\n", len(batchGame))
		indie, action := gf.filterByGenre(batchGame)

		err = gf.sendFilteredGames(indie, middleware.IndieGenre)
		if err != nil {
			_ = d.Nack(false, false)
			return err
		}

		// lo dejo comentado para testear
		// indieGames += len(indie)
		// actionGames += len(action)
		// log.Infof("Amount of Indie games sent: %v\n", indieGames)
		// log.Infof("Amount of Action games sent: %v\n", actionGames)

		err = gf.sendFilteredGames(action, middleware.ActionGenre)
		if err != nil {
			_ = d.Nack(false, false)
			return err
		}

		_ = d.Ack(false)
	}

	return nil
}

func (gf *GenreFilter) filterByGenre(gameBatch Batch) (Batch, Batch) {
	var indieGames Batch
	var actionGames Batch

	for _, game := range gameBatch {
		if slices.Contains(game.Genres, middleware.IndieGenre) {
			indieGames = append(indieGames, game)
		}

		if slices.Contains(game.Genres, middleware.ActionGenre) {
			actionGames = append(actionGames, game)
		}

	}

	return indieGames, actionGames
}

func (gf *GenreFilter) sendFilteredGames(batch Batch, genre string) error {
	if len(batch) > 0 {

		err := gf.m.Send(batch, middleware.GenresExchange, genre)
		if err != nil {
			log.Errorf("Could not send batch")
		}
	}
	return nil
}
