package main

import (
	"distribuidos/tp1/protocol"
	"fmt"
	"log"
	"net"
)

func main() {
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

	msg, err := unm.ReceiveMessage()
	if err != nil {
		log.Fatalf("failed to receive message %v", err)
	}

	log.Printf("received %v", msg)
}
