package leaderelection

import "encoding/binary"

type Message interface {
	Encode() ([]byte, error)
	Decode(msg []byte) error
	Type() uint8
}

type ElectionMsg struct {
	Ids []uint64
}

type CoordinatorMsg struct {
	Leader uint64
	Ids    []uint64
}

type AckMsg struct {
}

type MsgType uint8

const (
	Ack         MsgType = 'A'
	Coordinator MsgType = 'C'
	Election    MsgType = 'E'
	KeepAlive   MsgType = 'K'
)

type MsgHeader struct {
	Ty MsgType
	Id uint64
}

func (em ElectionMsg) Encode() ([]byte, error) { return encodeIds(em.Ids, nil) }
func (cm CoordinatorMsg) Encode() ([]byte, error) {
	buf, err := binary.Append(nil, binary.LittleEndian, cm.Leader)
	if err != nil {
		return []byte{}, err
	}

	msg, err := encodeIds(cm.Ids, buf)
	if err != nil {
		return []byte{}, err
	}
	return msg, nil
}
func (a AckMsg) Encode() ([]byte, error) { return []byte{}, nil }

func (em *ElectionMsg) Decode(msg []byte) error {
	var err error
	em.Ids, err = decodeIds(msg)
	return err
}
func (cm *CoordinatorMsg) Decode(msg []byte) error {
	n, err := binary.Decode(msg, binary.LittleEndian, &cm.Leader)
	if err != nil {
		return err
	}
	msg = msg[n:]

	cm.Ids, err = decodeIds(msg)
	return err
}
func (a AckMsg) Decode(msg []byte) error { return nil }

func (em ElectionMsg) Type() uint8    { return 'E' }
func (cm CoordinatorMsg) Type() uint8 { return 'C' }
func (a AckMsg) Type() uint8          { return 'A' }

func encodeIds(ids []uint64, buf []byte) ([]byte, error) {
	seen := uint64(len(ids))
	buf, err := binary.Append(buf, binary.LittleEndian, seen)
	if err != nil {
		return []byte{}, err
	}

	for i := 0; i < len(ids); i++ {
		buf, err = binary.Append(buf, binary.LittleEndian, uint64(ids[i]))
		if err != nil {
			return []byte{}, err
		}
	}
	return buf, nil
}

func decodeIds(msg []byte) ([]uint64, error) {
	var seen uint64
	n, err := binary.Decode(msg, binary.LittleEndian, &seen)
	if err != nil {
		return []uint64{}, err
	}
	msg = msg[n:]
	ids := make([]uint64, 0)

	for i := uint64(0); i < seen; i++ {
		var id uint64
		n, err := binary.Decode(msg, binary.LittleEndian, &id)
		if err != nil {
			return []uint64{}, err
		}
		ids = append(ids, id)
		msg = msg[n:]
	}
	return ids, nil
}
