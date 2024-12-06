package main

import (
	"distribuidos/tp1/utils"
	"fmt"
	"math/rand"
	"os/exec"
	"time"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

type config struct {
	NodesPath string
	Period    int
	LogLevel  string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("LogLevel", logging.INFO.String())
	v.SetDefault("Period", 6)

	_ = v.BindEnv("NodesPath", "NODES_PATH")
	_ = v.BindEnv("LogLevel", "LOG_LEVEL")
	_ = v.BindEnv("Period", "PERIOD")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

var log = logging.MustGetLogger("log")

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to get config")

	err = utils.InitLogger(cfg.LogLevel)
	utils.Expect(err, "Failed to init logger")

	nodes, err := utils.ReadNodes(cfg.NodesPath)
	utils.Expect(err, "Failed to read nodes")

	for {
		i := rand.Intn(len(nodes))
		node := nodes[i]

		log.Infof("Killing %v", node)
		err = kill(node)
		if err != nil {
			log.Errorf("Failed to kill node %v: %v", node, err)
			continue
		}

		time.Sleep(time.Duration(cfg.Period) * time.Millisecond)
	}
}

func kill(node string) error {
	cmd := fmt.Sprintf("docker kill -s 9 %v", node)
	return exec.Command("/bin/sh", "-c", cmd).Run()
}
