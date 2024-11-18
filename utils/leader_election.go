package utils

import (
	"encoding/binary"
	"fmt"
	"net"
	"slices"
	"time"
)

type LeaderElection struct {
	id          uint64
	leaderId    uint64
	conn        *net.UDPConn
	nextAddress *net.UDPAddr
	gotAck      chan bool
}

type MsgType rune

const (
	Election    MsgType = 'E'
	Coordinator MsgType = 'C'
	Ack         MsgType = 'A'
	KeepAlive   MsgType = 'K'
)

const MAX_ATTEMPTS = 4

func NewLeaderElection(id uint64, address string, nextAddress string) *LeaderElection {

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		Expect(err, "Did not receive a valid udp address")
	}
	nextAddr, err := net.ResolveUDPAddr("udp", nextAddress)
	if err != nil {
		Expect(err, "Did not receive a valid udp address for neighbor")
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		Expect(err, "Failed to start listening from connection")
	}

	l := &LeaderElection{
		id:          id,
		conn:        conn,
		nextAddress: nextAddr,
		gotAck:      make(chan bool),
	}

	return l
}

func (l *LeaderElection) AmILeader() bool {
	return l.id == l.leaderId
}

func (l *LeaderElection) Start() error {
	for {
		var buf []byte

		_, recvAddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			continue
		}

		var msgType MsgType

		n, err := binary.Decode(buf, binary.LittleEndian, msgType)
		if err != nil {
			log.Errorf("Failed to decode message type: %v", err)
			continue
		}

		buf = buf[n:]

		switch msgType {
		case Election:
			go func() {
				err = l.sendAck(recvAddr)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleElection(buf)
				if err != nil {
					log.Errorf("Failed to handle election message: %v", err)
				}
			}()
		case Coordinator:
			go func() {
				err = l.sendAck(recvAddr)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleCoordinator(buf)
				if err != nil {
					log.Errorf("Failed to handle coordinator message: %v", err)
				}
			}()
		case Ack:
			l.HandleAck()
		case KeepAlive:
			// to do
		}
	}
}

func (l *LeaderElection) HandleElection(msg []byte) error {
	ids, err := decodeIds(msg)
	if err != nil {
		return err
	}
	if slices.Contains(ids, l.id) {
		return l.StartCoordinator(ids)
	}
	ids = append(ids, l.id)

	encoded, err := encodeElection(ids)
	if err != nil {
		return err
	}

	return l.send(encoded, 1)
}

func (l *LeaderElection) StartCoordinator(ids []uint64) error {

	leader := slices.Max(ids)
	l.leaderId = leader

	encoded, err := encodeCoordinator(leader, []uint64{l.id})
	if err != nil {
		return err
	}

	return l.send(encoded, 1)
}

func (l *LeaderElection) HandleCoordinator(msg []byte) error {

	leader, ids, err := decodeCoordinator(msg)
	if err != nil {
		return err
	}

	if slices.Contains(ids, l.id) {
		return nil
	}

	l.leaderId = leader
	ids = append(ids, l.id)

	encoded, err := encodeCoordinator(leader, ids)
	if err != nil {
		return err
	}
	return l.send(encoded, 1)
}

func (l *LeaderElection) HandleAck() {
	l.gotAck <- true
}

func (l *LeaderElection) send(msg []byte, attempts int) error {
	if attempts == MAX_ATTEMPTS {
		// levantar nodo vecino
		return fmt.Errorf("Never got ack")
	}
	n, err := l.conn.WriteTo(msg, l.nextAddress)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Could not send full message")
	}
	for {
		select {
		case <-l.gotAck:
			return nil
		case <-time.After(time.Second):
			return l.send(msg, attempts+1)
		}
	}
}

func (l *LeaderElection) sendAck(prevNeighbor *net.UDPAddr) error {
	msg, err := binary.Append(nil, binary.LittleEndian, Ack)
	if err != nil {
		return err
	}

	n, err := l.conn.WriteTo(msg, prevNeighbor)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Could not send full message")
	}
	return nil
}

func encodeCoordinator(leader uint64, ids []uint64) ([]byte, error) {
	buf, err := binary.Append(nil, binary.LittleEndian, Coordinator)
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
	buf, err := binary.Append(nil, binary.LittleEndian, Election)
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
