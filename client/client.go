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
	conn, err := net.Dial("tcp", c.config.dataEndpointAddress)
	if err != nil {
		log.Fatalf("Could not connect to data endpoint: %v", err)
	}
	// send data hello
	// receive data accept
	/*
		1. enviar juegos
		2. enviar reviews
		3. cerrar conexión
	*/
	sendFile(GAMES_PATH)
	sendFile(REVIEWS_PATH)
	conn.Close()
}

func sendFile(filePath string) {
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
