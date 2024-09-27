package main

import (
	"distribuidos/tp1/middleware"
	"fmt"
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

	m := middleware.NewMessageHandler(conn)

	err = m.SendMessage([]byte("Hello, world!"))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	msg, err := m.ReceiveMessage()
	if err != nil {
		fmt.Println("Error receiving message:", err)
		return
	}

	fmt.Println("Received:", msg)
}
