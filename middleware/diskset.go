package middleware

import (
	"bufio"
	"distribuidos/tp1/database"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
)

type DiskSet struct {
	name string
	ids  map[int]struct{}
}

func NewSetDisk(name string) *DiskSet {
	return &DiskSet{
		name: name,
		ids:  make(map[int]struct{}),
	}
}

func (s *DiskSet) Mark(id int) {
	s.ids[id] = struct{}{}
}

func (s *DiskSet) Seen(id int) bool {
	_, ok := s.ids[id]
	return ok
}

func (s *DiskSet) MarkDisk(snapshot *database.Snapshot, id int) error {
	file, err := snapshot.Append(s.name)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.LittleEndian, uint64(id))
	if err != nil {
		return err
	}

	s.Mark(id)
	return nil
}

func (s *DiskSet) LoadDisk(db *database.Database) error {
	file, err := db.Get(s.name)
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	for {
		var id uint64
		err = binary.Read(reader, binary.LittleEndian, &id)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		s.Mark(int(id))
	}

	return nil
}
