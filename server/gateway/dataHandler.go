package main

import (
	"fmt"
	"log"
	"net"
)

func (g *gateway) startDataHandler() {
	address := fmt.Sprintf(":%v", g.config.dataEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to bind socket: %v", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go func(conn net.Conn) {
			// read ID
			// send OK if valid
			// send ERR if invalid
			// read DATA (Game/Review)
			// send to Rabbit
		}(conn)
	}
}
