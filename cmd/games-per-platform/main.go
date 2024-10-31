package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
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
	output    string
	count     map[Platform]int
	sequencer *utils.Sequencer
}

func (h *handler) handleGame(ch *middleware.Channel, data []byte) error {

	batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](data)
	if err != nil {
		return err
	}

	h.sequencer.Mark(batch.BatchID, batch.EOF)

	for _, g := range batch.Data {
		if g.Windows {
			h.count[Windows] += 1
		}
		if g.Linux {
			h.count[Linux] += 1
		}
		if g.Mac {
			h.count[Mac] += 1
		}
	}

	if h.sequencer.EOF() {
		for k, v := range h.count {
			log.Infof("Found %v games with %v support", v, string(k))
		}

		err := ch.Send(h.count, "", h.output)
		if err != nil {
			return err
		}

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

	qName := middleware.Cat(middleware.GamesQ1, "x", cfg.PartitionID)
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qName},
			{Name: middleware.PartialQ1},
		},
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			return &handler{
				count:     make(map[Platform]int),
				output:    middleware.PartialQ1,
				sequencer: utils.NewSequencer(),
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[*handler]{
			qName: (*handler).handleGame,
		},
	}

	node, err := middleware.NewNode(nodeCfg, conn)
	utils.Expect(err, "Failed to create node")

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = node.Run(ctx)
	utils.Expect(err, "Failed to run node")
}
