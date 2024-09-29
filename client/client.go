package main

import (
	"bufio"
	"distribuidos/tp1/protocol"
	"fmt"
	"log"
	"net"
	"os"
)

const GAMES_PATH = "./client/.data/games.csv"
const REVIEWS_PATH = "./client/.data/reviews.csv"

type FileType int

const (
	GamesFile FileType = iota
	ReviewsFile
)

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

func (c *client) start() {

	err := c.startConnection()
	if err != nil {
		log.Fatalf("Error connecting to connection endpoint: %v", err)
	}
	err = c.startDataConnection()
	if err != nil {
		log.Fatalf("Error connecting to data endpoint: %v", err)
	}
}

func (c *client) startConnection() error {
	conn, err := net.Dial("tcp", c.config.connectionEndpointAddress)
	if err != nil {
		log.Fatalf("Could not connect to connection endpoint: %v", err)
	}
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
		return fmt.Errorf("Could not receive message from connection: %w", err)
	}

	msg, ok := response.(*protocol.AcceptRequest)
	if !ok {
		return fmt.Errorf("Expected AcceptRequest message, received: %T", response)
	}

	c.id = msg.ClientID
	return nil
}

func (c *client) startDataConnection() error {
	dataConn, err := net.Dial("tcp", c.config.dataEndpointAddress)
	if err != nil {
		return fmt.Errorf("Could not connect to data endpoint: %w", err)
	}

	c.dataMarshaller = *protocol.NewMarshaller(dataConn)
	c.dataUnmarshaller = *protocol.NewUnmarshaller(dataConn)

	err = c.sendDataHello()
	if err != nil {
		return err
	}
	err = c.sendFile(GAMES_PATH, GamesFile)
	if err != nil {
		return fmt.Errorf("Error sending games file: %w", err)
	}
	err = c.sendFile(REVIEWS_PATH, ReviewsFile)
	if err != nil {
		return fmt.Errorf("Error sending reviews file: %w", err)
	}
	dataConn.Close()
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

func (c *client) sendFile(filePath string, fileType FileType) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Could not open file %v: %w", filePath, err)
	}
	defer file.Close()

	var batch [][]byte

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(batch) == c.config.packageSize {
			if fileType == GamesFile {
				err = c.sendGames(batch)
				if err != nil {
					fmt.Printf("Could not send batch: %v", err)
				}
			} else if fileType == ReviewsFile {
				err = c.sendReviews(batch)
				if err != nil {
					fmt.Printf("Could not send batch: %v", err)
				}
			}
			batch = [][]byte{}
		}
		line := scanner.Bytes()
		batch = append(batch, line)
	}

	if len(batch) != 0 {
		if fileType == GamesFile {
			err = c.sendGames(batch)
			if err != nil {
				fmt.Printf("Could not send batch: %v", err)
			}
		} else if fileType == ReviewsFile {
			err = c.sendReviews(batch)
			if err != nil {
				fmt.Printf("Could not send batch: %v", err)
			}
		}
	}

	return c.dataMarshaller.SendMessage(&protocol.Finish{})
}

func (c *client) sendGames(batch [][]byte) error {
	games := protocol.GameBatch{Games: batch}
	return c.dataMarshaller.SendMessage(&games)
}

func (c *client) sendReviews(batch [][]byte) error {
	reviews := protocol.ReviewBatch{Reviews: batch}
	return c.dataMarshaller.SendMessage(&reviews)
}

func getFileSize(filePath string) (uint64, error) {
	file, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("Could not get file info: %w", err)
	}
	return uint64(file.Size()), nil
}
