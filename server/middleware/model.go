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
	Score Score
	Text  string
}

type BatchGame struct {
	Data []Game
}

type BatchReview struct {
	Data []Review
}

type Message interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) (int, error)
}

func (g *Game) Serialize() (buf []byte, err error) {
	inner := struct {
		AppID                  uint64
		AveragePlaytimeForever uint64
		Windows                bool
		Mac                    bool
		Linux                  bool
		ReleaseDate            Date
	}{
		AppID:                  g.AppID,
		AveragePlaytimeForever: g.AveragePlaytimeForever,
		Windows:                g.Windows,
		Mac:                    g.Mac,
		Linux:                  g.Linux,
		ReleaseDate:            g.ReleaseDate,
	}
	buf, err = binary.Append(nil, binary.LittleEndian, &inner)
	if err != nil {
		return
	}

	buf, err = serializeString(buf, g.Name)
	if err != nil {
		return
	}

	buf, err = binary.Append(buf, binary.LittleEndian, uint32(len(g.Genres)))
	if err != nil {
		return
	}
	for _, s := range g.Genres {
		buf, err = serializeString(buf, s)
		if err != nil {
			return
		}
	}

	return
}

func (g *Game) Deserialize(buf []byte) (int, error) {
	var totalBytes int

	inner := struct {
		AppID                  uint64
		AveragePlaytimeForever uint64
		Windows                bool
		Mac                    bool
		Linux                  bool
		ReleaseDate            Date
	}{}
	n, err := binary.Decode(buf, binary.LittleEndian, &inner)
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}
	buf = buf[n:]

	g.AppID = inner.AppID
	g.AveragePlaytimeForever = inner.AveragePlaytimeForever
	g.Windows = inner.Windows
	g.Mac = inner.Mac
	g.Linux = inner.Linux
	g.ReleaseDate = inner.ReleaseDate

	n, err = deserializeString(buf, &g.Name)
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}
	buf = buf[n:]

	var length uint32
	n, err = binary.Decode(buf, binary.LittleEndian, &length)
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}
	buf = buf[n:]

	g.Genres = make([]string, length)
	for i := 0; i < int(length); i++ {
		n, err = deserializeString(buf, &g.Genres[i])
		totalBytes += n
		if err != nil {
			return totalBytes, err
		}
		buf = buf[n:]
	}

	return totalBytes, nil
}

func (r *Review) Serialize() (buf []byte, err error) {
	inner := struct {
		AppID uint64
		Score Score
	}{
		AppID: r.AppID,
		Score: r.Score,
	}
	buf, err = binary.Append(nil, binary.LittleEndian, &inner)
	if err != nil {
		return
	}
	buf, err = serializeString(buf, r.Text)
	return
}

func (r *Review) Deserialize(buf []byte) (int, error) {
	var totalBytes int

	inner := struct {
		AppID uint64
		Score Score
	}{}
	n, err := binary.Decode(buf, binary.LittleEndian, &inner)
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}
	buf = buf[n:]

	r.AppID = inner.AppID
	r.Score = inner.Score

	n, err = deserializeString(buf, &r.Text)
	totalBytes += n
	return totalBytes, err
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

func (b *BatchGame) Serialize() (buf []byte, err error) {
	buf, err = binary.Append(nil, binary.LittleEndian, uint32(len(b.Data)))
	if err != nil {
		return
	}
	for _, msg := range b.Data {
		serialized, err := msg.Serialize()
		if err != nil {
			return buf, err
		}
		buf = append(buf, serialized...)
	}
	return buf, nil
}

func (b *BatchGame) Deserialize(buf []byte) (int, error) {
	var length uint32
	var totalBytes int

	println(buf)
	n, err := binary.Decode(buf, binary.LittleEndian, &length)
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}
	buf = buf[n:]
	b.Data = make([]Game, length)

	for i := 0; i < int(length); i++ {
		n, err := (&b.Data[i]).Deserialize(buf)
		totalBytes += n
		if err != nil {
			return totalBytes, err
		}
		buf = buf[n:]
	}
	return totalBytes, nil
}

func (b *BatchReview) Serialize() (buf []byte, err error) {
	buf, err = binary.Append(nil, binary.LittleEndian, uint32(len(b.Data)))
	if err != nil {
		return
	}
	for _, msg := range b.Data {
		serialized, err := msg.Serialize()
		if err != nil {
			return buf, err
		}
		buf = append(buf, serialized...)
	}
	return buf, nil
}

func (b *BatchReview) Deserialize(buf []byte) (int, error) {
	var length uint32
	var totalBytes int

	println(buf)
	n, err := binary.Decode(buf, binary.LittleEndian, &length)
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}
	buf = buf[n:]
	b.Data = make([]Review, length)

	for i := 0; i < int(length); i++ {
		n, err := (&b.Data[i]).Deserialize(buf)
		totalBytes += n
		if err != nil {
			return totalBytes, err
		}
		buf = buf[n:]
	}
	return totalBytes, nil
}
