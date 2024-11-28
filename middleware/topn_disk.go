package middleware

import (
	"container/heap"
	"distribuidos/tp1/database"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"slices"
	"sort"
)

type TopNDisk struct {
	top  GameHeap
	name string
	n    int
}

func NewTopNDisk(name string, n int) *TopNDisk {
	return &TopNDisk{
		top:  make(GameHeap, 0, n),
		name: name,
		n:    n,
	}
}

func (t *TopNDisk) LoadDisk(snapshot *database.Database) error {
	file, err := snapshot.Get(t.name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	t.top = t.top[:0]
	for {
		var header struct {
			AppId    uint64
			Stat     uint64
			NameSize uint64
		}
		err = binary.Read(file, binary.LittleEndian, &header)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		name := make([]byte, header.NameSize)
		_, err := file.Read(name)
		if err != nil {
			return err
		}

		game := GameStat{
			AppID: header.AppId,
			Stat:  header.Stat,
			Name:  string(name),
		}

		t.top = append(t.top, game)
	}
}

func (t *TopNDisk) Put(g GameStat) {
	if t.top.Len() < t.n {
		heap.Push(&t.top, g)
	} else if g.Stat > t.top.Peek().Stat {
		heap.Pop(&t.top)
		heap.Push(&t.top, g)
	}
}

func (t *TopNDisk) Save(snapshot *database.Snapshot) error {
	file, err := snapshot.Create(t.name)
	if err != nil {
		return err
	}

	for _, g := range t.top {
		header := struct {
			AppId    uint64
			Stat     uint64
			NameSize uint64
		}{
			AppId:    g.AppID,
			Stat:     g.Stat,
			NameSize: uint64(len(g.Name)),
		}
		err = binary.Write(file, binary.LittleEndian, header)
		if err != nil {
			return err
		}
		_, err := file.WriteString(g.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TopNDisk) Get() []GameStat {
	var games []GameStat = slices.Clone(t.top)

	sort.Slice(games, func(i, j int) bool {
		return games[i].Stat > games[j].Stat
	})

	return games
}
