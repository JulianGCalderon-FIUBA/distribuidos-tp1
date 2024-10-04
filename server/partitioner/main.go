package main

import (
	"distribuidos/tp1/server/middleware"
	"errors"
	"fmt"
	"strconv"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP         string
	InputQueue       string
	OutputExchange   string
	PartitionsNumber int
	Type             string
}

const GameType string = "game"
const ReviewType string = "review"

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionsNumber", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionsNumber", "PARTITIONS_NUMBER")
	_ = v.BindEnv("InputQueue", "INPUT_QUEUE")
	_ = v.BindEnv("Type", "TYPE")

	var c config
	err := v.Unmarshal(&c)

	if c.InputQueue == "" {
		errors.New("InputQueue should not be empty")
	}
	if c.Type != GameType && c.Type != ReviewType {
		errors.New("Type should not be game or review")
	}
	if c.OutputExchange == "" {
		c.OutputExchange = fmt.Sprintf("%v-x", c.InputQueue)
	}

	return c, err
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	m, err := middleware.NewMiddleware(cfg.RabbitIP)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	err = m.InitPartitioner(cfg.InputQueue, cfg.OutputExchange, cfg.PartitionsNumber)
	if err != nil {
		log.Fatalf("Failed to init partition: %v", err)
	}
	log.Infof("Initialized partitioner infrastructure")

	dch, err := m.Subscribe(cfg.InputQueue)
	if err != nil {
		log.Fatalf("Failed to subscribe to input queue: %v", err)
	}

	for d := range dch {
		batch, err := middleware.Deserialize[middleware.Batch[middleware.Game]](d.Body)
		if err != nil {
			log.Errorf("Failed to deserialize batch %v", err)
			d.Nack(false, false)
			continue
		}

		partitions := make([]middleware.Batch[middleware.Game], cfg.PartitionsNumber)
		for _, game := range batch {
			partitionId := game.AppID % uint64(cfg.PartitionsNumber)
			partitions[partitionId] = append(partitions[partitionId], game)
		}

		for partitionId, partition := range partitions {
			err = m.Send(partition, cfg.OutputExchange, strconv.Itoa(partitionId))
			if err != nil {
				log.Errorf("Failed to send batch: %v", err)
				continue
			}
		}

		d.Ack(false)
	}
}
