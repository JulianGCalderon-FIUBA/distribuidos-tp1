package main

import (
	"bufio"
	"distribuidos/tp1/protocol"
	"fmt"
	"log"
	"net"
	"os"
)

const GAMES_PATH = "./.data/games.csv"
const REVIEWS_PATH = "./.data/reviews.csv"
const MAX_PAYLOAD_SIZE = 8096
const MSG_SIZE = 4
const TAG_SIZE = 4

type client struct {
	config           config
	id               uint64
	conn             net.Conn
	connMarshaller   protocol.Marshaller
	connUnmarshaller protocol.Unmarshaller
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

	c.connMarshaller.SendMessage(&request)
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

	dataMarshaller := *protocol.NewMarshaller(dataConn)
	dataUnmarshaller := *protocol.NewUnmarshaller(dataConn)

	err = c.sendDataHello(dataMarshaller, dataUnmarshaller)
	if err != nil {
		fmt.Printf("Could not establish connection with data endpoint: %v", err)
	}
	sendFile(GAMES_PATH, dataMarshaller, dataUnmarshaller)
	sendFile(REVIEWS_PATH, dataMarshaller, dataUnmarshaller)
	dataConn.Close()
}

func (c *client) sendDataHello(marshaller protocol.Marshaller, unmarshaller protocol.Unmarshaller) error {

	msg := protocol.DataHello{
		ClientID: c.id,
	}
	marshaller.SendMessage(&msg)

	responseAny, err := unmarshaller.ReceiveMessage()
	if err != nil {
		return fmt.Errorf("could not receive message from connection: %v", err)
	}
	_, ok := responseAny.(*protocol.DataAccept)
	if !ok {
		return fmt.Errorf("expected DataAccept message, received: %T", responseAny)
	}
	return nil
}

func sendFile(filePath string, marshaller protocol.Marshaller, unmarshaller protocol.Unmarshaller) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open file %v: %v", filePath, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

	}
}

func getFileSize(filePath string) uint64 {
	file, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Could not get file info: %v", err)
	}
	return uint64(file.Size())
}
