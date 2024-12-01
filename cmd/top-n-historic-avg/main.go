package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP    string
	TopN        int
	PartitionId int
	Input       string
	Output      string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("TopN", "10")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("TopN", "TOP_N")
	_ = v.BindEnv("PartitionId", "PARTITION_ID")
	_ = v.BindEnv("Input", "INPUT")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	db        *database.Database
	output    string
	topN      *middleware.TopNDisk
	sequencer *middleware.SequencerDisk
}

func (h *handler) handleBatch(ch *middleware.Channel, data []byte) error {
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

	if h.sequencer.Seen(batch.BatchID) {
		return nil
	}
	err = h.sequencer.MarkDisk(snapshot, batch.BatchID, batch.EOF)
	if err != nil {
		return err
	}

	for _, g := range batch.Data {
		gStat := middleware.GameStat{
			AppID: g.AppID,
			Name:  g.Name,
			Stat:  g.AveragePlaytimeForever,
		}
		h.topN.Put(gStat)
	}

	err = h.topN.Save(snapshot)
	if err != nil {
		return err
	}

	utils.MaybeExit(0.001)

	if h.sequencer.EOF() {
		err := h.conclude(ch)
		if err != nil {
			return err
		}

		utils.MaybeExit(0.50)

		ch.Finish()
	}
	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	games := h.topN.Get()

	err := ch.Send(games, "", h.output)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) Free() error {
	return h.db.RemoveAll()
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.Cat(cfg.Input, "x", cfg.PartitionId)
	qOutput := middleware.Cat(middleware.PartialQ2, cfg.PartitionId)
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: qOutput},
		},
	}.Declare(ch)

	if err != nil {
		utils.Expect(err, "Failed to declare topology")
	}

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			sequencer := middleware.NewSequencerDisk("sequencer")
			err = sequencer.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")

			topN := middleware.NewTopNDisk("TopN", cfg.TopN)
			err = topN.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")

			return &handler{
				db:        db,
				output:    qOutput,
				sequencer: sequencer,
				topN:      topN,
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[*handler]{
			qInput: (*handler).handleBatch,
		},
		OutputConfig: middleware.Output{
			Exchange: "",
			Keys:     []string{qOutput},
		},
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
