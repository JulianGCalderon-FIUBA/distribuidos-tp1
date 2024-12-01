package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"io"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP    string
	PartitionID int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionID", "0")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionID", "PARTITION_ID")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

type Platform string

const (
	Mac     Platform = "mac"
	Linux   Platform = "linux"
	Windows Platform = "windows"
)

type handler struct {
	db        *database.Database
	output    string
	sequencer *middleware.SequencerDisk
}

func (h *handler) handleGame(ch *middleware.Channel, data []byte) (err error) {
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

	var windowsCounter uint64
	var linuxCounter uint64
	var macCounter uint64

	exists, err := snapshot.Exists("counter")
	if err != nil {
		return err
	}
	counterFile, err := snapshot.Update("counter")
	if err != nil {
		return err
	}

	if exists {
		err = binary.Read(counterFile, binary.LittleEndian, &windowsCounter)
		if err != nil {
			return err
		}
		err = binary.Read(counterFile, binary.LittleEndian, &linuxCounter)
		if err != nil {
			return err
		}
		err = binary.Read(counterFile, binary.LittleEndian, &macCounter)
		if err != nil {
			return err
		}
	}

	for _, g := range batch.Data {
		if g.Windows {
			windowsCounter += 1
		}
		if g.Linux {
			linuxCounter += 1
		}
		if g.Mac {
			macCounter += 1
		}
	}

	_, err = counterFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	err = binary.Write(counterFile, binary.LittleEndian, windowsCounter)
	if err != nil {
		return err
	}
	err = binary.Write(counterFile, binary.LittleEndian, linuxCounter)
	if err != nil {
		return err
	}
	err = binary.Write(counterFile, binary.LittleEndian, macCounter)
	if err != nil {
		return err
	}

	utils.MaybeExit(0.001)

	if h.sequencer.EOF() {
		count := map[Platform]int{
			Windows: int(windowsCounter),
			Linux:   int(linuxCounter),
			Mac:     int(macCounter),
		}

		for k, v := range count {
			log.Infof("Found %v games with %v support", v, string(k))
		}

		err := ch.Send(count, "", h.output)
		if err != nil {
			return err
		}

		utils.MaybeExit(0.50)

		ch.Finish()
	}

	return nil
}

func (h *handler) Free() error {
	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	inputQ := middleware.Cat(middleware.GamesQ1, "x", cfg.PartitionID)
	outputQ := middleware.Cat(middleware.PartialQ1, cfg.PartitionID)
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: inputQ},
			{Name: outputQ},
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

			return &handler{
				db:        db,
				output:    outputQ,
				sequencer: sequencer,
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[*handler]{
			inputQ: (*handler).handleGame,
		},
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
