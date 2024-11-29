package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

const NODE_PORT = 7000
const RESTARTER_PORT = 14300
const NODE_UDP_ADDR = "0.0.0.0:7000"

// Fails if error is not nil, used for functions that
// cannot fail (ej: rabbit communication)
func Expect(err any, msg string) {
	if err != nil {
		log.Fatalf("%v: %v", msg, err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetUDPAddr(host string, port int) (*net.UDPAddr, error) {
	addr := fmt.Sprintf("%v:%v", host, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return udpAddr, nil
}

func OpenFileAll(name string, flag int, perm os.FileMode) (*os.File, error) {
	file, err := os.OpenFile(name, flag, perm)
	if errors.Is(err, fs.ErrNotExist) {
		err = os.MkdirAll(path.Dir(name), 0750)
		if err != nil {
			return nil, err
		}

		file, err = os.OpenFile(name, flag, perm)
	}
	return file, err
}
