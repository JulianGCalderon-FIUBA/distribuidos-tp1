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

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	Partitions int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Partitions", "PARTITIONS")

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
	db         *database.Database
	output     string
	count      map[Platform]int
	partitions map[int]struct{}
}

func buildHandler(partition int) middleware.HandlerFunc[*handler] {
	return func(h *handler, ch *middleware.Channel, data []byte) error {
		return h.handlePartialResult(ch, data, partition)
	}
}

func (h *handler) handlePartialResult(ch *middleware.Channel, data []byte, partition int) error {
	c, err := middleware.Deserialize[map[Platform]int](data)
	log.Infof("SEEN %v!!\n", partition)
	if _, notSeen := h.partitions[partition]; !notSeen {
		return nil
	}

	// save this on disk also
	delete(h.partitions, partition)

	if err != nil {
		return err
	}

	for k, v := range c {
		h.count[k] += v
	}

	if len(h.partitions) == 0 {
		return h.conclude(ch)
	}

	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	for k, v := range h.count {
		log.Infof("Found %v games with %v support", v, string(k))
	}

	result := protocol.Q1Result{
		Windows: h.count[Windows],
		Linux:   h.count[Linux],
		Mac:     h.count[Mac],
	}

	err := ch.SendAny(result, "", h.output)
	if err != nil {
		return err
	}

	ch.Finish()
	return nil
}

func (h *handler) Free() error {
	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q1Result{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	queues := make([]middleware.QueueConfig, 0)
	endpoints := make(map[string]middleware.HandlerFunc[*handler], 0)

	for i := 1; i <= cfg.Partitions; i++ {
		qName := middleware.Cat(middleware.PartialQ1, i)
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
		utils.Expect(err, "Failed to initialize topology")
	}

	nConfig := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			partitions := make(map[int]struct{})
			for i := 1; i <= cfg.Partitions; i++ {
				partitions[i] = struct{}{}
			}

			return &handler{
				db:         db,
				output:     middleware.Results,
				count:      make(map[Platform]int, 0),
				partitions: partitions,
			}
		},
		Endpoints: endpoints,
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	node, err := middleware.NewNode(nConfig, conn)
	if err != nil {
		utils.Expect(err, "Failed to start joiner node")
	}

	err = node.Run(ctx)
	if err != nil {
		utils.Expect(err, "Failed to run joiner node")
	}
}
