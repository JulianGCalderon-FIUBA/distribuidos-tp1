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

func (gf *GenreFilter) run() error {
	log.Infof("Genre filter started")
	err := gf.receive()
	if err != nil {
		log.Errorf("Failed to receive batches: %v", err)
	}

	return nil
}

func (gf *GenreFilter) receive() error {
	var indieSent int
	var actionSent int
	deliveryCh, err := gf.m.ReceiveFromQueue(middleware.GamesQueue)
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

		indie, action := gf.filterByGenre(batch)

		indieSent += len(indie.Data)
		actionSent += len(action.Data)

		err = gf.sendFilteredGames(indie, middleware.IndieGameKeys)
		if err != nil {
			_ = d.Nack(false, false)
			continue
		}

		err = gf.sendFilteredGames(action, middleware.ActionGameKeys)
		if err != nil {
			_ = d.Nack(false, false)
			continue
		}

		if batch.EOF {
			log.Infof("Finished filtering data for client: %v", batch.ClientID)
			log.Infof("Sent %v Indie games", indieSent)
			log.Infof("Sent %v Action games", actionSent)
		}

		_ = d.Ack(false)
	}

	return nil
}

func (gf *GenreFilter) filterByGenre(gameBatch Batch) (Batch, Batch) {
	indieGames := Batch{
		Data:     []middleware.Game{},
		ClientID: gameBatch.ClientID,
		BatchID:  gameBatch.BatchID,
		EOF:      gameBatch.EOF,
	}

	actionGames := Batch{
		Data:     []middleware.Game{},
		ClientID: gameBatch.ClientID,
		BatchID:  gameBatch.BatchID,
		EOF:      gameBatch.EOF,
	}

	for _, game := range gameBatch.Data {
		if slices.Contains(game.Genres, middleware.IndieGenre) {
			indieGames.Data = append(indieGames.Data, game)
		}

		if slices.Contains(game.Genres, middleware.ActionGenre) {
			actionGames.Data = append(actionGames.Data, game)
		}
	}

	return indieGames, actionGames
}

func (gf *GenreFilter) sendFilteredGames(batch Batch, genre string) error {
	err := gf.m.Send(batch, middleware.GenresExchange, genre)
	if err != nil {
		log.Errorf("Could not send batch")
	}
	return nil
}
