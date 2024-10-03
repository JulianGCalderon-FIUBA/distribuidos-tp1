package middleware

import (
	"bytes"
	"encoding/gob"
)

type Score int8

const (
	PositiveScore Score = 1
	NegativeScore Score = -1
)

type Game struct {
	AppID                  uint64
	AveragePlaytimeForever uint64
	Windows                bool
	Mac                    bool
	Linux                  bool
	ReleaseYear            uint16
	Name                   string
	Genres                 []string
}

type Review struct {
	AppID uint64
	Score Score
	Text  string
}

type Batch[T any] []T

func Serialize(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(v)
	return buf.Bytes(), err
}

func Deserialize[T any](buf []byte) (T, error) {
	var v T
	err := DeserializeInto(buf, &v)
	return v, err
}

func DeserializeInto(buf []byte, v any) error {
	r := bytes.NewBuffer(buf)
	enc := gob.NewDecoder(r)
	return enc.Decode(v)
}