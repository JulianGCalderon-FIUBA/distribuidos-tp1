package main

import (
	"distribuidos/tp1/protocol"
	"log"
)

type config struct {
	connectionEndpointAddress string
	dataEndpointAddress       string
	buffSize                  int
}

func getConfig() (config, error) {
	// todo: read from file
	return config{
		connectionEndpointAddress: "127.0.0.1:9001",
		dataEndpointAddress:       "127.0.0.1:9002",
		buffSize:                  8096,
	}, nil
}
	"net"
)

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	// cliente basico para testear, a refactorizar
	conn, err := net.Dial("tcp", "localhost:9001")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	m := protocol.NewMarshaller(conn)
	unm := protocol.NewUnmarshaller(conn)

	err = m.SendMessage(&protocol.RequestHello{
		GameSize:   1,
		ReviewSize: 6,
	})
	if err != nil {
		log.Fatalf("failed to send message %v", err)
	}
	client := newClient(config)
	client.start()

	msg, err := unm.ReceiveMessage()
	if err != nil {
		log.Fatalf("failed to receive message %v", err)
	}

	log.Printf("received %v", msg)
}
