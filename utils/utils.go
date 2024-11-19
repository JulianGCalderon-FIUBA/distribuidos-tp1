package utils

import (
	"net"
	"strconv"
	"strings"

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

func GetUDPAddr(address string) *net.UDPAddr {
	addr := strings.Split(address, ":")
	port, err := strconv.Atoi(addr[1])
	if err != nil {
		Expect(err, "Did not receive a valid address")
	}

	return &net.UDPAddr{
		IP:   net.ParseIP(addr[0]),
		Port: port,
	}
}
