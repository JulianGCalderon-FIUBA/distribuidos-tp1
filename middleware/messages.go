package middleware

import (
	"bytes"
	"encoding/gob"
)

type Output struct {
	Exchange string
	Keys     []string
}

const (
	NotClean int = 0
	CleanAll int = 1
	CleanId  int = 2
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

type GameStat struct {
	AppID uint64
	Name  string
	Stat  uint64
}

type Batch[T any] struct {
	Data    []T
	BatchID int
	EOF     bool
}

func Serialize(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(v)
	return buf.Bytes(), err
}

func SerializeAny(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&v)
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
