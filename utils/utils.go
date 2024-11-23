package utils

import (
	"os"

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
