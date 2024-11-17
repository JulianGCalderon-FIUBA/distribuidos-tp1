package main

import (
	"distribuidos/tp1/utils"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")
var nextAddr, _ = net.ResolveUDPAddr("udp", "127.0.0.1:9001")

type config struct {
	Id      uint64
	Address string
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("Id", 0)
	v.SetDefault("Address", "127.0.0.1:9000")

	_ = v.BindEnv("Id", "ID")
	_ = v.BindEnv("Address", "ADDRESS")

	var c config
	err := v.Unmarshal(&c)
	return c, err
}

func send(conn *net.UDPConn, msg []byte) error {
	n, err := conn.WriteTo(msg, nextAddr)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Could not send full message")
	}
	return nil
}

func main() {

	c, err := getConfig()
	utils.Expect(err, "Failed to get config")

	udpAddr, err := net.ResolveUDPAddr("udp", c.Address)
	if err != nil {
		utils.Expect(err, "Did not receive a valid udp address")
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		utils.Expect(err, "Failed to start listening from connection")
	}

	l := utils.NewLeaderElection(c.Id)

	for {
		var msg []byte
		var msgType rune

		_, err := conn.Read(msg)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
		}
		n, err := binary.Decode(msg, binary.LittleEndian, msgType)
		if err != nil {
			log.Errorf("Failed to decode message type: %v", err)
			continue
		}
		msg = msg[n:]

		switch msgType {
		case 'A':
			// handle ack
		case 'C':
			m, err := l.HandleCoordinator(msg)
			if err != nil {
				log.Errorf("Failed to handle coordinator message: %v", err)
				continue
			}
			if len(m) > 0 {
				err := send(conn, m)
				if err != nil {
					log.Errorf("Failed to send coordinator message: %v", err)
				}
			}
			// send ack
		case 'E':
			m, err := l.HandleElection(msg)
			if err != nil {
				log.Errorf("Failed to handle election message: %v", err)
				continue
			}
			err = send(conn, m)
			if err != nil {
				log.Errorf("Failed to send election message: %v", err)
			}
			// send ack
		}
	}
}
