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
	id       uint64
	replicas uint64

	condLeaderId *sync.Cond
	leaderId     uint64
	hasLeader    bool

	conn *net.UDPConn

	mu        *sync.Mutex
	gotAckMap map[uint64]chan bool
	lastMsgId uint64
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

const MAX_ATTEMPTS = 4

func NewLeaderElection(id uint64, address string, replicas uint64) *LeaderElection {

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	Expect(err, "Did not receive a valid address")

	conn, err := net.ListenUDP("udp", udpAddr)
	Expect(err, "Failed to start listening from connection")

	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	l := &LeaderElection{
		id:           id,
		replicas:     replicas,
		condLeaderId: cond,
		conn:         conn,
		mu:           &sync.Mutex{},
		gotAckMap:    make(map[uint64]chan bool),
		lastMsgId:    0,
	}

	return l
}

func (l *LeaderElection) WaitLeader(amILeader bool) {
	l.condLeaderId.L.Lock()
	defer l.condLeaderId.L.Unlock()
	for amILeader != l.amILeader() {
		l.condLeaderId.Wait()
	}
}

// requires lock
func (l *LeaderElection) amILeader() bool {
	return (l.id == l.leaderId) && l.hasLeader
}

func (l *LeaderElection) Start() error {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := l.StartElection()
		Expect(err, "Failed to start election")
	}()

	for {
		buf := make([]byte, 1024)
		_, recvAddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			continue
		}

		header := MsgHeader{}

		n, err := binary.Decode(buf, binary.LittleEndian, &header)
		if err != nil {
			log.Errorf("Failed to decode header: %v", err)
			continue
		}

		buf = buf[n:]

		switch header.Ty {
		case Ack:
			l.HandleAck(header.Id)
		case Coordinator:
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = l.sendAck(recvAddr, header.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleCoordinator(buf)
				if err != nil {
					log.Errorf("Failed to handle coordinator message: %v", err)
				}
			}()
		case Election:
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = l.sendAck(recvAddr, header.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleElection(buf)
				if err != nil {
					log.Errorf("Failed to handle election message: %v", err)
				}
			}()
		case KeepAlive:
			err = l.sendAck(recvAddr, header.Id)
			if err != nil {
				log.Errorf("Failed to send ack: %v", err)
			}
		}
	}
}

func (l *LeaderElection) StartElection() error {
	log.Infof("Starting election")
	ids := []uint64{l.id}

	encoded, err := l.encodeElection(ids)
	if err != nil {
		return err
	}

	return l.send(encoded, 1, Election)
}

func (l *LeaderElection) HandleElection(msg []byte) error {
	log.Infof("Received Election message")
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
	log.Infof("Starting coordinator")
	leader := slices.Max(ids)
	l.leaderId = leader

	encoded, err := l.encodeCoordinator(leader, []uint64{l.id})
	if err != nil {
		return err
	}

	return l.send(encoded, 1, Coordinator)
}

func (l *LeaderElection) HandleCoordinator(msg []byte) error {
	log.Infof("Received Coordinator message")

	leader, ids, err := decodeCoordinator(msg)
	if err != nil {
		return err
	}

	if slices.Contains(ids, l.id) {
		return nil
	}

	l.condLeaderId.L.Lock()
	l.leaderId = leader
	l.hasLeader = true
	l.condLeaderId.L.Unlock()

	l.condLeaderId.Signal()

	log.Infof("Leader is %v", leader)

	ids = append(ids, l.id)

	encoded, err := l.encodeCoordinator(leader, ids)
	if err != nil {
		return err
	}
	return l.send(encoded, 1, Coordinator)
}

func (l *LeaderElection) HandleAck(msgId uint64) {
	log.Infof("Received ack for message %v", msgId)
	l.mu.Lock()
	ch := l.gotAckMap[msgId]
	l.mu.Unlock()
	ch <- true
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

	neighborAddr, err := GetUDPAddr((l.id + 1) % l.replicas)
	if err != nil {
		return err
	}

	n, err := l.conn.WriteToUDP(msg, neighborAddr)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Could not send full message")
	}

	for {
		l.mu.Lock()
		ch := l.gotAckMap[header.Id]
		l.mu.Unlock()
		select {
		case <-ch:
			l.mu.Lock()
			delete(l.gotAckMap, header.Id)
			l.mu.Unlock()
			return nil
		case <-time.After(time.Second):
			log.Infof("Timeout, trying to send again message %v", header.Id)
			return l.send(msg, attempts+1, msgType)
		}
	}
}

func (l *LeaderElection) sendAck(prevNeighbor *net.UDPAddr, msgId uint64) error {

	header := MsgHeader{
		Ty: Ack,
		Id: msgId,
	}

	msg, err := binary.Append(nil, binary.LittleEndian, header)
	if err != nil {
		return err
	}

	n, err := l.conn.WriteToUDP(msg, prevNeighbor)
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
	return encodeIds(ids, nil)
}

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

func decodeCoordinator(msg []byte) (uint64, []uint64, error) {
	var leader uint64
	n, err := binary.Decode(msg, binary.LittleEndian, &leader)
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
		Ty: ty,
		Id: l.lastMsgId,
	}
}
