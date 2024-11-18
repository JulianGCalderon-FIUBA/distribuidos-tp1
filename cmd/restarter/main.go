package main

import (
	"distribuidos/tp1/utils"
	"net"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")
var nextAddr, _ = net.ResolveUDPAddr("udp", "127.0.0.1:9001")

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

	c, err := getConfig()
	utils.Expect(err, "Failed to get config")

	l := utils.NewLeaderElection(c.Id, c.Address, c.NextAddress)

	err = l.Start()
	utils.Expect(err, "Failed to start leader election")

}
