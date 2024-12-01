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

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	TopN       int
	Partitions int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("TopN", "10")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("TopN", "TOP_N")
	_ = v.BindEnv("Partitions", "PARTITIONS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

type handler struct {
	db     *database.Database
	output string
	topN   *middleware.TopNDisk
	joiner *middleware.JoinerDisk
}

func buildHandler(partition int) middleware.HandlerFunc[*handler] {
	return func(h *handler, ch *middleware.Channel, data []byte) error {
		return h.handlePartialResult(ch, data, partition)
	}
}

func (h *handler) handlePartialResult(ch *middleware.Channel, data []byte, partition int) error {
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

	partial, err := middleware.Deserialize[[]middleware.GameStat](data)
	if err != nil {
		return err
	}

	if h.joiner.Seen(partition) {
		if h.joiner.EOF() {
			ch.Finish()
		}
		return nil
	}
	err = h.joiner.Mark(snapshot, partition)
	for _, gStat := range partial {
		h.topN.Put(gStat)
	}

	err = h.topN.Save(snapshot)
	if err != nil {
		return err
	}

	utils.MaybeExit(0.001)

	if h.joiner.EOF() {
		log.Infof("Received all partial results")
		err = h.conclude(ch)
		utils.MaybeExit(0.50)
		return err
	}
	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	games := h.topN.Get()
	result := protocol.Q3Result{TopN: games}
	err := ch.SendAny(result, "", h.output)
	if err != nil {
		return err
	}

	ch.Finish()
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

	queues := make([]middleware.QueueConfig, 0)
	endpoints := make(map[string]middleware.HandlerFunc[*handler], 0)

	for i := 1; i <= cfg.Partitions; i++ {
		qName := middleware.Cat(middleware.PartialQ3, i)
		qcfg := middleware.QueueConfig{
			Name: qName,
		}
		queues = append(queues, qcfg)
		endpoints[qName] = buildHandler(i)
	}
	queues = append(queues, middleware.QueueConfig{Name: middleware.Results})

	err = middleware.Topology{
		Queues: queues,
	}.Declare(ch)

	if err != nil {
		utils.Expect(err, "Failed to declare topology")
	}

	nConfig := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			joiner := middleware.NewJoinerDisk("joiner", cfg.Partitions)
			err = joiner.Load(db)
			utils.Expect(err, "unrecoverable error")

			topN := middleware.NewTopNDisk("TopN", cfg.TopN)
			err = topN.LoadDisk(db)
			utils.Expect(err, "unrecoverable error")

			return &handler{
				db:     db,
				output: middleware.Results,
				joiner: joiner,
				topN:   topN,
			}
		},
		Endpoints: endpoints,
	}

	node, err := middleware.NewNode(nConfig, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
