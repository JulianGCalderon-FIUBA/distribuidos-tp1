package leaderelection

import (
	"context"
	"distribuidos/tp1/utils"
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/op/go-logging"
)

const NO_ID int = -1

var log = logging.MustGetLogger("log")
var ErrFallenNode = errors.New("Never got ack")

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

	fallenNeighbor chan uint64
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

func (l *LeaderElection) Start(ctx context.Context) error {
	var err error
	closer := utils.SpawnCloser(ctx, l.conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	go func() {
		err := l.startElection(ctx)
		utils.Expect(err, "Failed to start election")
	}()

	go func() {
		err := l.monitorNeighbor(ctx)
		if err != nil {
			log.Errorf("Failed to monitor neighbor: %v", err)
		}
	}()

	l.read(ctx)
	return nil
}

func (l *LeaderElection) read(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf := make([]byte, MAX_PACKAGE_SIZE)
			_, recvAddr, err := l.conn.ReadFromUDP(buf)
			if err != nil {
				log.Errorf("Failed to read: %v", err)
				continue
			}

			packet, err := Decode(buf)

			switch msg := packet.Msg.(type) {
			case Ack:
				l.handleAck(packet.Id)
			case Coordinator:
				go func() {
					err = l.sendAck(recvAddr, packet.Id)
					if err != nil {
						log.Errorf("Failed to send ack: %v", err)
					}
					err = l.handleCoordinator(ctx, msg)
					if err != nil {
						log.Errorf("Failed to handle coordinator message: %v", err)
					}
				}()
			case Election:
				go func() {
					err = l.sendAck(recvAddr, packet.Id)
					if err != nil {
						log.Errorf("Failed to send ack: %v", err)
					}
					err = l.handleElection(ctx, msg)
					if err != nil {
						log.Errorf("Failed to handle election message: %v", err)
					}
				}()
			case KeepAlive:
				err = l.sendAck(recvAddr, packet.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
			}
		}
	}
}

func (l *LeaderElection) monitorNeighbor(ctx context.Context) error {
	msg := KeepAlive{}
	err := l.sendToRing(ctx, msg)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return nil
	case <-l.fallenNeighbor: // recibir id para saber a quien levantar
		// levantar vecino
	}
	return nil
}

func (l *LeaderElection) startElection(ctx context.Context) error {
	log.Infof("Starting election")
	e := &Election{Ids: []uint64{l.id}}
	return l.sendToRing(ctx, e)
}

func (l *LeaderElection) handleElection(ctx context.Context, msg Election) error {
	log.Infof("Received Election message")

	if slices.Contains(msg.Ids, l.id) {
		return l.startCoordinator(ctx, msg.Ids)
	}
	msg.Ids = append(msg.Ids, l.id)

	return l.sendToRing(ctx, msg)
}

func (l *LeaderElection) startCoordinator(ctx context.Context, ids []uint64) error {
	log.Infof("Starting coordinator")
	leader := slices.Max(ids)
	l.leaderId = leader

	coor := Coordinator{
		Leader: leader,
		Ids:    []uint64{l.id},
	}

	return l.sendToRing(ctx, coor)
}

func (l *LeaderElection) handleCoordinator(ctx context.Context, msg Coordinator) error {
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
	return l.sendToRing(ctx, msg)
}

func (l *LeaderElection) handleAck(msgId uint64) {
	log.Infof("Received ack for message %v", msgId)
	l.mu.Lock()
	ch := l.gotAckMap[msgId]
	l.mu.Unlock()
	ch <- true
}

func (l *LeaderElection) sendToRing(ctx context.Context, msg Message) error {
	next := l.id + 1
	for next%l.replicas != l.id {
		addr, err := utils.GetUDPAddr(next % l.replicas)
		if err != nil {
			return err
		}
		err = l.safeSend(ctx, msg, addr)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrFallenNode) {
			return err
		}
		if next == l.id+1 {
			l.fallenNeighbor <- next
		}
		next += 1
	}
	return nil
}

func (l *LeaderElection) safeSend(ctx context.Context, msg Message, addr *net.UDPAddr) error {
	var err error
	id := l.newMsgId()
	packet := Packet{
		Id:  id,
		Msg: msg,
	}

	for attempts := 0; attempts < MAX_ATTEMPTS; attempts += 1 {
		err = l.send(ctx, packet, addr)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			return err
		}
	}
	log.Errorf("Never got ack for message %v", id)
	return ErrFallenNode
}

func (l *LeaderElection) send(ctx context.Context, p Packet, addr *net.UDPAddr) error {
	encoded, err := p.Encode()
	if err != nil {
		return err
	}

	n, err := l.conn.WriteToUDP(encoded, addr)
	if err != nil {
		return err
	}
	if n != len(encoded) {
		return fmt.Errorf("Could not send full message")
	}

	l.mu.Lock()
	ch := l.gotAckMap[p.Id]
	l.mu.Unlock()
	select {
	case <-ch:
		l.mu.Lock()
		delete(l.gotAckMap, p.Id)
		l.mu.Unlock()
		return nil
	case <-time.After(time.Second):
		log.Infof("Timeout, trying to send again message %v", p.Id)
		return os.ErrDeadlineExceeded
	case <-ctx.Done():
		return nil
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
