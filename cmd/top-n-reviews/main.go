package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/gob"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP    string
	PartitionID int
	N           int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("N", "5")
	v.SetDefault("PartitionID", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("N", "N")
	_ = v.BindEnv("PartitionID", "PARTITION_ID")

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

	batch, err := middleware.Deserialize[middleware.Batch[middleware.GameStat]](data)
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

	for _, stat := range batch.Data {
		h.topN.Put(stat)
	}

	err = h.topN.Save(snapshot)
	if err != nil {
		return err
	}

	if h.sequencer.EOF() {
		err := h.conclude(ch)
		if err != nil {
			return err
		}

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
	return h.db.Delete()
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q3Result{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.Cat(middleware.GroupedQ3, cfg.PartitionID)
	qOutput := middleware.Cat(middleware.PartialQ3, cfg.PartitionID)
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: middleware.PartialQ3},
		},
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			sequencer := middleware.NewSequencerDisk("sequencer")
			err = sequencer.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")

			topN := middleware.NewTopNDisk("TopN", cfg.N)
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
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
