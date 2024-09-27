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
		fmt.Println("Error:", err)
		return
	}
	defer conn.Close()

	m := middleware.NewMessageHandler(conn)
	m.SendMessage([]byte("Hello, world!"))

	msg, err := m.ReceiveMessage()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Received:", msg)
}
