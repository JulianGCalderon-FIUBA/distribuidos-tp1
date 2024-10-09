package main

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/filter"
	"distribuidos/tp1/server/middleware/node"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

type config struct {
	RabbitIP   string
	Input      string
	Output     string
	Partitions int
	Type       DataType
}

type DataType string

const (
	GameDataType   DataType = "game"
	ReviewDataType DataType = "review"
)

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Partitions", "PARTITIONS")
	_ = v.BindEnv("Input", "INPUT")
	_ = v.BindEnv("Output", "OUTPUT")
	_ = v.BindEnv("Type", "TYPE")

	var c config
	err := v.Unmarshal(&c)

	if c.Input == "" {
		return c, errors.New("InputQueue should not be empty")
	}
	if c.Type != GameDataType && c.Type != ReviewDataType {
		return c, fmt.Errorf("Type should be one of: [%v, %v]", string(GameDataType), string(ReviewDataType))
	}
	if c.Output == "" {
		c.Output = fmt.Sprintf("%v-x", c.Input)
	}

	return c, err
}

type gameHandler struct {
	partitionsNumber int
}

func (h gameHandler) Filter(g middleware.Game) []string {
	return []string{strconv.Itoa(int(g.AppID)%h.partitionsNumber + 1)}
}

type reviewHandler struct {
	partitionsNumber int
}

func (h reviewHandler) Filter(r middleware.Review) []string {
	return []string{strconv.Itoa(int(r.AppID)%h.partitionsNumber + 1)}
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	filterCfg := filter.Config{
		RabbitIP: cfg.RabbitIP,
		Queue:    cfg.Input,
		Exchange: node.ExchangeConfig{
			Name:        cfg.Output,
			Type:        amqp.ExchangeDirect,
			QueuesByKey: map[string][]string{},
		},
	}

	for i := 1; i <= cfg.Partitions; i++ {
		qName := fmt.Sprintf("%v-%v", cfg.Output, i)
		qKey := strconv.Itoa(i)
		qNames := filterCfg.Exchange.QueuesByKey[qKey]
		qNames = append(qNames, qName)
		filterCfg.Exchange.QueuesByKey[qKey] = qNames
	}

	var f *filter.Filter

	switch cfg.Type {
	case GameDataType:
		h := gameHandler{
			partitionsNumber: cfg.Partitions,
		}
		f, err = filter.NewFilter(filterCfg, h)
	case ReviewDataType:
		h := reviewHandler{
			partitionsNumber: cfg.Partitions,
		}
		f, err = filter.NewFilter(filterCfg, h)
	}

	utils.Expect(err, "Failed to create partitioner")
	err = f.Run(context.Background())
	utils.Expect(err, "Failed to run partitioner")

}
