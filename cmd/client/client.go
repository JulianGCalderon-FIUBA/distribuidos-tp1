package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

const GAMES_PATH = ".data/games.csv"
const REVIEWS_PATH = ".data/reviews.csv"
const RESULTS_PATH = ".results"
const MAX_RESULTS = 5

type client struct {
	config   config
	id       uint64
	conn     *protocol.Conn
	dataConn *protocol.Conn
	results  map[int]bool
}

func newClient(config config) *client {
	protocol.Register()
	return &client{
		config:  config,
		results: make(map[int]bool),
	}
}

// Connects client to connection endpoint and data endpoint
func (c *client) start(ctx context.Context) (err error) {
	err = c.startConnection()
	if err != nil {
		return
	}
	closer := utils.SpawnCloser(ctx, c.conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	if err = c.sendRequest(); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.sendData(ctx); err != nil {
			log.Errorf("Failed to send data: %v", err)
		}
	}()
	defer wg.Wait()

	if err = c.waitResults(); err != nil {
		return fmt.Errorf("failed to wait results: %w", err)
	}

	return nil
}

func (c *client) startConnection() error {
	conn, err := net.Dial("tcp", c.config.ConnectionEndpointAddress)
	if err != nil {
		return err
	}
	c.conn = protocol.NewConn(conn)
	log.Info("Connected to connection gateway")
	return nil
}

// Starts connection with connection endpoint, sending request hello and waiting for id
func (c *client) sendRequest() error {
	err := c.sendRequestHello()
	if err != nil {
		return fmt.Errorf("failed to send hello: %w", err)
	}
	err = c.waitID()
	if err != nil {
		return fmt.Errorf("failed to receive id: %w", err)
	}

	return nil
}

func (c *client) sendRequestHello() error {
	gameSize, err := getFileSize(GAMES_PATH)
	if err != nil {
		return err
	}
	reviewsSize, err := getFileSize(REVIEWS_PATH)
	if err != nil {
		return err
	}

	request := protocol.RequestHello{
		GameSize:   gameSize,
		ReviewSize: reviewsSize,
	}

	return c.conn.Send(&request)
}

func (c *client) waitID() error {
	var msg protocol.AcceptRequest
	err := c.conn.Recv(&msg)
	if err != nil {
		return fmt.Errorf("could not receive id from gateway: %w", err)
	}

	c.id = msg.ClientID
	log.Infof("Received ID: %v", c.id)
	return nil
}

// Starts connection with data endpoint and sends games and reviews files. When done closes connection
func (c *client) sendData(ctx context.Context) (err error) {
	err = c.startDataConnection()
	if err != nil {
		return
	}

	closer := utils.SpawnCloser(ctx, c.dataConn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	err = c.sendDataHello()
	if err != nil {
		return fmt.Errorf("failed to send data hello: %w", err)
	}
	err = c.sendFile(GAMES_PATH)
	if err != nil {
		return fmt.Errorf("failed to send games: %w", err)
	}
	log.Info("Sent all games")
	err = c.sendFile(REVIEWS_PATH)
	if err != nil {
		return fmt.Errorf("failed to send reviews: %w", err)
	}
	log.Info("Sent all reviews")

	return nil
}

func (c *client) startDataConnection() error {
	dataConn, err := net.Dial("tcp", c.config.DataEndpointAddress)
	if err != nil {
		return err
	}
	log.Info("Connected to data gateway")
	c.dataConn = protocol.NewConn(dataConn)
	return nil
}

func (c *client) sendDataHello() error {
	hello := protocol.DataHello{
		ClientID: c.id,
	}
	err := c.dataConn.Send(&hello)
	if err != nil {
		return err
	}

	var accept protocol.DataAccept
	return c.dataConn.Recv(&accept)
}

// Sends specified file in different batches of size obtained from config
func (c *client) sendFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, c.config.BatchSize)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			sendErr := c.dataConn.SendAny(&protocol.Batch{Data: buf[:n]})
			if sendErr != nil {
				return sendErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return c.dataConn.SendAny(&protocol.Finish{})
}

func (c *client) waitResults() error {
	writers, err := initResultWriters()
	if err != nil {
		return err
	}

	for {
		var r protocol.Result
		err := c.conn.Recv(&r)
		if err != nil {
			return err
		}

		switch r := r.(type) {
		case protocol.Q4Finish:
			log.Infof("Received Q4 Finish")
			c.results[r.Number()] = true
		default:
			log.Infof("Received Q%v results", r.Number())
			if r.Number() != 4 {
				c.results[r.Number()] = true
			}
			writer := writers[r.Number()-1]

			err = writer.WriteAll(r.ToCSV())
			if err != nil {
				return err
			}
		}

		if len(c.results) == MAX_RESULTS {
			log.Infof("Received all results")
			break
		}
	}

	return nil
}

func getFileSize(filePath string) (uint64, error) {
	file, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return uint64(file.Size()), nil
}

func initResultWriter(q protocol.Result) (*csv.Writer, error) {
	n := q.Number()
	path := fmt.Sprintf("%v/%v.csv", RESULTS_PATH, n)
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	writer := csv.NewWriter(file)
	_ = writer.Write(q.Header())
	writer.Flush()
	err = writer.Error()
	return writer, err
}

func initResultWriters() ([]*csv.Writer, error) {
	writers := make([]*csv.Writer, MAX_RESULTS)
	for i, result := range protocol.AllResultTypes() {
		writer, err := initResultWriter(result)
		if err != nil {
			return nil, err
		}

		writers[i] = writer
	}

	return writers, nil
}
