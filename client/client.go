package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"io"
	"net"
	"os"
)

const GAMES_PATH = ".data/games.csv"
const REVIEWS_PATH = ".data/reviews.csv"

type client struct {
	config   config
	id       uint64
	reqConn  *protocol.Conn
	dataConn *protocol.Conn
}

func newClient(config config) *client {
	protocol.Register()
	return &client{
		config: config,
	}
}

// Connects client to connection endpoint and data endpoint
func (c *client) start() error {
	if err := c.startConnection(); err != nil {
		return err
	}

	if err := c.startDataConnection(); err != nil {
		return err
	}

	return nil
}

// Starts connection with connection endpoint, sending request hello and waiting for id
func (c *client) startConnection() error {
	conn, err := net.Dial("tcp", c.config.ConnectionEndpointAddress)
	if err != nil {
		return err
	}
	c.reqConn = protocol.NewConn(conn)

	log.Info("Connected to connection gateway")

	err = c.sendRequestHello()
	if err != nil {
		return fmt.Errorf("failed to send hello: %w", err)
	}
	err = c.receiveID()
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

	return c.reqConn.Send(&request)
}

func (c *client) receiveID() error {
	var msg protocol.AcceptRequest
	err := c.reqConn.Recv(&msg)
	if err != nil {
		return fmt.Errorf("could not receive id from gateway: %w", err)
	}

	c.id = msg.ClientID
	log.Infof("Received ID: %v", c.id)
	return nil
}

// Starts connection with data endpoint and sends games and reviews files. When done closes connection
func (c *client) startDataConnection() error {
	dataConn, err := net.Dial("tcp", c.config.DataEndpointAddress)
	if err != nil {
		return err
	}

	log.Info("Connected to data gateway")

	c.dataConn = protocol.NewConn(dataConn)
	defer c.dataConn.Close()

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

func getFileSize(filePath string) (uint64, error) {
	file, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return uint64(file.Size()), nil
}
