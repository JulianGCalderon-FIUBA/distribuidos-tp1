package leaderelection

import (
	"distribuidos/tp1/utils"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/op/go-logging"
)

const NO_ID int = -1

var log = logging.MustGetLogger("log")

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

const MAX_ATTEMPTS = 4
const MAX_PACKAGE_SIZE = 1024

func NewLeaderElection(id uint64, address string, replicas uint64) *LeaderElection {

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	utils.Expect(err, "Did not receive a valid address")

	conn, err := net.ListenUDP("udp", udpAddr)
	utils.Expect(err, "Failed to start listening from connection")

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
		utils.Expect(err, "Failed to start election")
	}()

	for {
		buf := make([]byte, MAX_PACKAGE_SIZE)
		_, recvAddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			continue
		}

		packet, err := Decode(buf)

		switch msg := packet.Msg.(type) {
		case Ack:
			l.HandleAck(packet.Id)
		case Coordinator:
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = l.sendAck(recvAddr, packet.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleCoordinator(msg)
				if err != nil {
					log.Errorf("Failed to handle coordinator message: %v", err)
				}
			}()
		case Election:
			wg.Add(1)
			go func() {
				defer wg.Done()
				err = l.sendAck(recvAddr, packet.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = l.HandleElection(msg)
				if err != nil {
					log.Errorf("Failed to handle election message: %v", err)
				}
			}()
			/* case KeepAlive:
				err = l.sendAck(recvAddr, header.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
			} */
		}
	}
}

func (l *LeaderElection) StartElection() error {
	log.Infof("Starting election")
	e := &Election{Ids: []uint64{l.id}}
	return l.send(e, 1, NO_ID)
}

func (l *LeaderElection) HandleElection(msg Election) error {
	log.Infof("Received Election message")

	if slices.Contains(msg.Ids, l.id) {
		return l.StartCoordinator(msg.Ids)
	}
	msg.Ids = append(msg.Ids, l.id)

	return l.send(msg, 1, NO_ID)
}

func (l *LeaderElection) StartCoordinator(ids []uint64) error {
	log.Infof("Starting coordinator")
	leader := slices.Max(ids)
	l.leaderId = leader

	coor := Coordinator{
		Leader: leader,
		Ids:    []uint64{l.id},
	}

	return l.send(coor, 1, NO_ID)
}

func (l *LeaderElection) HandleCoordinator(msg Coordinator) error {
	log.Infof("Received Coordinator message")

	if slices.Contains(msg.Ids, l.id) {
		return nil
	}

	l.condLeaderId.L.Lock()
	l.leaderId = msg.Leader
	l.hasLeader = true
	l.condLeaderId.L.Unlock()

	l.condLeaderId.Signal()

	log.Infof("Leader is %v", msg.Leader)

	msg.Ids = append(msg.Ids, l.id)
	return l.send(msg, 1, NO_ID)
}

func (l *LeaderElection) HandleAck(msgId uint64) {
	log.Infof("Received ack for message %v", msgId)
	l.mu.Lock()
	ch := l.gotAckMap[msgId]
	l.mu.Unlock()
	ch <- true
}

func (l *LeaderElection) send(msg Message, attempts int, msgId int) error {
	if attempts == MAX_ATTEMPTS {
		// levantar nodo vecino
		return fmt.Errorf("Never got ack")
	}
	// esto es para que en el retry no asigne un id nuevo al mismo mensaje, acepto sugerencias de mejoras
	var id uint64
	if msgId == NO_ID {
		id = l.newMsgId()
	} else {
		id = uint64(msgId)
	}

	packet := Packet{
		Id:  id,
		Msg: msg}

	encoded, err := packet.Encode()
	if err != nil {
		return err
	}

	neighborAddr, err := utils.GetUDPAddr((l.id + 1) % l.replicas)
	if err != nil {
		return err
	}

	n, err := l.conn.WriteToUDP(encoded, neighborAddr)
	if err != nil {
		return err
	}
	if n != len(encoded) {
		return fmt.Errorf("Could not send full message")
	}

	for {
		l.mu.Lock()
		ch := l.gotAckMap[id]
		l.mu.Unlock()
		select {
		case <-ch:
			l.mu.Lock()
			delete(l.gotAckMap, id)
			l.mu.Unlock()
			return nil
		case <-time.After(time.Second):
			log.Infof("Timeout, trying to send again message %v", id)
			return l.send(msg, attempts+1, int(id))
		}
	}
}

func (l *LeaderElection) sendAck(prevNeighbor *net.UDPAddr, msgId uint64) error {

	packet := Packet{
		Id:  msgId,
		Msg: Ack{},
	}

	msg, err := packet.Encode()
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

func (l *LeaderElection) newMsgId() uint64 {
	l.mu.Lock()
	l.lastMsgId += 1
	l.gotAckMap[l.lastMsgId] = make(chan bool)
	defer l.mu.Unlock()

	return l.lastMsgId
}
