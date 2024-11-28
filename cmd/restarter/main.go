package main

import (
	"context"
	"distribuidos/tp1/restarter-protocol"
	"distribuidos/tp1/utils"
	"os/signal"
	"syscall"

	logging "github.com/op/go-logging"
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
	v.SetDefault("Replicas", 4)

	_ = v.BindEnv("Id", "ID")

	_ = v.BindEnv("Replicas", "REPLICAS")
	_ = v.BindEnv("Address", "ADDRESS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {

	cfg, err := getConfig()
	utils.Expect(err, "Failed to get config")

	r := restarter.NewRestarter(cfg.Address, cfg.Id, cfg.Replicas)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	go func() {
		err = r.Start(ctx)
		utils.Expect(err, "Failed to start leader election")
	}()
	go func() {
		for {
			r.WaitLeader(true)
			log.Infof("I am leader (id %v) and I woke up", cfg.Id)
			monitorCtx, cancelMonitor := context.WithCancel(ctx)
			r.StartMonitoring(monitorCtx)

			// stop monitoring
			r.WaitLeader(false)
			log.Infof("I am no longer leader (id %v)", cfg.Id)
			cancelMonitor()
			<-monitorCtx.Done()
		}
	}()
	<-ctx.Done()
}
