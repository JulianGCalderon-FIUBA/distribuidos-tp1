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

	c.dataMarshaller = *protocol.NewMarshaller(dataConn)
	c.dataUnmarshaller = *protocol.NewUnmarshaller(dataConn)

	err = c.sendDataHello()
	if err != nil {
		fmt.Printf("Could not establish data hello exchange with data endpoint: %v", err)
	}
	c.sendFile(GAMES_PATH)
	c.sendFile(REVIEWS_PATH)
	dataConn.Close()
}

func (c *client) sendDataHello() error {

	msg := protocol.DataHello{
		ClientID: c.id,
	}
	c.dataMarshaller.SendMessage(&msg)

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

func (c *client) sendFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open file %v: %v", filePath, err)
		return
	}
	defer file.Close()

	var pack [][]byte

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(pack) == c.config.packageSize {
			c.sendPackage(pack)
			pack = [][]byte{}
		}

		line := scanner.Bytes()
		pack = append(pack, line)
	}
}

func (c *client) sendPackage(pack [][]byte) {

}

func getFileSize(filePath string) uint64 {
	file, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Could not get file info: %v", err)
	}
	return uint64(file.Size())
}
