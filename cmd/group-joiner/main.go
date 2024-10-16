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
	output          string
	lastBatchId     int
	eofReceived     int
	totalPartitions int
	sequencer       map[int]*utils.Sequencer
}

func buildHandler(partition int) middleware.HandlerFunc[handler] {
	return func(h *handler, ch *middleware.Channel, data []byte) error {
		return h.handleBatch(ch, data, partition)
	}
}

func (h *handler) handleBatch(ch *middleware.Channel, data []byte, partition int) error {
	batch, err := middleware.Deserialize[middleware.Batch[middleware.GameStat]](data)
	if err != nil {
		return err
	}

	h.sequencer[partition].Mark(batch.BatchID, batch.EOF)

	b := middleware.Batch[middleware.GameStat]{
		Data:     batch.Data,
		ClientID: batch.ClientID,
		BatchID:  h.lastBatchId,
		EOF:      false,
	}

	err = ch.Send(b, "", h.output)
	h.lastBatchId += 1
	if err != nil {
		return err
	}

	if h.sequencer[partition].EOF() {
		log.Infof("Received EOF from partition %v", partition)
		h.eofReceived += 1
	}

	if h.eofReceived == h.totalPartitions {
		b := middleware.Batch[middleware.GameStat]{EOF: true}
		return ch.Send(b, "", h.output)
	}

	return nil
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	queues := make([]middleware.QueueConfig, 0)
	endpoints := make(map[string]middleware.HandlerFunc[handler], 0)

	for i := 1; i <= cfg.Partitions; i++ {
		qname := middleware.Cat(cfg.Input, i)
		qcfg := middleware.QueueConfig{
			Name: qname,
		}
		queues = append(queues, qcfg)
		endpoints[qname] = buildHandler(i)
	}

	output := middleware.QueueConfig{Name: cfg.Output}

	queues = append(queues, output)

	err = middleware.Topology{
		Queues: queues,
	}.Declare(ch)
	utils.Expect(err, "Failed to declare queues")

	nodeCfg := middleware.Config[handler]{
		Builder: func(clientID int) handler {
			sequencer := make(map[int]*utils.Sequencer)
			for i := 1; i <= cfg.Partitions; i++ {
				sequencer[i] = utils.NewSequencer()
			}
			return handler{
				output:          cfg.Output,
				lastBatchId:     0,
				eofReceived:     0,
				totalPartitions: cfg.Partitions,
				sequencer:       sequencer,
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
