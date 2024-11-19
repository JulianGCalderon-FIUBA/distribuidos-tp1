package main

import (
	"distribuidos/tp1/utils"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	Id          uint64
	Address     string
	NextAddress string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("Id", 0)
	v.SetDefault("Address", "127.0.0.1:9000")

	_ = v.BindEnv("Id", "ID")
	_ = v.BindEnv("Address", "ADDRESS")
	_ = v.BindEnv("NextAddress", "NEIGHBOR_ADDRESS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func main() {

	cfg, err := getConfig()
	utils.Expect(err, "Failed to get config")

	l := utils.NewLeaderElection(cfg.Id, cfg.Address, cfg.NextAddress)

	/* wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		err = l.Start()
		utils.Expect(err, "Failed to start leader election")
	}()
	wg.Wait()
	for {
		l.WaitLeader(true)
		log.Infof("I am leader (id %v) and I woke up", cfg.Id)
		// reiniciar nodos
		l.WaitLeader(false)
	} */

	err = l.Start()
	utils.Expect(err, "Failed to start leader election")

}
