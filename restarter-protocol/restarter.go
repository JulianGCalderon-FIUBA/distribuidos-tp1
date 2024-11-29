package restarter

import (
	"context"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
)

const CONFIG_PATH = ".node-config.csv"
const MAX_ATTEMPTS = 3
const MAX_PACKAGE_SIZE = 1024

const RESTARTER_NAME = "restarter-"

var log = logging.MustGetLogger("log")

var ErrFallenNode = errors.New("Never got ack")
var ErrTimeout = errors.New("Never got ack")

type Restarter struct {
	id           uint64
	nodes        []string
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

func readNodeConfig() ([]string, error) {
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

	var nodes []string

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

		nodes = append(nodes, node[0])
	}
	return nodes, nil
}

func NewRestarter(address string, id uint64, replicas uint64) (*Restarter, error) {
	nodes, err := readNodeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read nodes config: %v", err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("did not receive a valid address: %v", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to start listening from connection: %v", err)
	}

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
	}, nil
}

func (r *Restarter) Start(ctx context.Context) error {
	var err error
	closer := utils.SpawnCloser(ctx, r.conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	log.Infof("Listening at %v", r.conn.LocalAddr().String())

	go func() {
		time.Sleep(3 * time.Second)
		err := r.startElection(ctx)
		utils.Expect(err, "Failed to start election")
	}()

	go r.monitorNode(ctx, fmt.Sprintf("%v%v", RESTARTER_NAME, (r.id+1)%r.replicas), utils.RESTARTER_PORT)

	return r.read(ctx)
}

func (r *Restarter) StartMonitoring(ctx context.Context) {
	for _, node := range r.nodes {
		r.wg.Add(1)

		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		go func(nodeName string) {
			defer r.wg.Done()
			r.monitorNode(ctx, nodeName, utils.NODE_PORT)
		}(node)
	}

	<-ctx.Done()
	r.wg.Wait()
}

func (r *Restarter) monitorNode(ctx context.Context, containerName string, port int) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(rand.Intn(4000)+3000) * time.Millisecond):
			msg := KeepAlive{}

			err := r.safeSend(ctx, msg, containerName, port)
			if errors.Is(err, ErrFallenNode) {
				err := r.restartNode(ctx, containerName)
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

func (r *Restarter) read(ctx context.Context) error {
	for {
		buf := make([]byte, MAX_PACKAGE_SIZE)
		_, recvAddr, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("Failed to read: %v", err)
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
			err = r.sendAck(recvAddr, packet.Id)
			if err != nil {
				log.Errorf("Failed to send ack: %v", err)
			}
		}
	}
}

func (r *Restarter) safeSend(ctx context.Context, msg Message, name string, port int) error {
	var err error
	msgId := r.newMsgId()

	addr, err := utils.GetUDPAddr(name, port)
	if err != nil {
		return errors.Join(ErrFallenNode, err)
	}

	packet := Packet{
		Id:  msgId,
		Msg: msg,
	}

	for attempts := 0; attempts < MAX_ATTEMPTS; attempts++ {
		err = r.send(ctx, packet, addr)
		if errors.Is(err, ErrTimeout) {
			log.Errorf("Timeout, trying to send again message %v", msgId)
		} else {
			return err
		}

	}

	return errors.Join(ErrFallenNode, err)
}

func (r *Restarter) send(ctx context.Context, p Packet, addr *net.UDPAddr) error {
	encoded, err := p.Encode()
	if err != nil {
		return err
	}
	_, err = r.conn.WriteToUDP(encoded, addr)
	if err != nil {
		return err
	}

	r.mu.Lock()
	ch := r.ackMap[p.Id]
	r.mu.Unlock()

	select {
	case <-ch:
		r.mu.Lock()
		delete(r.ackMap, p.Id)
		r.mu.Unlock()
	case <-time.After(2 * time.Second):
		return ErrTimeout
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

	_, err = r.conn.WriteToUDP(msg, prevNeighbor)
	if err != nil {
		return err
	}
	return nil
}

func (r *Restarter) handleAck(msgId uint64) {
	r.mu.Lock()
	ch := r.ackMap[msgId]
	r.mu.Unlock()

	ch <- true
}

func (r *Restarter) newMsgId() uint64 {
	r.mu.Lock()
	r.lastMsgId += 1
	// we make the channel buffered to avoid blocking the read loop
	r.ackMap[r.lastMsgId] = make(chan bool, 1)
	defer r.mu.Unlock()

	return r.lastMsgId
}

func (r *Restarter) restartNode(ctx context.Context, containerName string) error {
	if r.isLeader(containerName) {
		log.Infof("Leader has fallen")
		err := r.startElection(ctx)
		if err != nil {
			return err
		}
	}

	log.Infof("Node %v has fallen. Restarting...", containerName)

	cmdStr := fmt.Sprintf("docker start %v", containerName)
	err := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr).Run()
	if err != nil {
		return err
	}

	// wait if SIGTERM signal triggered
	time.Sleep(5 * time.Second)

	return nil
}

func (r *Restarter) isLeader(containerName string) bool {
	if !strings.Contains(containerName, RESTARTER_NAME) {
		return false
	}

	neighborId := (r.id + 1) % r.replicas

	return neighborId == r.leaderId
}
