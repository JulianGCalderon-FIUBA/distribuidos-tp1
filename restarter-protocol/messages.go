package restarter

import "encoding/binary"

type Message interface {
	Encode(buf []byte) ([]byte, error)
	Type() MsgType
}

type Packet struct {
	Id  uint64
	Msg Message
}

type MsgType uint8

const (
	AckMsg         MsgType = 'A'
	CoordinatorMsg MsgType = 'C'
	ElectionMsg    MsgType = 'E'
	KeepAliveMsg   MsgType = 'K'
)

func Decode(buf []byte) (Packet, error) {
	var ty MsgType
	n, err := binary.Decode(buf, binary.LittleEndian, &ty)
	if err != nil {
		return Packet{}, err
	}
	buf = buf[n:]

	var id uint64
	n, err = binary.Decode(buf, binary.LittleEndian, &id)
	if err != nil {
		return Packet{}, err
	}
	buf = buf[n:]

	var msg Message
	switch ty {
	case AckMsg:
		msg, err = DecodeAck(buf)
		if err != nil {
			return Packet{}, err
		}
	case CoordinatorMsg:
		msg, err = DecodeCoordinator(buf)
		if err != nil {
			return Packet{}, err
		}
	case ElectionMsg:
		msg, err = DecodeElection(buf)
		if err != nil {
			return Packet{}, err
		}
	}

	return Packet{
		Id:  id,
		Msg: msg,
	}, err
}

func (p Packet) Encode() ([]byte, error) {
	buf, err := binary.Append(nil, binary.LittleEndian, p.Msg.Type())
	if err != nil {
		return nil, err
	}

	buf, err = binary.Append(buf, binary.LittleEndian, p.Id)
	if err != nil {
		return nil, err
	}

	return p.Msg.Encode(buf)
}

type Election struct {
	Ids []uint64
}

type Coordinator struct {
	Leader uint64
	Ids    []uint64
}

type Ack struct {
}

type KeepAlive struct {
}

// Encode messages
func (e Election) Encode(buf []byte) ([]byte, error) { return encodeIds(e.Ids, buf) }
func (c Coordinator) Encode(buf []byte) ([]byte, error) {
	buf, err := binary.Append(buf, binary.LittleEndian, c.Leader)
	if err != nil {
		return []byte{}, err
	}

	buf, err = encodeIds(c.Ids, buf)
	if err != nil {
		return []byte{}, err
	}
	return buf, nil
}
func (a Ack) Encode(buf []byte) ([]byte, error)       { return buf, nil }
func (k KeepAlive) Encode(buf []byte) ([]byte, error) { return buf, nil }

// Decode messages
func DecodeElection(buf []byte) (Election, error) {
	ids, err := decodeIds(buf)
	return Election{Ids: ids}, err
}

func DecodeCoordinator(buf []byte) (Coordinator, error) {
	var leader uint64
	n, err := binary.Decode(buf, binary.LittleEndian, &leader)
	if err != nil {
		return Coordinator{}, err
	}
	buf = buf[n:]

	ids, err := decodeIds(buf)
	return Coordinator{Leader: leader, Ids: ids}, err
}
func DecodeAck(buf []byte) (Ack, error)             { return Ack{}, nil }
func DecodeKeepAlive(buf []byte) (KeepAlive, error) { return KeepAlive{}, nil }

// Return message type
func (e Election) Type() MsgType    { return ElectionMsg }
func (C Coordinator) Type() MsgType { return CoordinatorMsg }
func (a Ack) Type() MsgType         { return AckMsg }
func (k KeepAlive) Type() MsgType   { return KeepAliveMsg }

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
