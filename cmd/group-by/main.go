package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"maps"
	"os/signal"
	"slices"
	"syscall"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP    string
	PartitionID int
	GameInput   string
	ReviewInput string
	Output      string
	BatchSize   int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionID", "1")
	v.SetDefault("BatchSize", "100")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionID", "PARTITION_ID")
	_ = v.BindEnv("GameInput", "GAME_INPUT")
	_ = v.BindEnv("ReviewInput", "REVIEW_INPUT")
	_ = v.BindEnv("Output", "OUTPUT")
	_ = v.BindEnv("BatchSize", "BATCH_SIZE")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

type handler struct {
	games           map[uint64]middleware.ReviewsPerGame
	reviews         map[uint64]int
	gameSequencer   utils.Sequencer
	reviewSequencer utils.Sequencer
	batchSize       int
	output          string
}

func (h *handler) handleGame(_ *middleware.Channel, data []byte) error {
	batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](data)
	if err != nil {
		return err
	}

	h.gameSequencer.Mark(batch.BatchID, batch.EOF)

	for _, g := range batch.Data {
		game := middleware.ReviewsPerGame{
			AppID:   g.AppID,
			Name:    g.Name,
			Reviews: 0,
		}
		if count, ok := h.reviews[g.AppID]; ok {
			game.Reviews = uint64(count)
			delete(h.reviews, g.AppID)
		}

		h.games[g.AppID] = game
	}

	if h.gameSequencer.EOF() {
		clear(h.reviews)
	}

	return nil
}

func (h *handler) handleReview(ch *middleware.Channel, data []byte) error {
	batch, err := middleware.Deserialize[middleware.Batch[middleware.Review]](data)
	if err != nil {
		return err
	}

	h.reviewSequencer.Mark(batch.BatchID, batch.EOF)

	for _, r := range batch.Data {
		if game, ok := h.games[r.AppID]; ok {
			game.Reviews += 1
			h.games[r.AppID] = game
		} else if !h.gameSequencer.EOF() {
			h.reviews[r.AppID] += 1
		}
	}

	if h.reviewSequencer.EOF() {
		return h.Conclude(ch)
	}

	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	for k, v := range h.games {
		if v.Reviews == 0 {
			delete(h.games, k)
		}
	}

	batch := middleware.Batch[middleware.ReviewsPerGame]{
		Data:     []middleware.ReviewsPerGame{},
		ClientID: 1,
		BatchID:  0,
		EOF:      false,
	}

	games := slices.Collect(maps.Values(h.games))
	log.Infof("Sending %v reviews per game", len(games))

	for len(games) > 0 {
		currBatchSize := min(h.batchSize, len(games))
		var batchData []middleware.ReviewsPerGame
		games, batchData = games[currBatchSize:], games[:currBatchSize]

		batch.EOF = len(games) == 0
		batch.Data = batchData

		err := ch.Send(batch, "", h.output)
		if err != nil {
			return err
		}

		batch.BatchID += 1
	}

	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	output := middleware.Cat(cfg.Output, cfg.PartitionID)
	gameInput := middleware.Cat(cfg.GameInput, "x", cfg.PartitionID)
	reviewInput := middleware.Cat(cfg.ReviewInput, "x", cfg.PartitionID)

	nodeCfg := middleware.Config[handler]{
		RabbitIP: cfg.RabbitIP,
		Topology: middleware.Topology{
			Queues: []middleware.QueueConfig{
				{Name: gameInput},
				{Name: reviewInput},
				{Name: output},
			},
		},
		Builder: func(clientID int) handler {
			return handler{
				games:           make(map[uint64]middleware.ReviewsPerGame),
				reviews:         make(map[uint64]int),
				gameSequencer:   utils.NewSequencer(),
				reviewSequencer: utils.NewSequencer(),
				batchSize:       cfg.BatchSize,
				output:          output,
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[handler]{
			gameInput:   (*handler).handleGame,
			reviewInput: (*handler).handleReview,
		},
	}

	node, err := middleware.NewNode(nodeCfg)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
