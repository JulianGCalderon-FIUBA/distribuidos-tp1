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
	"sync"
	"time"

	"github.com/op/go-logging"
)

const CONFIG_PATH = ".node-config.csv"
const MAX_ATTEMPTS = 3
const MAX_PACKAGE_SIZE = 1024

var log = logging.MustGetLogger("log")

var ErrFallenNode = errors.New("Never got ack")

type Restarter struct {
	nodes     map[string]string
	conn      *net.UDPConn
	mu        *sync.Mutex
	ackMap    map[uint64]chan bool
	lastMsgId uint64
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

func NewRestarter(address string) *Restarter {
	nodes, err := readNodeConfig()
	utils.Expect(err, "Failed to read nodes config")

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	utils.Expect(err, "Did not receive a valid address")

	conn, err := net.ListenUDP("udp", udpAddr)
	utils.Expect(err, "Failed to start listening from connection")

	return &Restarter{
		nodes:     nodes,
		conn:      conn,
		mu:        &sync.Mutex{},
		ackMap:    make(map[uint64]chan bool),
		lastMsgId: 0,
	}
}

func (r *Restarter) Start(ctx context.Context) error {
	var err error
	closer := utils.SpawnCloser(ctx, r.conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.read()
	}()

	for node := range r.nodes {
		wg.Add(1)
		go func(nodeName string) {
			defer wg.Done()
			time.Sleep(5 * time.Second) // ver como hacer que espere con docker
			err = r.monitorNode(ctx, nodeName)
			utils.Expect(err, "Failed to monitor node")
		}(node)
	}

	defer wg.Wait()
	return nil
}

func (r *Restarter) monitorNode(ctx context.Context, nodeName string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", nodeName, r.nodes[nodeName]))
	utils.Expect(err, "Failed to resolve address")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
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

func (r *Restarter) read() {
	for {
		buf := make([]byte, MAX_PACKAGE_SIZE)
		_, _, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			return
		}

		packet, err := Decode(buf)
		if err != nil {
			log.Errorf("Failed to decode packet: %v", err)
			continue
		}

		switch packet.Msg.(type) { // lo dejo asi para ver si se puede reutilizar con la Leader Election
		case Ack:
			r.handleAck(packet.Id)
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

func (r *Restarter) handleAck(msgId uint64) {
	r.mu.Lock()
	ch := r.ackMap[msgId]
	r.mu.Unlock()

	ch <- true
}

func (r *Restarter) newMsgId() uint64 {
	r.mu.Lock()
	r.lastMsgId += 1
	r.ackMap[r.lastMsgId] = make(chan bool)
	defer r.mu.Unlock()

	return r.lastMsgId
}

func (r *Restarter) restartNode(ctx context.Context, containerName string) error {
	log.Infof("Restarting [%v]", containerName)
	time.Sleep(2 * time.Second) // wait if SIGTERM signal triggered
	cmdStr := fmt.Sprintf("docker restart %v", containerName)
	_, err := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr).Output()

	return err
}
