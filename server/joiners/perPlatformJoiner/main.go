package main

import (
	"distribuidos/tp1/utils"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP         string
	PartitionsNumber int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("PartitionsNumber", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("PartitionsNumber", "PARTITIONS_NUMBER")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")

	r, err := newJoiner(cfg)
	utils.Expect(err, "Failed to create partitioner")

	err = r.run()
	utils.Expect(err, "Failed to run partitioner")
}
