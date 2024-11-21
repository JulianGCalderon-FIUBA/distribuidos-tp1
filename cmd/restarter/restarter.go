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
)

const CONFIG_PATH = ".node-config.csv"

type restarter struct {
	address string
	nodes   map[string]uint64
}

func newRestarter(config config) (*restarter, error) {
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
		address: config.Address,
		nodes:   nodes,
	}, nil
}

func (r *restarter) start(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", r.address)
	if err != nil {
		return fmt.Errorf("failed to resolve address %w:", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", r.address, err)
	}

	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	wg := &sync.WaitGroup{}
	// for node, port := range r.nodes {
	// nodeAddr := fmt.Sprintf("%v:%d", node, port)

	udpAddr, err = net.ResolveUDPAddr("udp", "genre-filter-1:7000")
	if err != nil {
		return err
	}
	log.Infof("Monitoring node %v", udpAddr)
	_, err = conn.WriteToUDP([]byte("Hello node!!\n"), udpAddr)
	if err != nil {
		log.Errorf("Write error: %v", err)
		return err
	}

	// go func(conn *net.UDPConn) {
	// 	err = r.monitorNode(conn, nodeAddr)
	// 	if err != nil {
	// 		log.Errorf("Error monitoring node: %v", err)
	// 	}
	// }(conn)
	// }

	defer wg.Wait()
	return nil
}

func (r *restarter) monitorNode(conn *net.UDPConn, nodeName string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", nodeName)
	if err != nil {
		return err
	}
	log.Infof("Monitoring node %v", udpAddr)
	_, err = conn.WriteToUDP([]byte("Hello node!!\n"), udpAddr)
	if err != nil {
		log.Errorf("Write error: %v", err)
		return err
	}

	buf := make([]byte, 1024)
	for {
		nRead, raddr, err := conn.ReadFromUDP(buf)

		if err != nil {
			log.Errorf("Read error: %v", err)
			return err
		}

		msg := string(buf[:nRead])
		log.Infof("Received: %s", msg)

		aliveMsg := "ALIVE"
		_, err = conn.WriteToUDP([]byte(aliveMsg), raddr)
		if err != nil {
			log.Errorf("Error sending message: %v", err)
		}
	}
}
