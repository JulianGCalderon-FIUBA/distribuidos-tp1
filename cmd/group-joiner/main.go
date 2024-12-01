package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	Partitions int
	Input      string
	Output     string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Partitions", "PARTITIONS")
	_ = v.BindEnv("Input", "INPUT")
	_ = v.BindEnv("Output", "OUTPUT")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

type handler struct {
	db          *database.Database
	output      string
	lastBatchId int
	sequencers  map[int]*middleware.SequencerDisk
}

func buildHandler(partition int) middleware.HandlerFunc[*handler] {
	return func(h *handler, ch *middleware.Channel, data []byte) error {
		return h.handleBatch(ch, data, partition)
	}
}

func (h *handler) handleBatch(ch *middleware.Channel, data []byte, partition int) error {
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

	if h.sequencers[partition].Seen(batch.BatchID) {
		allEof := true
		for _, v := range h.sequencers {
			allEof = allEof && v.EOF()
		}
		if allEof {
			ch.Finish()
		}
		return nil
	}
	err = h.sequencers[partition].MarkDisk(snapshot, batch.BatchID, batch.EOF)
	if err != nil {
		return err
	}

	file, err := snapshot.Update("id")
	if err != nil {
		return err
	}
	var id uint64
	err = binary.Read(file, binary.LittleEndian, &id)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	b := middleware.Batch[middleware.GameStat]{
		Data:    batch.Data,
		BatchID: int(id),
		EOF:     false,
	}

	id += 1
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian, id)
	if err != nil {
		return err
	}

	err = ch.Send(b, "", h.output)
	if err != nil {
		return err
	}

	if h.sequencers[partition].EOF() {
		log.Infof("Received EOF from partition %v", partition)
	}

	allEof := true
	for _, v := range h.sequencers {
		allEof = allEof && v.EOF()
	}

	utils.MaybeExit(0.001)

	if allEof {
		b := middleware.Batch[middleware.GameStat]{
			BatchID: int(id),
			EOF:     true,
		}
		err := ch.Send(b, "", h.output)
		if err != nil {
			return err
		}
		ch.Finish()

		utils.MaybeExit(0.50)

	}

	return nil
}

func (h *handler) Free() error {
	return h.db.Delete()
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	queues := make([]middleware.QueueConfig, 0)
	endpoints := make(map[string]middleware.HandlerFunc[*handler], 0)

	for i := 1; i <= cfg.Partitions; i++ {
		qName := middleware.Cat(cfg.Input, i)
		qcfg := middleware.QueueConfig{
			Name: qName,
		}
		queues = append(queues, qcfg)
		endpoints[qName] = buildHandler(i)
	}

	output := middleware.QueueConfig{Name: cfg.Output}

	queues = append(queues, output)

	err = middleware.Topology{
		Queues: queues,
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			sequencers := make(map[int]*middleware.SequencerDisk)
			for i := 1; i <= cfg.Partitions; i++ {
				sequencers[i] = middleware.NewSequencerDisk(fmt.Sprintf("sequencer-%v", i))
				err = sequencers[i].LoadDisk(db)
				utils.Expect(err, "unrecoverable error")
			}
			return &handler{
				db:          db,
				output:      cfg.Output,
				lastBatchId: 0,
				sequencers:  sequencers,
			}
		},
		Endpoints: endpoints,
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
