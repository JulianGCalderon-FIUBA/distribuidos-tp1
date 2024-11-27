package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"io"
	"os"
	"os/signal"
	"path"
	"slices"
	"strconv"
	"syscall"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

const GAMES_DIR = "games"

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

	batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](data)
	if err != nil {
		return err
	}

	if h.gameSequencer.Seen(batch.BatchID) {
		return nil
	}

	err = h.gameSequencer.MarkDisk(snapshot, batch.BatchID, batch.EOF)
	if err != nil {
		return err
	}

	games := make(map[uint64]string)

	for _, g := range batch.Data {
		path := path.Join(GAMES_DIR, strconv.Itoa(int(g.AppID)))
		exists, err := snapshot.Exists(path)
		if err != nil {
			return err
		}
		if exists {
			err = rename(snapshot, path, g.Name)
			if err != nil {
				return err
			}
		} else {
			err = insert(snapshot, path, g)
			if err != nil {
				return err
			}
		}
		games[g.AppID] = g.Name
	}

	if h.gameSequencer.EOF() {
		log.Infof("Received game EOF")
	}

	if h.gameSequencer.EOF() && h.reviewSequencer.EOF() {
		return h.conclude(ch, nil, games)
	}

	return nil
}

func rename(snapshot *database.Snapshot, path string, name string) error {
	statsFile, err := snapshot.Append(path)
	if err != nil {
		return err
	}
	return binary.Write(statsFile, binary.LittleEndian, []byte(name))
}

func insert(snapshot *database.Snapshot, path string, g middleware.Game) error {
	statsFile, err := snapshot.Create(path)
	if err != nil {
		return err
	}
	header := struct {
		AppId uint64
		Stats uint64
	}{
		AppId: g.AppID,
	}
	err = binary.Write(statsFile, binary.LittleEndian, header)
	if err != nil {
		return nil
	}
	return binary.Write(statsFile, binary.LittleEndian, []byte(g.Name))
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

	batch, err := middleware.Deserialize[middleware.Batch[middleware.Review]](data)
	if err != nil {
		return err
	}
	/* log.Infof("Received review batch %v", batch.BatchID)
	return nil */

	if h.reviewSequencer.Seen(batch.BatchID) {
		log.Infof("already seen batch %v", batch.BatchID)
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
		path := path.Join(GAMES_DIR, strconv.Itoa(int(id)))
		exists, err := snapshot.Exists(path)
		if err != nil {
			return err
		}
		statsFile, err := snapshot.Update(path)
		if err != nil {
			return err
		}

		header := struct {
			Id   uint64
			Stat uint64
		}{
			Id:   id,
			Stat: stat,
		}

		if exists {
			err = binary.Read(statsFile, binary.LittleEndian, &header)
			if err != nil {
				return err
			}
			stat += header.Stat
		}
		err = increment(statsFile, id, stat)
		if err != nil {
			return err
		}
	}

	if h.reviewSequencer.EOF() {
		log.Infof("Received review EOF")
	}

	if h.reviewSequencer.EOF() && h.gameSequencer.EOF() {
		return h.conclude(ch, reviews, nil)
	}

	return nil
}

func increment(statsFile *os.File, id uint64, stat uint64) error {
	_, err := statsFile.Seek(0, 0)
	if err != nil {
		return err
	}
	err = binary.Write(statsFile, binary.LittleEndian, id)
	if err != nil {
		return err
	}
	return binary.Write(statsFile, binary.LittleEndian, stat)
}

func (h *handler) conclude(ch *middleware.Channel, reviews map[uint64]uint64, games map[uint64]string) error {
	stats, err := h.getAll(reviews, games)
	if err != nil {
		return err
	}

	stats = slices.DeleteFunc(stats, func(g middleware.GameStat) bool {
		return g.Stat == 0 || g.Name == ""
	})

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

func (h *handler) getAll(reviews map[uint64]uint64, games map[uint64]string) ([]middleware.GameStat, error) {
	entries, err := h.db.GetAll(GAMES_DIR)
	if err != nil {
		return nil, err
	}

	stats := make([]middleware.GameStat, 0)

	for _, e := range entries {
		file, err := h.db.Get(e)
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		var header struct {
			AppId uint64
			Stat  uint64
		}
		n, err := binary.Decode(content, binary.LittleEndian, &header)
		if err != nil {
			return nil, err
		}

		if reviews != nil {
			stat, ok := reviews[header.AppId]
			if ok {
				header.Stat += stat
			}
		}

		name := string(content[n:])

		if games != nil {
			gname, ok := games[header.AppId]
			if ok {
				name = gname
			}
		}

		g := middleware.GameStat{
			AppID: header.AppId,
			Stat:  header.Stat,
			Name:  name,
		}
		stats = append(stats, g)
	}

	return stats, nil
}

func (h *handler) Free() error {
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

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			gameSequencer := middleware.NewSequencerDisk("game-sequencer")
			err = gameSequencer.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")
			reviewSequencer := middleware.NewSequencerDisk("review-sequencer")
			err = reviewSequencer.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")

			return &handler{
				db:              db,
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
