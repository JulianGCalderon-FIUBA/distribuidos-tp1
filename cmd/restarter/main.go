package main

import (
	"context"
	"distribuidos/tp1/restarter-protocol"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	Id       uint64
	Address  string
	Replicas uint64
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("Id", 0)
	v.SetDefault("Address", "127.0.0.1:9000")
	v.SetDefault("Replicas", 4)

	_ = v.BindEnv("Id", "ID")
	_ = v.BindEnv("Address", "ADDRESS")
	_ = v.BindEnv("Replicas", "REPLICAS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {

	cfg, err := getConfig()
	utils.Expect(err, "Failed to get config")

	l := restarter.NewLeaderElection(cfg.Id, cfg.Address, cfg.Replicas)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	go func() {
		err = l.Start(ctx)
		utils.Expect(err, "Failed to start leader election")
	}()
	go func() {
		for {
			l.WaitLeader(true)
			log.Infof("I am leader (id %v) and I woke up", cfg.Id)
			// start reiniciar en go rutinas
			l.WaitLeader(false)
			log.Infof("I am no longer leader (id %v)", cfg.Id)
			// frenar trabajo -> shutdown a las go rutinas
		}
	}()

	<-ctx.Done()
}
