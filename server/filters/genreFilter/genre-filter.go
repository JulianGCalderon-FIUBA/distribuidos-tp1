package main

import (
	"distribuidos/tp1/server/middleware"
	"slices"
)

const (
	ActionGenre = "Action"
	IndieGenre  = "Indie"
)

func filterByGenre(gameBatch *middleware.Batch[middleware.Game], genre string) []middleware.Game {
	var filteredGames []middleware.Game

	for _, game := range *gameBatch {
		if slices.Contains(game.Genres, genre) {
			filteredGames = append(filteredGames, game)
		}
	}

	return filteredGames
}

func filterGames(m *middleware.Middleware, config config) error {
	log.Infof("Genre filter started")

	actionGames := middleware.Batch[middleware.Game]{}
	indieGames := middleware.Batch[middleware.Game]{}

	for {
		batch, err := m.ReceiveBatch(middleware.GamesQueue)
		if err != nil {
			return err
		}

		batchGame := &middleware.Batch[middleware.Game]{}
		err = middleware.DeserializeInto(batch, &batchGame)
		if err != nil {
			return err
		}

		actionGames = append(actionGames, filterByGenre(batchGame, ActionGenre)...)
		indieGames = append(indieGames, filterByGenre(batchGame, IndieGenre)...)

		actionGames, err = sendFilteredGames(m, actionGames, config.BatchSize, ActionGenre)
		if err != nil {
			return err
		}
		log.Infof("Amount of Action games after sending: %#+v\n", len(actionGames))

		indieGames, err = sendFilteredGames(m, indieGames, config.BatchSize, IndieGenre)
		if err != nil {
			return err
		}
		log.Infof("Amount of Indie games after sending: %#+v\n", len(indieGames))
	}
}

func sendFilteredGames(m *middleware.Middleware, batch middleware.Batch[middleware.Game], batchSize int, genre string) ([]middleware.Game, error) {
	if len(batch) >= batchSize {
		batchToSend := batch[0:batchSize]

		err := m.SendToExchange(batchToSend, middleware.GenresExchange, genre)
		if err != nil {
			log.Errorf("Could not send batch")
		}
		return batch[batchSize:], nil
	}

	return batch, nil
}
