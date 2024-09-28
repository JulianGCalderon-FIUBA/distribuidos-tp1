package middleware

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const PACKET_SIZE = 1

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

	size := int8(size_bytes[0])

	fmt.Printf("Received message of size %d\n", size)

	msg, err := m.readMessage(size)
	return string(msg), err
}

func (m *MessageHandler) SendMessage(msg []byte) error {
	if err := m.sendHeader(int8(len(msg))); err != nil {
		return err
	}

	if _, err := m.conn.Write(msg); err != nil {
		return err
	}

	return nil
}

func (m *MessageHandler) readMessage(size int8) ([]byte, error) {
	buf := make([]byte, size)

	if _, err := io.ReadFull(m.conn, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (m *MessageHandler) sendHeader(size int8) error {
	if err := binary.Write(m.conn, binary.BigEndian, size); err != nil {
		return err
	}

	return nil
}
