package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"fmt"

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

type reviewHandler struct{}

type gameHandler struct {
	h *reviewHandler
}

func (h gameHandler) Aggregate(g middleware.Game) error {
	return nil
}
func (h gameHandler) Conclude() (any, error) {
	return 5, nil
}
func (h reviewHandler) Aggregate(g middleware.Review) error {
	return nil
}
func (h reviewHandler) Conclude() (any, error) {
	return 5, nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	qName := fmt.Sprintf("%v-x-%v", middleware.TopNAmountReviewsGamesQueue, cfg.PartitionID)
	gameAggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    qName,
		Output:   "demo",
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
		Output:   "demo",
	}

	reviewAgg, err := aggregator.NewAggregator(reviewCfg, h)
	utils.Expect(err, "Failed to create partitioner")
	err = reviewAgg.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")
}
