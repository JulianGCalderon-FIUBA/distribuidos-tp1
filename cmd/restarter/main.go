package main

import (
	"context"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	Address string 
}

func getConfig() (config, error) {
	v := viper.New()

	_ = v.BindEnv("Address", "ADDRESS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	
	r, err := newRestarter(cfg)
	if err != nil {
		log.Fatalf("Failed to create restarter: %v", err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	err = r.start(ctx)
	if err != nil {
		log.Fatalf("Failed to run restarter: %v", err)
	}
}
