package middleware

import (
	"bufio"
	"distribuidos/tp1/database"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
)

type JoinerDisk struct {
	name string
	seen map[int]struct{}
	size int
}

func NewJoinerDisk(name string, size int) *JoinerDisk {
	return &JoinerDisk{
		name: name,
		seen: make(map[int]struct{}),
		size: size,
	}
}

func (j *JoinerDisk) Mark(snapshot *database.Snapshot, id int) error {
	file, err := snapshot.Append(j.name)
	if err != nil {
		return err
	}

	err = binary.Write(file, binary.LittleEndian, uint64(id))
	if err != nil {
		return err
	}

	j.seen[id] = struct{}{}

	return nil
}

func (j *JoinerDisk) Seen(id int) bool {
	_, seen := j.seen[id]
	return seen
}

func (j *JoinerDisk) EOF() bool {
	return len(j.seen) == j.size
}

func (j *JoinerDisk) Load(db *database.Database) error {
	file, err := db.Get(j.name)
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

		j.seen[int(id)] = struct{}{}
	}

	return nil
}
