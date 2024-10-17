package main

import (
	"context"
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
	output     string
	count      map[Platform]int
	partitions int
}

func (h *handler) handlePartialResult(ch *middleware.Channel, data []byte) error {
	c, err := middleware.Deserialize[map[Platform]int](data)
	h.partitions--

	if err != nil {
		return err
	}

	for k, v := range c {
		h.count[k] += v
	}

	if h.partitions == 0 {
		return h.conclude(ch)
	}

	return nil
}

func (h *handler) conclude(ch *middleware.Channel) error {
	for k, v := range h.count {
		log.Infof("Found %v games with %v support", v, string(k))
	}

	result := protocol.Q1Results{
		Windows: h.count[Windows],
		Linux:   h.count[Linux],
		Mac:     h.count[Mac],
	}

	return ch.SendAny(result, "", middleware.Results)
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q1Results{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	qInput := middleware.PartialQ1
	err = middleware.Topology{
		Queues: []middleware.QueueConfig{
			{Name: qInput},
			{Name: middleware.Results},
		},
	}.Declare(ch)

	if err != nil {
		utils.Expect(err, "Failed to initialize topology")
	}

	nConfig := middleware.Config[handler]{
		Builder: func(clientID int) handler {
			return handler{
				output:     middleware.Results,
				count:      make(map[Platform]int, 0),
				partitions: cfg.Partitions,
			}
		},
		Endpoints: map[string]middleware.HandlerFunc[handler]{
			qInput: (*handler).handlePartialResult,
		},
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

	log.Infof("top-n-historic-avg joiner started")
}
