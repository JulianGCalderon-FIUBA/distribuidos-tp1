package main

import (
	"context"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

const CONFIG_PATH = ".node-config.csv"
const MAX_ATTEMPTS = 3
const MAX_PACKAGE_SIZE = 1024

type restarter struct {
	nodes     map[string]string
	conn      *net.UDPConn
	mu        *sync.Mutex
	gotAckMap map[string]chan bool
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
		gotAckMap: make(map[string]chan bool),
	}, nil
}

func (r *restarter) start(ctx context.Context) error {
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		var err error
		closer := utils.SpawnCloser(ctx, r.conn)
		defer func() {
			closeErr := closer.Close()
			err = errors.Join(err, closeErr)
		}()
		r.readFromSocket()
	}()

	for name := range r.nodes {
		wg.Add(1)
		go func() {
			var err error
			closer := utils.SpawnCloser(ctx, r.conn)
			defer func() {
				closeErr := closer.Close()
				err = errors.Join(err, closeErr)
			}()

			err = r.send(0, name)
			if err != nil {
				log.Errorf("Error monitoring node: %v", err)
			}
		}()
	}

	defer wg.Wait()
	return nil
}

func (r *restarter) readFromSocket() {
	for {
		buf := make([]byte, MAX_PACKAGE_SIZE)
		_, _, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Failed to read: %v", err)
			continue
		}

		header := utils.MsgHeader{}

		n, err := binary.Decode(buf, binary.LittleEndian, &header)
		if err != nil {
			log.Errorf("Failed to decode header: %v", err)
			continue
		}

		nodeName := string(buf[n:])

		log.Infof("Received ack of node %v", nodeName)
		r.mu.Lock()
		ch := r.gotAckMap[nodeName]
		r.mu.Unlock()
		ch <- true
	}
}

func (r *restarter) send(attempts int, nodeName string) error {
	if attempts == MAX_ATTEMPTS {
		// handle crashed node
		return fmt.Errorf("Never got ack")
	}

	msg, err := binary.Append(nil, binary.LittleEndian, utils.MsgHeader{Ty: utils.KeepAlive})
	if err != nil {
		return err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", nodeName, r.nodes[nodeName]))
	if err != nil {
		return err
	}

	n, err := r.conn.WriteToUDP(msg, udpAddr)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("Could not send full message")
	}

	for {
		r.mu.Lock()
		ch := r.gotAckMap[nodeName]
		r.mu.Unlock()
		select {
		case <-ch:
			r.mu.Lock()
			delete(r.gotAckMap, nodeName)
			r.mu.Unlock()
			return nil
		case <-time.After(time.Second):
			log.Infof("Timeout, trying to send KeepAlive again")
			return r.send(attempts+1, nodeName)
		}
	}
}
