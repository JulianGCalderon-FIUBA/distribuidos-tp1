package leaderelection

import "encoding/binary"

type Message interface {
	Encode(buf []byte) ([]byte, error)
	Decode(buf []byte) error
	EncodeWithHeader(msgId uint64) ([]byte, error)
	DecodeWithHeader(buf []byte) (MsgHeader, error)
	Type() MsgType
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

// Encode messages
func (em ElectionMsg) Encode(buf []byte) ([]byte, error) { return encodeIds(em.Ids, buf) }
func (cm CoordinatorMsg) Encode(buf []byte) ([]byte, error) {
	buf, err := binary.Append(buf, binary.LittleEndian, cm.Leader)
	if err != nil {
		return []byte{}, err
	}

	buf, err = encodeIds(cm.Ids, buf)
	if err != nil {
		return []byte{}, err
	}
	return buf, nil
}
func (am AckMsg) Encode(buf []byte) ([]byte, error) { return []byte{}, nil }

// Encode messages with header
func (em ElectionMsg) EncodeWithHeader(msgId uint64) ([]byte, error) {

	h := MsgHeader{
		Ty: em.Type(),
		Id: msgId,
	}

	buf, err := binary.Append(nil, binary.LittleEndian, h)
	if err != nil {
		return []byte{}, err
	}

	return em.Encode(buf)
}
func (cm CoordinatorMsg) EncodeWithHeader(msgId uint64) ([]byte, error) {

	h := MsgHeader{
		Ty: cm.Type(),
		Id: msgId,
	}

	buf, err := binary.Append(nil, binary.LittleEndian, h)
	if err != nil {
		return []byte{}, err
	}

	return cm.Encode(buf)
}
func (am AckMsg) EncodeWithHeader(msgId uint64) ([]byte, error) {
	h := MsgHeader{
		Ty: am.Type(),
		Id: msgId,
	}

	buf, err := binary.Append(nil, binary.LittleEndian, h)
	return buf, err
}

// Decode messages
func (em *ElectionMsg) Decode(buf []byte) error {
	var err error
	em.Ids, err = decodeIds(buf)
	return err
}
func (cm *CoordinatorMsg) Decode(buf []byte) error {
	n, err := binary.Decode(buf, binary.LittleEndian, &cm.Leader)
	if err != nil {
		return err
	}
	buf = buf[n:]

	cm.Ids, err = decodeIds(buf)
	return err
}
func (a AckMsg) Decode(buf []byte) error { return nil }

// Decode messages with header
func (em *ElectionMsg) DecodeWithHeader(buf []byte) (MsgHeader, error) {
	header := MsgHeader{}
	n, err := binary.Decode(buf, binary.LittleEndian, &header)
	if err != nil {
		return header, err
	}
	buf = buf[n:]
	return header, em.Decode(buf)
}

func (cm *CoordinatorMsg) DecodeWithHeader(buf []byte) (MsgHeader, error) {
	header := MsgHeader{}
	n, err := binary.Decode(buf, binary.LittleEndian, &header)
	if err != nil {
		return header, err
	}
	buf = buf[n:]

	return header, cm.Decode(buf)
}
func (a AckMsg) DecodeWithHeader(buf []byte) (MsgHeader, error) {
	header := MsgHeader{}
	_, err := binary.Decode(buf, binary.LittleEndian, &header)
	if err != nil {
		return header, err
	}
	return header, nil
}

func (em ElectionMsg) Type() MsgType    { return Election }
func (cm CoordinatorMsg) Type() MsgType { return Coordinator }
func (a AckMsg) Type() MsgType          { return Ack }

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
