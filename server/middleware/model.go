package middleware

import (
	"encoding/binary"
	"slices"
)

type Score int8

const (
	PositiveScore Score = 1
	NegativeScore Score = -1
)

type Date struct {
	Day   uint8
	Month uint8
	Year  uint16
}

type Game struct {
	AppID                  uint64
	AveragePlaytimeForever uint64
	Windows                bool
	Mac                    bool
	Linux                  bool
	ReleaseDate            Date
	Name                   string
	Genres                 []string
}

type Review struct {
	AppID uint64
	Text  string
	Score Score
}

func (b *Game) Serialize() (buf []byte, err error) {
	inner := struct {
		AppID                  uint64
		AveragePlaytimeForever uint64
		Windows                bool
		Mac                    bool
		Linux                  bool
		ReleaseDate            Date
	}{
		AppID:                  b.AppID,
		AveragePlaytimeForever: b.AveragePlaytimeForever,
		Windows:                b.Windows,
		Mac:                    b.Mac,
		Linux:                  b.Linux,
		ReleaseDate:            b.ReleaseDate,
	}
	buf, err = binary.Append(nil, binary.LittleEndian, &inner)
	if err != nil {
		return
	}

	buf, err = serializeString(buf, b.Name)
	if err != nil {
		return
	}

	buf, err = binary.Append(buf, binary.LittleEndian, uint32(len(b.Genres)))
	if err != nil {
		return
	}
	for _, s := range b.Genres {
		buf, err = serializeString(buf, s)
		if err != nil {
			return
		}
	}

	return
}

func (g *Game) Deserialize(buf []byte) error {
	inner := struct {
		AppID                  uint64
		AveragePlaytimeForever uint64
		Windows                bool
		Mac                    bool
		Linux                  bool
		ReleaseDate            Date
	}{}
	n, err := binary.Decode(buf, binary.LittleEndian, &inner)
	if err != nil {
		return err
	}
	buf = buf[n:]

	g.AppID = inner.AppID
	g.AveragePlaytimeForever = inner.AveragePlaytimeForever
	g.Windows = inner.Windows
	g.Mac = inner.Mac
	g.Linux = inner.Linux
	g.ReleaseDate = inner.ReleaseDate

	n, err = deserializeString(buf, &g.Name)
	if err != nil {
		return err
	}
	buf = buf[n:]

	var length uint32
	n, err = binary.Decode(buf, binary.LittleEndian, &length)
	if err != nil {
		return err
	}
	buf = buf[n:]

	g.Genres = make([]string, length)
	for i := 0; i < int(length); i++ {
		n, err = deserializeString(buf, &g.Genres[i])
		if err != nil {
			return err
		}
		buf = buf[n:]
	}

	return nil
}

func serializeString(buf []byte, s string) ([]byte, error) {
	buf, err := binary.Append(buf, binary.LittleEndian, uint32(len(s)))
	if err != nil {
		return buf, err
	}

	buf = append(buf, s...)

	return buf, nil
}

func deserializeString(buf []byte, s *string) (int, error) {
	var length uint32
	n, err := binary.Decode(buf, binary.LittleEndian, &length)
	if err != nil {
		return n, err
	}
	buf = buf[n:]

	copied := slices.Clone(buf[:length])
	*s = string(copied)

	return n + int(length), nil
}
