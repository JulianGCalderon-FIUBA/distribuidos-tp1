package middleware

type Sequencer struct {
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
	return id <= s.latestID && !missing
}
