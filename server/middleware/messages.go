package middleware

import (
	"bytes"
	"distribuidos/tp1/protocol"
	"encoding/gob"
)

type Score int8

const (
	PositiveScore Score = 1
	NegativeScore Score = -1
)

const (
	IndieGenre  = "Indie"
	ActionGenre = "Action"
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

type Batch[T any] struct {
	Data     []T
	ClientID int
	BatchID  int
	EOF      bool
}

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

func DeserializeResults(buf []byte, msg any) error {
	r := bytes.NewBuffer(buf)
	enc := gob.NewDecoder(r)
	gob.Register(protocol.Q1Results{})
	gob.Register(protocol.Q2Results{})
	gob.Register(protocol.Q3Results{})
	gob.Register(protocol.Q4Results{})
	gob.Register(protocol.Q5Results{})
	return enc.Decode(msg)
}
