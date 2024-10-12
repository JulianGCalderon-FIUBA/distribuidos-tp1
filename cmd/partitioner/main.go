package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"os/signal"
	"strconv"
	"syscall"

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
		c.Output = middleware.Cat(c.Input, "x")
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

	filterCfg := middleware.FilterConfig{
		RabbitIP:    cfg.RabbitIP,
		Queue:       cfg.Input,
		Exchange:    cfg.Output,
		QueuesByKey: make(map[string][]string),
	}

	for i := 1; i <= cfg.Partitions; i++ {
		qName := fmt.Sprintf("%v-%v", cfg.Output, i)
		qKey := strconv.Itoa(i)
		qNames := filterCfg.QueuesByKey[qKey]
		qNames = append(qNames, qName)
		filterCfg.QueuesByKey[qKey] = qNames
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	switch cfg.Type {
	case GameDataType:
		h := gameHandler{
			partitionsNumber: cfg.Partitions,
		}
		n, err := middleware.NewFilter(filterCfg, h.Filter)
		utils.Expect(err, "Failed to create partitioner")
		err = n.Run(ctx)
		utils.Expect(err, "Failed to run partitioner")
	case ReviewDataType:
		h := reviewHandler{
			partitionsNumber: cfg.Partitions,
		}
		n, err := middleware.NewFilter(filterCfg, h.Filter)
		utils.Expect(err, "Failed to create partitioner")
		err = n.Run(ctx)
		utils.Expect(err, "Failed to run partitioner")
	}

}
