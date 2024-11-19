package utils

import (
	"fmt"
	"net"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// Fails if error is not nil, used for functions that
// cannot fail (ej: rabbit communication)
func Expect(err any, msg string) {
	if err != nil {
		log.Fatalf("%v: %v", msg, err)
	}
}

func GetUDPAddr(id uint64) (*net.UDPAddr, error) {
	addr := fmt.Sprintf("restarter-%v:1430%v", id, id)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return udpAddr, nil
}
