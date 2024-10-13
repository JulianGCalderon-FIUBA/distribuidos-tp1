package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/aggregator"
	"distribuidos/tp1/utils"
	"maps"
	"os/signal"
	"slices"
	"sync"
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

type reviewHandler struct {
	games     map[uint64]middleware.GameStat
	reviews   map[uint64]int
	gameEof   bool
	batchSize int
	m         *sync.Mutex
	output    string
}

type gameHandler struct {
	h *reviewHandler
}

func (h *gameHandler) Aggregate(_ *middleware.Channel, batch middleware.Batch[middleware.Game]) error {
	h.h.m.Lock()
	defer h.h.m.Unlock()

	for _, g := range batch.Data {
		game := middleware.GameStat{
			AppID: g.AppID,
			Name:  g.Name,
			Stat:  0,
		}
		if count, ok := h.h.reviews[g.AppID]; ok {
			game.Stat = uint64(count)
			delete(h.h.reviews, g.AppID)
		}

		h.h.games[g.AppID] = game
	}

	return nil
}

func (h *reviewHandler) Aggregate(_ *middleware.Channel, batch middleware.Batch[middleware.Review]) error {
	h.m.Lock()
	defer h.m.Unlock()

	for _, r := range batch.Data {
		if game, ok := h.games[r.AppID]; ok {
			game.Stat += 1
			h.games[r.AppID] = game
		} else if !h.gameEof {
			h.reviews[r.AppID] += 1
		}
	}

	return nil
}

func (h *gameHandler) Conclude(_ *middleware.Channel) error {
	h.h.m.Lock()
	defer h.h.m.Unlock()

	h.h.gameEof = true
	clear(h.h.reviews)
	return nil
}

func (h *reviewHandler) Conclude(ch *middleware.Channel) error {
	h.m.Lock()
	defer h.m.Unlock()

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
	log.Infof("Sending %v reviews per game", len(games))
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

	output := middleware.Cat(cfg.Output, cfg.PartitionID)
	input := middleware.Cat(cfg.GameInput, "x", cfg.PartitionID)
	gameAggCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    input,
		Output:   output,
	}

	h := reviewHandler{
		games:     make(map[uint64]middleware.GameStat),
		reviews:   make(map[uint64]int),
		gameEof:   false,
		batchSize: cfg.BatchSize,
		m:         &sync.Mutex{},
		output:    output,
	}
	gh := gameHandler{
		h: &h,
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	go func() {
		defer wg.Done()
		gameAgg, err := aggregator.NewAggregator(gameAggCfg, &gh)
		utils.Expect(err, "Failed to create game aggregator")
		err = gameAgg.Run(ctx)
		utils.Expect(err, "Failed to run game aggregator")
	}()

	input = middleware.Cat(cfg.ReviewInput, "x", cfg.PartitionID)
	reviewCfg := aggregator.Config{
		RabbitIP: cfg.RabbitIP,
		Input:    input,
		Output:   output,
	}

	reviewAgg, err := aggregator.NewAggregator(reviewCfg, &h)
	utils.Expect(err, "Failed to create review aggregator")
	err = reviewAgg.Run(ctx)
	utils.Expect(err, "Failed to run review aggregator")

	wg.Wait()
}
