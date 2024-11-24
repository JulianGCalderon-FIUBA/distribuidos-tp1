package middleware

import (
	"bufio"
	"distribuidos/tp1/database"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
)

type Sequencer struct {
	name       string
	missingIDs map[int]struct{}
	latestID   int
	fakeEOF    bool
}

func NewSequencer() *Sequencer {
	return &Sequencer{
		missingIDs: make(map[int]struct{}),
		latestID:   -1,
		fakeEOF:    false,
	}
}

// should eventually replace `NewSequencer`
func NewSequencerDisk(name string) *Sequencer {
	return &Sequencer{
		name:       name,
		missingIDs: make(map[int]struct{}),
		latestID:   -1,
		fakeEOF:    false,
	}
}

func (s *Sequencer) Mark(id int, EOF bool) {
	delete(s.missingIDs, id)

	for i := s.latestID + 1; i < id; i++ {
		s.missingIDs[i] = struct{}{}
	}
	s.latestID = max(id, s.latestID)

	if EOF {
		s.fakeEOF = true
	}
}

func (s *Sequencer) EOF() bool {
	return s.fakeEOF && len(s.missingIDs) == 0
}

func (s *Sequencer) Seen(id int) bool {
	_, missing := s.missingIDs[id]
	return id < s.latestID && !missing
}

func (s *Sequencer) MarkDisk(snapshot *database.Snapshot, id int, EOF bool) error {
	if EOF {
		_, err := snapshot.Create(fmt.Sprintf("%v-EOF", s.name))
		if err != nil {
			return err
		}
	}

	file, err := snapshot.Append(s.name)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.LittleEndian, uint64(id))
	if err != nil {
		return err
	}

	s.Mark(id, EOF)

	return nil
}

func (s *Sequencer) LoadDisk(db *database.Database) error {
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

		s.Mark(int(id), false)
	}

	exists, err := db.Exists(fmt.Sprintf("%v-EOF", s.name))
	if err != nil {
		return err
	}
	if exists {
		s.fakeEOF = true
	}

	return nil
}
