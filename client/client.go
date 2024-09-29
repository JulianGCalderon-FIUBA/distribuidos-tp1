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

	c.startConnection()
	c.startDataConnection()
}

func (c *client) startConnection() {
	conn, err := net.Dial("tcp", c.config.connectionEndpointAddress)
	if err != nil {
		log.Fatalf("Could not connect to connection endpoint: %v", err)
	}
	c.conn = conn
	c.connMarshaller = *protocol.NewMarshaller(conn)
	c.connUnmarshaller = *protocol.NewUnmarshaller(conn)

	c.sendRequestHello()
	c.receiveID()
}

func (c *client) sendRequestHello() {

	request := protocol.RequestHello{
		GameSize:   getFileSize(GAMES_PATH),
		ReviewSize: getFileSize(REVIEWS_PATH),
	}

	err := c.connMarshaller.SendMessage(&request)
	if err != nil {
		fmt.Printf("Could not send message: %v", err)
	}
}

func (c *client) receiveID() {

	response, err := c.connUnmarshaller.ReceiveMessage()
	if err != nil {
		fmt.Printf("Could not receive message from connection: %v", err)
	}

	msg, ok := response.(*protocol.AcceptRequest)
	if !ok {
		fmt.Printf("Expected AcceptRequest message, received: %T", response)
	}

	c.id = msg.ClientID
}

func (c *client) startDataConnection() {
	dataConn, err := net.Dial("tcp", c.config.dataEndpointAddress)
	if err != nil {
		log.Fatalf("Could not connect to data endpoint: %v", err)
	}

	c.dataMarshaller = *protocol.NewMarshaller(dataConn)
	c.dataUnmarshaller = *protocol.NewUnmarshaller(dataConn)

	err = c.sendDataHello()
	if err != nil {
		fmt.Printf("Could not establish data hello exchange with data endpoint: %v", err)
	}
	c.sendFile(GAMES_PATH, GamesFile)
	c.sendFile(REVIEWS_PATH, ReviewsFile)
	dataConn.Close()
}

func (c *client) sendDataHello() error {

	msg := protocol.DataHello{
		ClientID: c.id,
	}
	err := c.dataMarshaller.SendMessage(&msg)
	if err != nil {
		return fmt.Errorf("could not send message: %v", err)
	}

	responseAny, err := c.dataUnmarshaller.ReceiveMessage()
	if err != nil {
		return fmt.Errorf("could not receive message from connection: %v", err)
	}
	_, ok := responseAny.(*protocol.DataAccept)
	if !ok {
		return fmt.Errorf("expected DataAccept message, received: %T", responseAny)
	}
	return nil
}

func (c *client) sendFile(filePath string, fileType FileType) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open file %v: %v", filePath, err)
		return
	}
	defer file.Close()

	var batch [][]byte

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(batch) == c.config.packageSize {
			if fileType == GamesFile {
				c.sendGames(batch)
			} else if fileType == ReviewsFile {
				c.sendReviews(batch)
			}
			batch = [][]byte{}
		}
		line := scanner.Bytes()
		batch = append(batch, line)
	}
}

func (c *client) sendGames(batch [][]byte) {
	games := protocol.GameBatch{Games: batch}
	err := c.dataMarshaller.SendMessage(&games)
	if err != nil {
		fmt.Printf("Could not send message: %v", err)
	}
}

func (c *client) sendReviews(batch [][]byte) {
	reviews := protocol.ReviewBatch{Reviews: batch}
	err := c.dataMarshaller.SendMessage(&reviews)
	if err != nil {
		fmt.Printf("Could not send message: %v", err)
	}
}

func getFileSize(filePath string) uint64 {
	file, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Could not get file info: %v", err)
	}
	return uint64(file.Size())
}
