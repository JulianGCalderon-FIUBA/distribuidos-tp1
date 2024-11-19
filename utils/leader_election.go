package utils

import (
	"encoding/binary"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"
)

type LeaderElection struct {
	id uint64

	condLeaderId *sync.Cond
	leaderId     uint64

	conn        *net.UDPConn
	nextAddress *net.UDPAddr

	mu        *sync.Mutex
	gotAckMap map[uint64]chan bool
	lastMsgId uint64
}

type MsgType rune

const (
	Ack         MsgType = 'A'
	Coordinator MsgType = 'C'
	Election    MsgType = 'E'
	KeepAlive   MsgType = 'K'
)

type MsgHeader struct {
	ty MsgType
	id uint64
}

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

	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	l := &LeaderElection{
		id:           id,
		condLeaderId: cond,
		conn:         conn,
		nextAddress:  nextAddr,
		mu:           &sync.Mutex{},
		gotAckMap:    make(map[uint64]chan bool),
		lastMsgId:    id,
	}

	return l
}

func (l *LeaderElection) WaitLeader(amILeader bool) {
	l.condLeaderId.L.Lock()
	defer l.condLeaderId.L.Unlock()
	for amILeader == (l.id != l.leaderId) {
		l.condLeaderId.Wait()
	}
}

func (l *LeaderElection) Start() error {
	// start election
	for {
		var buf []byte

		_, recvAddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			continue
		}

		var header MsgHeader
		n, err := binary.Decode(buf, binary.LittleEndian, header)
		if err != nil {
			log.Errorf("Failed to decode message type: %v", err)
			continue
		}

		buf = buf[n:]

		switch header.ty {
		case Ack:
			l.HandleAck(header.id)
		case Coordinator:
			go func() {
				err = l.sendAck(recvAddr, header.id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleCoordinator(buf)
				if err != nil {
					log.Errorf("Failed to handle coordinator message: %v", err)
				}
			}()
		case Election:
			go func() {
				err = l.sendAck(recvAddr, header.id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleElection(buf)
				if err != nil {
					log.Errorf("Failed to handle election message: %v", err)
				}
			}()
		case KeepAlive:
			err = l.sendAck(recvAddr, header.id)
			if err != nil {
				log.Errorf("Failed to send ack: %v", err)
			}
		}
	}
}

func (l *LeaderElection) StartElection() error {
	ids := []uint64{l.id}

	encoded, err := l.encodeElection(ids)
	if err != nil {
		return err
	}

	return l.send(encoded, 1, Election)
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

	encoded, err := l.encodeElection(ids)
	if err != nil {
		return err
	}

	return l.send(encoded, 1, Election)
}

func (l *LeaderElection) StartCoordinator(ids []uint64) error {

	leader := slices.Max(ids)
	l.leaderId = leader

	encoded, err := l.encodeCoordinator(leader, []uint64{l.id})
	if err != nil {
		return err
	}

	return l.send(encoded, 1, Coordinator)
}

func (l *LeaderElection) HandleCoordinator(msg []byte) error {

	leader, ids, err := decodeCoordinator(msg)
	if err != nil {
		return err
	}

	if slices.Contains(ids, l.id) {
		return nil
	}

	l.condLeaderId.L.Lock()
	l.leaderId = leader
	l.condLeaderId.L.Unlock()

	l.condLeaderId.Signal()

	ids = append(ids, l.id)

	encoded, err := l.encodeCoordinator(leader, ids)
	if err != nil {
		return err
	}
	return l.send(encoded, 1, Coordinator)
}

func (l *LeaderElection) HandleAck(msgId uint64) {
	l.mu.Lock()
	l.gotAckMap[msgId] <- true
	l.mu.Unlock()
}

func (l *LeaderElection) send(msg []byte, attempts int, msgType MsgType) error {
	if attempts == MAX_ATTEMPTS {
		// levantar nodo vecino
		return fmt.Errorf("Never got ack")
	}

	header := l.newHeader(msgType)
	buf, err := binary.Append(nil, binary.LittleEndian, header)
	if err != nil {
		return err
	}
	msg = append(buf, msg...)

	n, err := l.conn.WriteTo(msg, l.nextAddress)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Could not send full message")
	}

	for {
		l.mu.Lock()
		defer l.mu.Unlock()
		ch := l.gotAckMap[header.id]
		select {
		case <-ch:
			delete(l.gotAckMap, header.id)
			return nil
		case <-time.After(time.Second):
			return l.send(msg, attempts+1, msgType)
		default:
			l.mu.Unlock()
		}
	}
}

func (l *LeaderElection) sendAck(prevNeighbor *net.UDPAddr, msgId uint64) error {

	header := MsgHeader{
		ty: Ack,
		id: msgId,
	}

	msg, err := binary.Append(nil, binary.LittleEndian, header)
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

func (l *LeaderElection) encodeCoordinator(leader uint64, ids []uint64) ([]byte, error) {

	buf, err := binary.Append(nil, binary.LittleEndian, leader)
	if err != nil {
		return []byte{}, err
	}

	msg, err := encodeIds(ids, buf)
	if err != nil {
		return []byte{}, err
	}
	return msg, nil
}

func (l *LeaderElection) encodeElection(ids []uint64) ([]byte, error) {
	buf := make([]byte, 0)
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

func (l *LeaderElection) newHeader(ty MsgType) MsgHeader {
	l.mu.Lock()
	l.lastMsgId += 1
	l.gotAckMap[l.lastMsgId] = make(chan bool)
	l.mu.Unlock()

	return MsgHeader{
		ty: ty,
		id: l.lastMsgId,
	}
}
