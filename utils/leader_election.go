package utils

import (
	"encoding/binary"
	"slices"
)

type LeaderElection struct {
	id        uint64
	leader_id uint64
}

func NewLeaderElection(id uint64) *LeaderElection {
	return &LeaderElection{
		id:        id,
		leader_id: id,
	}
}

func (l *LeaderElection) AmILeader() bool {
	return l.id == l.leader_id
}

func (l *LeaderElection) HandleElection(msg []byte) ([]byte, error) {

	ids, err := decodeIds(msg)
	if err != nil {
		return []byte{}, err
	}
	if slices.Contains(ids, l.id) {
		return l.StartCoordinator(ids)
	}
	ids = append(ids, l.id)

	return encodeElection(ids)
}

func (l *LeaderElection) StartCoordinator(ids []uint64) ([]byte, error) {

	leader := slices.Max(ids)
	l.leader_id = leader

	return encodeCoordinator(leader, []uint64{l.id})
}

func (l *LeaderElection) HandleCoordinator(msg []byte) ([]byte, error) {

	leader, ids, err := decodeCoordinator(msg)
	if err != nil {
		return []byte{}, err
	}

	if slices.Contains(ids, l.id) {
		return []byte{}, nil
	}

	l.leader_id = leader
	ids = append(ids, l.id)

	return encodeCoordinator(leader, ids)
}

func encodeCoordinator(leader uint64, ids []uint64) ([]byte, error) {
	buf, err := binary.Append(nil, binary.LittleEndian, 'C')
	if err != nil {
		return []byte{}, err
	}
	buf, err = binary.Append(buf, binary.LittleEndian, leader)
	if err != nil {
		return []byte{}, err
	}

	msg, err := encodeIds(ids, buf)
	if err != nil {
		return []byte{}, err
	}
	return msg, nil
}

func encodeElection(ids []uint64) ([]byte, error) {
	buf, err := binary.Append(nil, binary.LittleEndian, 'E')
	if err != nil {
		return []byte{}, err
	}

	return encodeIds(ids, buf)
}

func encodeIds(ids []uint64, buf []byte) ([]byte, error) {
	seen := uint64(len(ids))
	buf, err := binary.Append(buf, binary.LittleEndian, seen)
	if err != nil {
		return []byte{}, err
	}

	for id := range ids {
		buf, err = binary.Append(buf, binary.LittleEndian, id)
		if err != nil {
			return []byte{}, err
		}
	}
	return buf, nil
}

func decodeIds(msg []byte) ([]uint64, error) {
	ids := make([]uint64, 0)
	var seen uint64
	n, err := binary.Decode(msg, binary.LittleEndian, seen)
	if err != nil {
		return []uint64{}, err
	}

	msg = msg[n:]
	for i := uint64(0); i < seen; i++ {
		var id uint64
		n, err := binary.Decode(msg, binary.LittleEndian, id)
		if err != nil {
			return []uint64{}, err
		}
		ids = append(ids, id)
		msg = msg[n:]
	}

	return ids, nil
}

func decodeCoordinator(msg []byte) (uint64, []uint64, error) {
	var leader uint64
	n, err := binary.Decode(msg, binary.LittleEndian, leader)
	if err != nil {
		return 0, []uint64{}, err
	}
	msg = msg[n:]

	ids, err := decodeIds(msg)
	if err != nil {
		return 0, []uint64{}, err
	}

	return leader, ids, err
}
