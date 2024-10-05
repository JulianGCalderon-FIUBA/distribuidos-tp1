package utils

import "github.com/op/go-logging"

var log = logging.MustGetLogger("log")

// Fails if error is not nil, used for functions that
// cannot fail (ej: rabbit communication)
func Expect(err any, msg string) {
	if err != nil {
		log.Fatalf("%v: %v", msg, err)
	}
}
