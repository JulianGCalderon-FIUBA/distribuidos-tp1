package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"maps"
	"os/signal"
	"slices"
	"syscall"

	logging "github.com/op/go-logging"
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
	games           map[uint64]middleware.GameStat
	reviews         map[uint64]int
	gameSequencer   *utils.Sequencer
	reviewSequencer *utils.Sequencer
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
		game := middleware.GameStat{
			AppID: g.AppID,
			Name:  g.Name,
			Stat:  0,
		}
		if count, ok := h.reviews[g.AppID]; ok {
			game.Stat = uint64(count)
			delete(h.reviews, g.AppID)
		}

		h.games[g.AppID] = game
	}

	if h.gameSequencer.EOF() {
		log.Infof("Received game EOF")
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
			game.Stat += 1
			h.games[r.AppID] = game
		} else if !h.gameSequencer.EOF() {
			h.reviews[r.AppID] += 1
		}
	}

	if h.reviewSequencer.EOF() {
		log.Infof("Received review EOF")
		return h.Conclude(ch)
	}

	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	for k, v := range h.games {
		if v.Stat == 0 {
			delete(h.games, k)
		}
	}

	batch := middleware.Batch[middleware.GameStat]{
		Data:     []middleware.GameStat{},
		ClientID: 1,
		BatchID:  0,
		EOF:      false,
	}

	games := slices.Collect(maps.Values(h.games))

	for len(games) > 0 {
		currBatchSize := min(h.batchSize, len(games))
		var batchData []middleware.GameStat
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

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	output := middleware.Cat(cfg.Output, cfg.PartitionID)
	gameInput := middleware.Cat(cfg.GameInput, "x", cfg.PartitionID)
	reviewInput := middleware.Cat(cfg.ReviewInput, "x", cfg.PartitionID)
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: gameInput},
			{Name: reviewInput},
			{Name: output},
		},
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[handler]{
		Builder: func(clientID int) handler {
			return handler{
				games:           make(map[uint64]middleware.GameStat),
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

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
