package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
	"slices"
)

const (
	ActionGenre = "Action"
	IndieGenre  = "Indie"
)

func filterByGenre(gameBatch *middleware.BatchGame, genre string) []middleware.Game {
	var filteredGames []middleware.Game

	for _, game := range gameBatch.Data {
		if slices.Contains(game.Genres, genre) {
			filteredGames = append(filteredGames, game)
		}
	}

	return filteredGames
}

func filterGames(m *middleware.Middleware, config config) error {
	fmt.Println("Genre filter started")

	actionGames := middleware.BatchGame{}
	indieGames := middleware.BatchGame{}

	for {
		batch, err := m.ReceiveBatch("games")
		if err != nil {
			return err
		}

		batchGame := &middleware.BatchGame{}
		_, err = batchGame.Deserialize(batch)
		if err != nil {
			return err
		}

		actionGames.Data = append(actionGames.Data, filterByGenre(batchGame, ActionGenre)...)
		indieGames.Data = append(indieGames.Data, filterByGenre(batchGame, IndieGenre)...)

		actionGames.Data, err = sendFilteredGames(m, actionGames, config.BatchSize, ActionGenre)
		if err != nil {
			return err
		}
		fmt.Printf("Amount of Action games after sending: %#+v\n", len(actionGames.Data))

		indieGames.Data, err = sendFilteredGames(m, indieGames, config.BatchSize, IndieGenre)
		if err != nil {
			return err
		}
		fmt.Printf("Amount of Indie games after sending: %#+v\n", len(indieGames.Data))
	}
}

func sendFilteredGames(m *middleware.Middleware, batch middleware.BatchGame, batchSize int, genre string) ([]middleware.Game, error) {
	if len(batch.Data) >= batchSize {
		batchToSend := middleware.BatchGame{}
		batchToSend.Data = batch.Data[0:batchSize]

		err := m.SendFilteredGames(batchToSend, genre)
		if err != nil {
			fmt.Println("Could not send batch")
		}
		return batch.Data[batchSize:], nil
	}

	return batch.Data, nil
}
