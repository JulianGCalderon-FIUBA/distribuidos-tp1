package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const GAMES_PATH = "./.data/games.csv"
const REVIEWS_PATH = "./.data/reviews.csv"

type client struct {
	config           config
	id               uint64
	conn             net.Conn
	connMarshaller   protocol.Marshaller
	connUnmarshaller protocol.Unmarshaller
	dataMarshaller   protocol.Marshaller
	dataUnmarshaller protocol.Unmarshaller
}

func newClient(config config) *client {
	client := &client{
		config: config,
	}
	return client
}

// Connects client to connection endpoint and data endpoint
func (c *client) start() {
	if err := c.startConnection(); err != nil {
		log.Fatalf("Error connecting to connection endpoint: %v", err)
	}

	if err := c.startDataConnection(); err != nil {
		log.Fatalf("Error connecting to data endpoint: %v", err)
	}
}

// Starts connection with connection endpoint, sending request hello and waiting for id
func (c *client) startConnection() error {
	conn, err := net.Dial("tcp", c.config.ConnectionEndpointAddress)
	if err != nil {
		log.Fatalf("Could not connect to connection endpoint: %v", err)
	}
	// todo: remove when receiving results
	defer conn.Close()

	c.conn = conn
	c.connMarshaller = *protocol.NewMarshaller(conn)
	c.connUnmarshaller = *protocol.NewUnmarshaller(conn)

	err = c.sendRequestHello()
	if err != nil {
		return err
	}
	return c.receiveID()
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

	return c.connMarshaller.SendMessage(&request)

}

func (c *client) receiveID() error {

	response, err := c.connUnmarshaller.ReceiveMessage()
	if err != nil {
		return fmt.Errorf("could not receive message from connection: %w", err)
	}

	msg, ok := response.(*protocol.AcceptRequest)
	if !ok {
		return fmt.Errorf("expected AcceptRequest message, received: %T", response)
	}

	c.id = msg.ClientID
	fmt.Printf("Received ID: %v\n", c.id)
	return nil
}

// Starts connection with data endpoint and sends games and reviews files. When done closes connection
func (c *client) startDataConnection() error {
	dataConn, err := net.Dial("tcp", c.config.DataEndpointAddress)
	if err != nil {
		return fmt.Errorf("could not connect to data endpoint: %w", err)
	}
	defer dataConn.Close()

	c.dataMarshaller = *protocol.NewMarshaller(dataConn)
	c.dataUnmarshaller = *protocol.NewUnmarshaller(dataConn)

	err = c.sendDataHello()
	if err != nil {
		return err
	}

	err = c.sendFile(GAMES_PATH)
	if err != nil {
		return fmt.Errorf("error sending games file: %w", err)
	}
	err = c.sendFile(REVIEWS_PATH)
	if err != nil {
		return fmt.Errorf("error sending reviews file: %w", err)
	}
	return nil
}

func (c *client) sendDataHello() error {

	msg := protocol.DataHello{
		ClientID: c.id,
	}
	err := c.dataMarshaller.SendMessage(&msg)
	if err != nil {
		return fmt.Errorf("could not send message: %w", err)
	}

	responseAny, err := c.dataUnmarshaller.ReceiveMessage()
	if err != nil {
		return fmt.Errorf("could not receive message from connection: %w", err)
	}
	_, ok := responseAny.(*protocol.DataAccept)
	if !ok {
		return fmt.Errorf("expected DataAccept message, received: %T", responseAny)
	}
	return nil
}

// Sends specified file in different batches of size obtained from config
func (c *client) sendFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file %v: %w", filePath, err)
	}
	defer file.Close()

	buf := make([]byte, c.config.BatchSize)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			sendErr := c.dataMarshaller.SendMessage(&protocol.Batch{Data: buf[:n]})
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

	return c.dataMarshaller.SendMessage(&protocol.Finish{})
}

func getFileSize(filePath string) (uint64, error) {
	file, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("could not get file info: %w", err)
	}
	return uint64(file.Size()), nil
}
