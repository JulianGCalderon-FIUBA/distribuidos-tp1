package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"fmt"
	"maps"
	"slices"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP    string
	PartitionID int
	GameInput   string
	ReviewInput string
	Output      string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionID", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionID", "PARTITION_ID")
	_ = v.BindEnv("GameInput", "GAME_INPUT")
	_ = v.BindEnv("ReviewInput", "REVIEW_INPUT")
	_ = v.BindEnv("Output", "OUTPUT")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

type reviewHandler struct {
	games   map[uint64]middleware.ReviewsPerGame
	reviews map[uint64]int
	gameEof bool
}

type gameHandler struct {
	h *reviewHandler
}

func (h gameHandler) Aggregate(g middleware.Game) error {
	game := middleware.ReviewsPerGame{
		AppID: g.AppID,
		Name:  g.Name,
	}
	if count, ok := h.h.reviews[g.AppID]; ok {
		game.Reviews = count
		delete(h.h.reviews, g.AppID)
	}

	h.h.games[g.AppID] = game

	return nil
}

func (h reviewHandler) Aggregate(r middleware.Review) error {
	if game, ok := h.games[r.AppID]; ok {
		game.Reviews += 1
		h.games[r.AppID] = game
		return nil
	}
	if !h.gameEof {
		h.reviews[r.AppID] += 1
	}

	return nil
}

func (h gameHandler) Conclude() (any, error) {
	h.h.gameEof = true
	clear(h.h.reviews)
	return nil, nil
}
func (h reviewHandler) Conclude() (any, error) {
	games := slices.Collect(maps.Values(h.games))
	return games, nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	qName := fmt.Sprintf("%v-x-%v", middleware.TopNAmountReviewsGamesQueue, cfg.PartitionID)
	gameAggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    qName,
		Output:   cfg.Output,
	}

	h := reviewHandler{}
	gh := gameHandler{
		h: &h,
	}

	gameAgg, err := aggregator.NewAggregator(gameAggCfg, gh)
	utils.Expect(err, "Failed to create partitioner")
	err = gameAgg.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
	qName = fmt.Sprintf("%v-x-%v", middleware.TopNAmountReviewsQueue, cfg.PartitionID)
	reviewCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    qName,
		Output:   cfg.Output,
	}

	reviewAgg, err := aggregator.NewAggregator(reviewCfg, h)
	utils.Expect(err, "Failed to create partitioner")
	err = reviewAgg.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
}
