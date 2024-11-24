package main

import (
	"context"
	"distribuidos/tp1/restarter_protocol"
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
)

const CONFIG_PATH = ".node-config.csv"
const MAX_ATTEMPTS = 3
const MAX_PACKAGE_SIZE = 1024

var ErrFallenNode = errors.New("Never got ack")

type restarter struct {
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

func newRestarter(config config) (*restarter, error) {
	nodes, err := readNodeConfig()
	if err != nil {
		return nil, err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", config.Address)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &restarter{
		nodes:     nodes,
		conn:      conn,
		mu:        &sync.Mutex{},
		ackMap:    make(map[uint64]chan bool),
		lastMsgId: 0,
	}, nil
}

func (r *restarter) start(ctx context.Context) error {
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
		r.readFromSocket()
	}()

	for node := range r.nodes {
		wg.Add(1)
		go func(nodeName string) {
			defer wg.Done()
			time.Sleep(5 * time.Second) // ver como hacer que espere con docker
			err = r.monitorNode(ctx, nodeName)
			if err != nil {
				log.Errorf("Error monitoring node %v: %v", nodeName, err)
			}
		}(node)
	}

	defer wg.Wait()
	return nil
}

func (r *restarter) monitorNode(ctx context.Context, nodeName string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", nodeName, r.nodes[nodeName]))
	if err != nil {
		return fmt.Errorf("Failed to resolve address: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(time.Second) // lo dejo para debuggear, hay que sacarlo
			err = r.safeSend(ctx, restarter_protocol.KeepAlive{}, udpAddr)
			if err != nil {
				if errors.Is(err, ErrFallenNode) {
					err = r.restartNode(ctx, nodeName)
					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("Failed to send keep alive: %v", err)
				}
			}
		}
	}
}

func (r *restarter) readFromSocket() {
	for {
		buf := make([]byte, MAX_PACKAGE_SIZE)
		_, _, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			return
		}

		packet, err := restarter_protocol.Decode(buf)
		if err != nil {
			log.Errorf("Failed to decode packet: %v", err)
			return
		}

		switch packet.Msg.(type) {
		case restarter_protocol.Ack:
			r.handleAck(packet.Id)
		}
	}
}

func (r *restarter) safeSend(ctx context.Context, msg restarter_protocol.Message, addr *net.UDPAddr) error {
	var err error
	msgId := r.newMsgId()

	packet := restarter_protocol.Packet{
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

func (r *restarter) send(ctx context.Context, p restarter_protocol.Packet, addr *net.UDPAddr) error {
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

func (r *restarter) handleAck(msgId uint64) {
	r.mu.Lock()
	ch, ok := r.ackMap[msgId]
	r.mu.Unlock()

	if !ok {
		log.Errorf("Channel for msg %v does not exist", msgId)
	} else {
		ch <- true
	}
}

func (r *restarter) newMsgId() uint64 {
	r.mu.Lock()
	r.lastMsgId += 1
	r.ackMap[r.lastMsgId] = make(chan bool)
	defer r.mu.Unlock()

	return r.lastMsgId
}

func (r *restarter) restartNode(_ context.Context, containerName string) error {
	log.Infof("Restarting [%v]", containerName)
	cmdStr := fmt.Sprintf("docker restart %v", containerName)
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()

	// cmd := exec.Command("docker", "restart", containerName)
	// out, err := cmd.Output()
	log.Infof("Restart output: %s", out)
	if err != nil {
		return err
	}
	return nil
}
