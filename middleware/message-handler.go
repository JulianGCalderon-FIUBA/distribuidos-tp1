package middleware

import (
	"net"
	"encoding/binary"
)

const PACKET_SIZE = 4

type MessageHandler struct {
	conn net.Conn
}

func NewMessageHandler(conn net.Conn) *MessageHandler {
	return &MessageHandler{
		conn: conn,
	}
}

func (m *MessageHandler) ReceiveMessage() (string, error) {
	size_bytes, err := m.readMessage(PACKET_SIZE)
	if err != nil {
		return "", err
	}

	size := int32(binary.BigEndian.Uint32(size_bytes))

	msg, err := m.readMessage(size)
	return string(msg), err
}

func (m *MessageHandler) SendMessage(msg []byte) error {
	err := m.sendHeader(int32(len(msg)))

	if err != nil {
		return err
	}

	totalSent := 0

	for totalSent < len(msg) {
		n, err := m.conn.Write((msg[totalSent:]))
		if err != nil {
			return err
		}
		totalSent += n
	}

	return nil
}

func (m *MessageHandler) readMessage(size int32) ([]byte, error) {
	buf := make([]byte, size)
	totalRead := 0

	for int32(totalRead) < size {
		n, err := m.conn.Read(buf[totalRead:])
		if err != nil {
			return nil, err
		}
		totalRead += n
	}
	return buf, nil
}

func (m *MessageHandler) sendHeader(size int32) error {
	err := binary.Write(m.conn, binary.BigEndian, int32(size))
	if err != nil {
		return err
	}

	return nil
}