package main

import (
	"bufio"
	"distribuidos/tp1/middleware"
	"encoding/binary"
	"fmt"
	"io"
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
	config     config
	id         int
	conn       net.Conn
	connReader bufio.Reader
	connWriter bufio.Writer
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
	c.connReader = *bufio.NewReader(c.conn)
	c.connWriter = *bufio.NewWriterSize(c.conn, c.config.buffSize)

	c.sendRequestHello()
	c.receiveID()
}

func (c *client) sendRequestHello() {
	// está bien que sea el mismo buffer? desde el otro lado saben cuándo empieza un número y cuándo termina el otro? supongo que sí porque son uint32?
	buf := make([]byte, MAX_PAYLOAD_SIZE)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(getFileSize(GAMES_PATH)))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(getFileSize(REVIEWS_PATH)))

	size := int32(len(buf))
	sendHeader(size, middleware.RequestHelloTag, &c.connWriter)
	_, err := c.connWriter.Write(buf) // habria que handlear o asumimos que no hay short write?
	if err != nil {
		fmt.Printf("Could not write message: %v", err)
	}
	c.connWriter.Flush()
}

func (c *client) receiveID() {

	size, tag, err := readHeader(&c.connReader)
	if err != nil || tag != middleware.AcceptRequestTag {
		fmt.Printf("Did not receive expected message, don't have id")
		c.close()
		return
	}
	buf := make([]byte, size)
	_, err = io.ReadFull(&c.connReader, buf)
	if err != nil {
		fmt.Printf("Could not read valid id: %v", err)
		c.close()
		return
	}

	binary.Decode(buf, binary.LittleEndian, &c.id)
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

func (c *client) close() {
	c.conn.Close()
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

func readHeader(reader io.Reader) (int32, middleware.MessageTag, error) {

	var header struct {
		size int32
		tag  middleware.MessageTag
	}
	bytes := make([]byte, MSG_SIZE+TAG_SIZE)
	_, err := io.ReadFull(reader, bytes)
	if err != nil {
		fmt.Printf("Could not read header: %v", err)
		return 0, 0, err
	}
	binary.Decode(bytes, binary.LittleEndian, &header)

	return header.size, header.tag, nil
}

func sendHeader(size int32, tag middleware.MessageTag, writer io.Writer) {
	err := binary.Write(writer, binary.LittleEndian, size)
	if err != nil {
		fmt.Printf("Failed to write size")
	}
	err = binary.Write(writer, binary.LittleEndian, int32(tag))
	if err != nil {
		fmt.Printf("Failed to write tag")
	}
}

func getFileSize(filePath string) uint64 {
	file, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Could not get file info: %v", err)
	}
	return uint64(file.Size())
}
