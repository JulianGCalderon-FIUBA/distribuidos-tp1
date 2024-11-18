package main

import (
	"context"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const CONFIG_PATH = ".node-config.csv"

type restarter struct {
	nodes map[string]uint64
}

func newRestarter() (*restarter, error) {
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

	var nodes = make(map[string]uint64)

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

		port, err := strconv.Atoi(node[1])
		if err != nil {
			return nil, err
		}
		nodes[node[0]] = uint64(port)
	}

	return &restarter{
		nodes: nodes,
	}, nil
}

func (r *restarter) start(ctx context.Context) error {
	time.Sleep(500 * time.Millisecond)
	log.Infof("Starting connections")
	wg := &sync.WaitGroup{}

	for _, port := range r.nodes {
		udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return fmt.Errorf("failed to get udp address")
		}

		conn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", udpAddr, err)
		}

		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()

			closer := utils.SpawnCloser(ctx, conn)
			defer func() {
				closeErr := closer.Close()
				err = errors.Join(err, closeErr)
			}()

			if err := r.monitorNode(conn); err != nil {
				fmt.Printf("Error monitoring node %s: %v\n", conn.RemoteAddr().String(), err)
			}
		}(conn)
	}

	defer wg.Wait()
	return nil
}

func (r *restarter) monitorNode(conn net.Conn) error {
	_, err := conn.Write([]byte("Hello node!!\n"))
	if err != nil {
		log.Errorf("Error writing connection: %v", err)
		return err
	}

	buf := make([]byte, 1024)
	for {
		nRead, err := conn.Read(buf)
		if err != nil {
			log.Errorf("Error reading from connection: %v", err)
			return err
		}

		msg := string(buf[:nRead])
		log.Infof("Received: %s", msg)

		aliveMsg := "ALIVE"
		_, err = conn.Write([]byte(aliveMsg))
		if err != nil {
			log.Errorf("Error sending ACK: %v", err)
		}
	}
}
