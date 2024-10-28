package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"slices"
	"strconv"
	"sync"
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
	Path        string
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
	_ = v.BindEnv("Path", "PATH")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

type handler struct {
	diskMap         *middleware.DiskMap
	gameSequencer   *utils.Sequencer
	reviewSequencer *utils.Sequencer
	batchSize       int
	output          string
	mu              *sync.Mutex
}

func (h *handler) handleGame(_ *middleware.Channel, data []byte) error {
	batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](data)
	if err != nil {
		return err
	}

	h.gameSequencer.Mark(batch.BatchID, batch.EOF)

	for _, g := range batch.Data {
		h.mu.Lock()
		saved, err := h.diskMap.Get(g.AppID)
		if err != nil {
			return err
		}

		if saved != nil {
			saved.Name = g.Name
			err = h.diskMap.Insert(*saved)
			if err != nil {
				return err
			}
		} else {
			stat := middleware.GameStat{
				AppID: g.AppID,
				Name:  g.Name,
			}
			err = h.diskMap.Insert(stat)
			if err != nil {
				return err
			}
		}

		h.mu.Unlock()
	}

	if h.gameSequencer.EOF() {
		log.Infof("Received game EOF")
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
		h.mu.Lock()
		saved, err := h.diskMap.Get(r.AppID)
		if err != nil {
			return err
		}

		if saved != nil {
			saved.Stat += 1
			err = h.diskMap.Insert(*saved)
			if err != nil {
				return err
			}
		} else {
			stat := middleware.GameStat{
				AppID: r.AppID,
				Name:  "",
				Stat:  1,
			}
			err = h.diskMap.Insert(stat)
			if err != nil {
				return err
			}
		}

		h.mu.Unlock()
	}

	if h.reviewSequencer.EOF() {
		log.Infof("Received review EOF")
		return h.Conclude(ch)
	}

	return nil
}

func (h *handler) Conclude(ch *middleware.Channel) error {
	h.mu.Lock()
	games, err := h.diskMap.GetAll()
	if err != nil {
		return err
	}
	h.mu.Unlock()

	games = slices.DeleteFunc(games, func(g middleware.GameStat) bool {
		return g.Stat == 0
	})

	log.Infof("found %v games with more than 0 reviews", len(games))

	batch := middleware.Batch[middleware.GameStat]{
		Data:     []middleware.GameStat{},
		ClientID: 1,
		BatchID:  0,
		EOF:      false,
	}

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
			path := middleware.Cat(cfg.Path, strconv.Itoa(clientID))
			diskMap, err := middleware.NewDiskMap(path)
			utils.Expect(err, "Failed to build new disk map")

			return handler{
				diskMap:         diskMap,
				gameSequencer:   utils.NewSequencer(),
				reviewSequencer: utils.NewSequencer(),
				batchSize:       cfg.BatchSize,
				output:          output,
				mu:              &sync.Mutex{},
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
