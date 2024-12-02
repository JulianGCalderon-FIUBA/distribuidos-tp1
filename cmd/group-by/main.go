package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"slices"
	"syscall"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

const GAMES_DIR string = "games"

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
	db              *database.Database
	diskMap         *middleware.DiskMap
	gameSequencer   *middleware.SequencerDisk
	reviewSequencer *middleware.SequencerDisk

	batchSize int
	output    string
}

func (h *handler) handleGame(ch *middleware.Channel, data []byte) error {
	snapshot, err := h.db.NewSnapshot()
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			cerr := snapshot.Commit()
			utils.Expect(cerr, "unrecoverable error")
		default:
			cerr := snapshot.Abort()
			utils.Expect(cerr, "unrecoverable error")
		}
	}()

	h.diskMap.Start()

	batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](data)
	if err != nil {
		return err
	}

	if h.gameSequencer.Seen(batch.BatchID) {
		if h.reviewSequencer.EOF() && h.gameSequencer.EOF() {
			ch.Finish()
		}
		return nil
	}

	err = h.gameSequencer.MarkDisk(snapshot, batch.BatchID, batch.EOF)
	if err != nil {
		return err
	}

	for _, g := range batch.Data {
		err = h.diskMap.Rename(snapshot, g.AppID, g.Name)
	}

	utils.MaybeExit(0.0001)

	if h.gameSequencer.EOF() {
		log.Infof("Received game EOF")
	}

	if h.gameSequencer.EOF() && h.reviewSequencer.EOF() {
		err = h.conclude(ch)
		utils.MaybeExit(0.2)
		return err
	}

	return nil
}

func (h *handler) handleReview(ch *middleware.Channel, data []byte) error {
	snapshot, err := h.db.NewSnapshot()
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			cerr := snapshot.Commit()
			utils.Expect(cerr, "unrecoverable error")
		default:
			cerr := snapshot.Abort()
			utils.Expect(cerr, "unrecoverable error")
		}
	}()

	h.diskMap.Start()

	batch, err := middleware.Deserialize[middleware.Batch[middleware.Review]](data)
	if err != nil {
		return err
	}

	if h.reviewSequencer.Seen(batch.BatchID) {
		if h.reviewSequencer.EOF() && h.gameSequencer.EOF() {
			ch.Finish()
		}
		return nil
	}

	err = h.reviewSequencer.MarkDisk(snapshot, batch.BatchID, batch.EOF)
	if err != nil {
		return err
	}

	reviews := make(map[uint64]uint64)
	for _, r := range batch.Data {
		reviews[r.AppID] += 1
	}

	for id, stat := range reviews {
		err = h.diskMap.Increment(snapshot, id, stat)
		if err != nil {
			return err
		}
	}

	utils.MaybeExit(0.0001)

	if h.reviewSequencer.EOF() {
		log.Infof("Received review EOF")
	}

	if h.reviewSequencer.EOF() && h.gameSequencer.EOF() {
		err := h.conclude(ch)
		utils.MaybeExit(0.2)
		return err
	}

	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	stats, err := h.getAll()
	if err != nil {
		return err
	}

	batch := middleware.Batch[middleware.GameStat]{
		Data:    []middleware.GameStat{},
		BatchID: 0,
		EOF:     false,
	}
	if len(stats) == 0 {
		batch.EOF = true
		return ch.Send(batch, "", h.output)
	}

	for len(stats) > 0 {
		currBatchSize := min(h.batchSize, len(stats))
		var batchData []middleware.GameStat
		stats, batchData = stats[currBatchSize:], stats[:currBatchSize]

		batch.EOF = len(stats) == 0
		batch.Data = batchData

		err := ch.Send(batch, "", h.output)
		if err != nil {
			return err
		}

		batch.BatchID += 1
	}

	ch.Finish()
	return nil
}

func (h *handler) getAll() ([]middleware.GameStat, error) {
	stats, err := h.diskMap.GetAll(h.db)
	if err != nil {
		return nil, err
	}

	stats = slices.DeleteFunc(stats, func(g middleware.GameStat) bool {
		return g.Stat == 0 || g.Name == ""
	})

	return stats, nil
}

func (h *handler) Free() error {
	return h.db.Delete()
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

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			diskMap := middleware.NewDiskMap(GAMES_DIR)
			utils.Expect(err, "unrecoverable error")

			gameSequencer := middleware.NewSequencerDisk("game-sequencer")
			err = gameSequencer.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")
			reviewSequencer := middleware.NewSequencerDisk("review-sequencer")
			err = reviewSequencer.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")

			return &handler{
				db:              db,
				diskMap:         diskMap,
				gameSequencer:   gameSequencer,
				reviewSequencer: reviewSequencer,
				batchSize:       cfg.BatchSize,
				output:          output,
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[*handler]{
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
