package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io"
	"math"
	"os/signal"
	"sort"
	"syscall"

	"github.com/spf13/viper"
)

type config struct {
	RabbitIP   string
	Percentile int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Percentile", 90)

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Percentile", "PERCENTILE")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	db        *database.Database
	sequencer *middleware.SequencerDisk

	output     string
	percentile float64
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
		if h.sequencer.EOF() {
			ch.Finish()
		}
		return nil
	}

	err = h.sequencer.MarkDisk(snapshot, batch.BatchID, batch.EOF)
	if err != nil {
		return err
	}

	percentileFile, err := snapshot.Append("percentile")
	if err != nil {
		return err
	}

	for _, g := range batch.Data {
		header := struct {
			AppId    uint64
			Stat     uint64
			NameSize uint64
		}{
			AppId:    g.AppID,
			Stat:     g.Stat,
			NameSize: uint64(len(g.Name)),
		}
		err = binary.Write(percentileFile, binary.LittleEndian, header)
		if err != nil {
			return err
		}

		err = binary.Write(percentileFile, binary.LittleEndian, []byte(g.Name))
		if err != nil {
			return err
		}
	}

	utils.MaybeExit(0.001)

	if h.sequencer.EOF() {
		err = h.conclude(ch, batch.Data)
		if err != nil {
			return err
		}
		ch.Finish()

		utils.MaybeExit(0.2)

	}

	return nil
}

func (h *handler) conclude(ch *middleware.Channel, data []middleware.GameStat) error {
	sorted, err := h.readData()
	if err != nil {
		return err
	}
	for _, stat := range data {
		sorted = sortedInsert(sorted, stat)
	}
	n := float64(len(sorted))
	index := max(0, int(math.Ceil(h.percentile/100.0*n))-1)
	results := sorted[index:]
	p := protocol.Q5Result{
		Percentile90: results,
	}

	return ch.SendAny(p, "", h.output)
}

func (h *handler) readData() ([]middleware.GameStat, error) {
	percentileFile, err := h.db.Get("percentile")
	if err != nil {
		return nil, err
	}
	defer percentileFile.Close()
	sorted := make([]middleware.GameStat, 0)

	for {
		header := struct {
			AppId    uint64
			Stat     uint64
			NameSize uint64
		}{}
		err := binary.Read(percentileFile, binary.LittleEndian, &header)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		name := make([]byte, header.NameSize)
		err = binary.Read(percentileFile, binary.LittleEndian, &name)
		if err != nil {
			return nil, err
		}
		stat := middleware.GameStat{
			AppID: header.AppId,
			Name:  string(name),
			Stat:  header.Stat,
		}
		sorted = sortedInsert(sorted, stat)
	}
	return sorted, nil
}

func sortedInsert(sorted []middleware.GameStat, stat middleware.GameStat) []middleware.GameStat {
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i].Stat >= stat.Stat })
	sorted = append(sorted, middleware.GameStat{})
	copy(sorted[i+1:], sorted[i:])
	sorted[i] = stat
	return sorted
}

func (h *handler) Free() error {
	return h.db.Delete()
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q5Result{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.GroupedQ5Percentile
	qOutput := middleware.Results

	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: qOutput},
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
				output:     middleware.Results,
				percentile: float64(cfg.Percentile),
				db:         db,
				sequencer:  sequencer,
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[*handler]{
			middleware.GroupedQ5Percentile: (*handler).handleBatch,
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
