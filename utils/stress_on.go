//go:build stress

package utils

import (
	"math/rand"
	"os"
)

func MaybeExit(p float32) {
	if rand.Float32() < p {
		os.Exit(1)
	}
}
