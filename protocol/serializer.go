package protocol

import (
	"encoding/binary"
	"slices"
)

type Message interface {
	Tag() MessageTag
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

func (m *RequestHello) Tag() MessageTag {
	return RequestHelloTag
}

func (m *RequestHello) Serialize() ([]byte, error) {
	return binary.Append(nil, binary.LittleEndian, m)
}

func (m *RequestHello) Deserialize(buf []byte) error {
	_, err := binary.Decode(buf, binary.LittleEndian, m)
	return err
}

func (m *AcceptRequest) Tag() MessageTag {
	return AcceptRequestTag
}

func (m *AcceptRequest) Serialize() ([]byte, error) {
	return binary.Append(nil, binary.LittleEndian, m)
}

func (m *AcceptRequest) Deserialize(buf []byte) error {
	_, err := binary.Decode(buf, binary.LittleEndian, m)
	return err
}

func (m *DataHello) Tag() MessageTag {
	return DataHelloTag
}

func (m *DataHello) Serialize() ([]byte, error) {
	return binary.Append(nil, binary.LittleEndian, m)
}

func (m *DataHello) Deserialize(buf []byte) error {
	_, err := binary.Decode(buf, binary.LittleEndian, m)
	return err
}

func (m *DataAccept) Tag() MessageTag {
	return DataAcceptTag
}

func (m *DataAccept) Serialize() ([]byte, error) {
	return binary.Append(nil, binary.LittleEndian, m)
}

func (m *DataAccept) Deserialize(buf []byte) error {
	_, err := binary.Decode(buf, binary.LittleEndian, m)
	return err
}

func (m *Prepare) Tag() MessageTag {
	return PrepareTag
}

func (m *Prepare) Serialize() ([]byte, error) {
	msg, err := binary.Append(nil, binary.LittleEndian, uint32(len(m.Header)))
	if err != nil {
		return nil, err
	}
	msg = append(msg, m.Header...)
	return msg, nil
}

func (m *Prepare) Deserialize(buf []byte) error {
	var headerLength uint32
	read, err := binary.Decode(buf, binary.LittleEndian, &headerLength)
	if err != nil {
		return err
	}
	buf = buf[read:]
	m.Header = slices.Clone(buf[:headerLength])
	return nil
}

func (m *Batch) Tag() MessageTag {
	return BatchTag
}

func (m *Batch) Serialize() ([]byte, error) {
	var msg []byte
	var err error
	msg, err = binary.Append(msg, binary.LittleEndian, uint32(len(m.Lines)))
	if err != nil {
		return nil, err
	}

	for _, array := range m.Lines {
		msg, err = binary.Append(msg, binary.LittleEndian, uint32(len(array)))
		if err != nil {
			return nil, err
		}
		msg = append(msg, array...)
	}
	return msg, err
}

func (m *Batch) Deserialize(buf []byte) error {
	var amountArrays uint32
	read, err := binary.Decode(buf, binary.LittleEndian, &amountArrays)
	if err != nil {
		return err
	}
	buf = buf[read:]

	for i := 0; i < int(amountArrays); i++ {
		var size uint32
		read, err := binary.Decode(buf, binary.LittleEndian, &size)
		if err != nil {
			return err
		}
		buf = buf[read:]
		array := buf[:size]
		m.Lines = append(m.Lines, array)
		buf = buf[size:]
	}

	return nil
}

func (m *Finish) Tag() MessageTag {
	return FinishTag
}

func (m *Finish) Serialize() ([]byte, error) {
	return binary.Append(nil, binary.LittleEndian, m)
}

func (m *Finish) Deserialize(buf []byte) error {
	_, err := binary.Decode(buf, binary.LittleEndian, m)
	return err
}
