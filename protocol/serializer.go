package protocol

import (
	"encoding/binary"
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

func (m *GameBatch) Tag() MessageTag {
	return GameBatchTag
}

func (m *GameBatch) Serialize() ([]byte, error) {
	panic("unimplemented")
}

func (m *GameBatch) Deserialize(buf []byte) error {
	panic("unimplemented")
}

func (m *ReviewBatch) Tag() MessageTag {
	return ReviewBatchTag
}

func (m *ReviewBatch) Serialize() ([]byte, error) {
	panic("unimplemented")
}

func (m *ReviewBatch) Deserialize(buf []byte) error {
	panic("unimplemented")
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
