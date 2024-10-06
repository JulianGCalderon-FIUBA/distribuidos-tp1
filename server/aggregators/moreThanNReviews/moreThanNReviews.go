package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
	"sync"
)

type Join struct {
	name       string
	review_num int
}

type GameReviewJoiner struct {
	cfg     config
	m       middleware.Middleware
	games   map[uint64]Join
	reviews map[uint64]int
}

type BatchGame middleware.Batch[middleware.Game]
type BatchReview middleware.Batch[middleware.Review]

func newAggregator(cfg config) (*GameReviewJoiner, error) {
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		return nil, err
	}

	err = m.InitMoreThanNReviews(cfg.ID)
	if err != nil {
		return nil, err
	}

	games := map[uint64]Join{}
	reviews := map[uint64]int{}

	return &GameReviewJoiner{
		cfg:     cfg,
		m:       *m,
		games:   games,
		reviews: reviews,
	}, nil
}

func (j *GameReviewJoiner) run() {
	log.Infof("Game review joiner is running")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err := j.receiveGames()
		if err != nil {
			log.Errorf("Failed to receive games: %v", err)
		}
	}()

	go func() {
		err := j.receiveReviews()
		if err != nil {
			log.Errorf("Failed to receive reviews: %v", err)
		}
	}()

	wg.Wait()
}

func (j *GameReviewJoiner) receiveGames() error {
	deliveryCh, err := j.m.ReceiveFromQueue(fmt.Sprintf("%v-x-%v", middleware.MoreThanNReviewsGamesQueue, j.cfg.ID))
	if err != nil {
		return err
	}

	for d := range deliveryCh {
		batch, err := middleware.Deserialize[BatchGame](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch: %v", err)
			err = d.Nack(false, false)
			if err != nil {
				return fmt.Errorf("failed to nack batch: %v", err)
			}
			continue
		}
		j.saveGames(batch)
		err = d.Ack(false)
		if err != nil {
			return fmt.Errorf("failed to ack batch: %v", err)
		}
		if batch.EOF {
			log.Infof("Received games EOF from client %v", batch.ClientID)
			break
		}
	}
	return nil
}

func (j *GameReviewJoiner) receiveReviews() error {
	deliveryCh, err := j.m.ReceiveFromQueue(fmt.Sprintf("%v-x-%v", middleware.NThousandEnglishReviewsQueue, j.cfg.ID))
	if err != nil {
		return err
	}

	for d := range deliveryCh {
		batch, err := middleware.Deserialize[BatchReview](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch: %v", err)
			err = d.Nack(false, false)
			if err != nil {
				return fmt.Errorf("failed to nack batch: %v", err)
			}
			continue
		}
		j.saveReviews(batch)
		j.sendResults()
		err = d.Ack(false)
		if err != nil {
			return fmt.Errorf("failed to ack batch: %v", err)
		}
		if batch.EOF {
			log.Infof("Received reviews EOF from client %v", batch.ClientID)
			break
		}
	}
	return nil
}

func (j *GameReviewJoiner) saveGames(batch BatchGame) {
	for _, game := range batch.Data {
		if val, ok := j.reviews[game.AppID]; ok {
			j.games[game.AppID] = Join{name: game.Name, review_num: val}
			delete(j.reviews, game.AppID)
			return
		} else {
			j.games[game.AppID] = Join{name: game.Name, review_num: 0}
		}
	}
}

func (j *GameReviewJoiner) saveReviews(batch BatchReview) {
	for _, review := range batch.Data {
		if val, ok := j.games[review.AppID]; ok {
			j.games[review.AppID] = Join{name: val.name, review_num: val.review_num + 1}
		} else {
			j.reviews[review.AppID] += 1
		}
	}
}

func (j *GameReviewJoiner) sendResults() {
	for k, v := range j.games {
		if v.review_num >= j.cfg.N {
			// después hay que enviar el resultado
			delete(j.games, k)
			log.Infof("Result: %v", v.name)
		}
	}
}
