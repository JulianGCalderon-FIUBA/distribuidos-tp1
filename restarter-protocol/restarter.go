package restarter

import (
	"context"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"slices"
	"sync"
	"time"

	"github.com/op/go-logging"
)

const CONFIG_PATH = ".node-config.csv"
const MAX_ATTEMPTS = 3
const MAX_PACKAGE_SIZE = 1024

const CONTAINER_NAME = "restarter-"
const RESTARTER_PORT = 14300

var log = logging.MustGetLogger("log")

var ErrFallenNode = errors.New("Never got ack")

type Restarter struct {
	id           uint64
	nodes        map[string]string
	replicas     uint64
	conn         *net.UDPConn
	ackMap       map[uint64]chan bool
	lastMsgId    uint64
	condLeaderId *sync.Cond
	leaderId     uint64
	hasLeader    bool
	mu           *sync.Mutex
	wg           *sync.WaitGroup
}

func readNodeConfig() (map[string]string, error) {
	file, err := os.Open(CONFIG_PATH)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	csvReader := csv.NewReader(file)

	_, err = csvReader.Read()
	if err != nil {
		return nil, err
	}

	var nodes = make(map[string]string)

	for {
		node, err := csvReader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			log.Errorf("Failed to parse row: %v", err)
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		port := node[1] // config file is guaranteed to have 2 fields for all its rows
		nodes[node[0]] = port
	}
	return nodes, nil
}

func NewRestarter(address string, id uint64, replicas uint64) *Restarter {
	nodes, err := readNodeConfig()
	utils.Expect(err, "Failed to read nodes config")

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	utils.Expect(err, "Did not receive a valid address")

	conn, err := net.ListenUDP("udp", udpAddr)
	utils.Expect(err, "Failed to start listening from connection")

	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	return &Restarter{
		nodes:        nodes,
		conn:         conn,
		id:           id,
		replicas:     replicas,
		condLeaderId: cond,
		mu:           &mu,
		ackMap:       make(map[uint64]chan bool),
		lastMsgId:    0,
		wg:           &sync.WaitGroup{},
	}
}

func (r *Restarter) WaitLeader(amILeader bool) {
	r.condLeaderId.L.Lock()
	defer r.condLeaderId.L.Unlock()
	for amILeader != r.amILeader() {
		r.condLeaderId.Wait()
	}
}

// requires lock
func (r *Restarter) amILeader() bool {
	return (r.id == r.leaderId) && r.hasLeader
}

func (r *Restarter) Start(ctx context.Context) error {
	var err error
	closer := utils.SpawnCloser(ctx, r.conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	go func() {
		time.Sleep(3 * time.Second)
		err := r.startElection(ctx)
		utils.Expect(err, "Failed to start election")
	}()

	go r.monitorNeighbor(ctx)

	r.read(ctx)

	return nil
}

func (r *Restarter) StartMonitoring(ctx context.Context) {
	for node := range r.nodes {
		r.wg.Add(1)
		go func(nodeName string) {
			defer r.wg.Done()
			time.Sleep(time.Second)
			err := r.monitorNode(ctx, nodeName)
			utils.Expect(err, "Failed to monitor node")
		}(node)
	}
}

func (r *Restarter) monitorNode(ctx context.Context, nodeName string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", nodeName, r.nodes[nodeName]))
	utils.Expect(err, "Failed to resolve address")

	for {
		select {
		case <-ctx.Done():
			log.Infof("I am no longer leader of recevied a SIGTERM")
			return nil
		default:
			time.Sleep(time.Second)
			err = r.safeSend(ctx, KeepAlive{}, udpAddr)
			if err != nil {
				if errors.Is(err, ErrFallenNode) {
					err = r.restartNode(ctx, nodeName)
					utils.Expect(err, fmt.Sprintf("Failed to restart node %v", nodeName))
				} else {
					return fmt.Errorf("Failed to send keep alive: %v", err)
				}
			}
		}
	}
}

func (r *Restarter) read(ctx context.Context) {
	for {
		buf := make([]byte, MAX_PACKAGE_SIZE)
		_, recvAddr, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			continue
		}

		packet, err := Decode(buf)
		if err != nil {
			log.Errorf("Failed to decode packet: %v", err)
			continue
		}

		switch msg := packet.Msg.(type) {
		case Ack:
			r.handleAck(packet.Id)
		case Coordinator:
			go func() {
				err = r.sendAck(recvAddr, packet.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = r.handleCoordinator(ctx, msg)
				if err != nil {
					log.Errorf("Failed to handle coordinator message: %v", err)
				}
			}()
		case Election:
			go func() {
				err = r.sendAck(recvAddr, packet.Id)
				if err != nil {
					log.Errorf("Failed to send ack: %v", err)
				}
				err = r.handleElection(ctx, msg)
				if err != nil {
					log.Errorf("Failed to handle election message: %v", err)
				}
			}()
		case KeepAlive:
			log.Infof("Received keep alive from %v", recvAddr)
			err = r.sendAck(recvAddr, packet.Id)
			if err != nil {
				log.Errorf("Failed to send ack: %v", err)
			}
		}
	}
}

func (r *Restarter) safeSend(ctx context.Context, msg Message, addr *net.UDPAddr) error {
	var err error
	msgId := r.newMsgId()

	packet := Packet{
		Id:  msgId,
		Msg: msg}

	for attempts := 0; attempts < MAX_ATTEMPTS; attempts++ {
		err = r.send(ctx, packet, addr)
		if err == nil {
			return nil
		}
	}

	log.Errorf("Never got ack for message %d", msgId)
	return ErrFallenNode
}

func (r *Restarter) send(ctx context.Context, p Packet, addr *net.UDPAddr) error {
	encoded, err := p.Encode()
	if err != nil {
		return err
	}
	n, err := r.conn.WriteToUDP(encoded, addr)
	if err != nil {
		return err
	}
	if n != len(encoded) {
		return fmt.Errorf("Could not send full message")
	}

	r.mu.Lock()
	ch := r.ackMap[p.Id]
	r.mu.Unlock()
	select {
	case <-ch:
		log.Infof("Received ack for message %v", p.Id)
		r.mu.Lock()
		delete(r.ackMap, p.Id)
		r.mu.Unlock()
	case <-time.After(time.Second):
		log.Infof("Timeout, trying to send again message %v", p.Id)
		return os.ErrDeadlineExceeded
	case <-ctx.Done():
		return nil
	}
	return nil
}

func (r *Restarter) sendAck(prevNeighbor *net.UDPAddr, msgId uint64) error {
	packet := Packet{
		Id:  msgId,
		Msg: Ack{},
	}

	msg, err := packet.Encode()
	if err != nil {
		return err
	}

	n, err := r.conn.WriteToUDP(msg, prevNeighbor)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Could not send full message")
	}
	return nil
}

func (r *Restarter) handleAck(msgId uint64) {
	r.mu.Lock()
	ch := r.ackMap[msgId]
	r.mu.Unlock()

	ch <- true
}

func (r *Restarter) startElection(ctx context.Context) error {
	log.Infof("Starting election")
	e := &Election{Ids: []uint64{r.id}}
	return r.sendToRing(ctx, e)
}

func (r *Restarter) handleElection(ctx context.Context, msg Election) error {
	log.Infof("Received Election message")

	if slices.Contains(msg.Ids, r.id) {
		return r.startCoordinator(ctx, msg.Ids)
	}
	msg.Ids = append(msg.Ids, r.id)

	return r.sendToRing(ctx, msg)
}

func (r *Restarter) startCoordinator(ctx context.Context, ids []uint64) error {
	log.Infof("Starting coordinator")
	leader := slices.Max(ids)
	r.condLeaderId.L.Lock()
	r.leaderId = leader
	r.hasLeader = true
	r.condLeaderId.L.Unlock()

	r.condLeaderId.Signal()

	log.Infof("Leader is %v", leader)

	coor := Coordinator{
		Leader: leader,
		Ids:    []uint64{r.id},
	}

	return r.sendToRing(ctx, coor)
}

func (r *Restarter) handleCoordinator(ctx context.Context, msg Coordinator) error {
	log.Infof("Received Coordinator message")

	if slices.Contains(msg.Ids, r.id) {
		return nil
	}

	r.condLeaderId.L.Lock()
	r.leaderId = msg.Leader
	r.hasLeader = true
	r.condLeaderId.L.Unlock()

	r.condLeaderId.Signal()

	log.Infof("Leader is %v", msg.Leader)

	msg.Ids = append(msg.Ids, r.id)
	return r.sendToRing(ctx, msg)
}

func (r *Restarter) sendToRing(ctx context.Context, msg Message) error {
	next := r.id + 1
	for {
		host := fmt.Sprintf("%v%v", CONTAINER_NAME, next%r.replicas)
		addr, _ := utils.GetUDPAddr(host, RESTARTER_PORT)
		err := r.safeSend(ctx, msg, addr)
		if err == nil {
			return nil
		}
		log.Infof("Neighbor %v is not answering, sending message to next one", next%r.replicas)
		next += 1
	}
}

func (r *Restarter) newMsgId() uint64 {
	r.mu.Lock()
	r.lastMsgId += 1
	r.ackMap[r.lastMsgId] = make(chan bool)
	defer r.mu.Unlock()

	return r.lastMsgId
}

func (r *Restarter) monitorNeighbor(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msg := KeepAlive{}
			host := fmt.Sprintf("%v%v", CONTAINER_NAME, (r.id+1)%r.replicas)
			addr, _ := utils.GetUDPAddr(host, RESTARTER_PORT)
			err := r.safeSend(ctx, msg, addr)
			if errors.Is(err, ErrFallenNode) {
				err := r.restartNeighbor(ctx, (r.id+1)%r.replicas)
				if err != nil {
					log.Errorf("Failed to restart neighbor: %v", err)
				}
				continue
			}
			if err != nil {
				log.Errorf("Failed to send keep alive: %v", err)
			}
		}
	}
}

func (r *Restarter) restartNeighbor(ctx context.Context, id uint64) error {
	log.Errorf("Neighbor %v has fallen. Restarting...", id)
	if id == r.leaderId {
		log.Infof("Leader has fallen, starting election")
		err := r.startElection(ctx)
		if err != nil {
			return err
		}
	}
	return r.restartNode(ctx, fmt.Sprintf("%v%v", CONTAINER_NAME, id))
}

func (r *Restarter) restartNode(ctx context.Context, containerName string) error {
	log.Infof("Restarting [%v]", containerName)
	time.Sleep(2 * time.Second) // wait if SIGTERM signal triggered
	cmdStr := fmt.Sprintf("docker restart %v", containerName)
	_, err := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr).Output()

	return err
}
