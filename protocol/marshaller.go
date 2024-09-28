package protocol

import (
	"encoding/binary"
	"io"
)

type Header struct {
	Size uint64
	Tag  MessageTag
}

type Unmarshaller struct {
	r io.Reader
}

func NewUnmarshaller(r io.Reader) *Unmarshaller {
	return &Unmarshaller{
		r: r,
	}
}

func (m *Unmarshaller) ReceiveMessage() (Message, error) {
	var header Header
	err := binary.Read(m.r, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	// interface magic: we assign to interface msg the correct type based
	// on the header tag, and then we deserialize polimorphicaly
	var msg Message
	switch header.Tag {
	case RequestHelloTag:
		msg = &RequestHello{}
	case AcceptRequestTag:
		msg = &AcceptRequest{}
	case DataHelloTag:
		msg = &DataHello{}
	case DataAcceptTag:
		msg = &DataAccept{}
	case GameBatchTag:
		msg = &GameBatch{}
	case ReviewBatchTag:
		msg = &ReviewBatch{}
	case FinishTag:
		msg = &Finish{}
	}

	buf := make([]byte, header.Size)
	_, err = io.ReadFull(m.r, buf)
	if err != nil {
		return nil, err
	}

	err = msg.Deserialize(buf)
	return msg, err
}

type Marshaller struct {
	w io.Writer
}

func NewMarshaller(w io.Writer) *Marshaller {
	return &Marshaller{
		w: w,
	}
}

func (m *Marshaller) SendMessage(msg Message) error {
	buf, err := msg.Serialize()
	if err != nil {
		return err
	}

	header := Header{
		Size: uint64(len(buf)),
		Tag:  msg.Tag(),
	}
	err = binary.Write(m.w, binary.LittleEndian, header)
	if err != nil {
		return err
	}

	_, err = m.w.Write(buf)
	return err
}
