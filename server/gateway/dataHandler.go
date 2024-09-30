package main

import (
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server"
	"distribuidos/tp1/server/middleware"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

func (g *gateway) startDataHandler() {
	address := fmt.Sprintf(":%v", g.config.DataEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to bind socket: %v", err)
	}

	g.initRabbit()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()
			err := g.handleClientData(conn)
			if err != nil {
				log.Printf("error while handling client: %v", err)
			}
		}(conn)
	}
}

func (g *gateway) initRabbit() {
	rabbitChan, err := g.rabbitConn.Channel()
	if err != nil {
		log.Fatalf("failed to bind rabbit connection: %v", err)
	}

	err = rabbitChan.ExchangeDeclare(
		middleware.ReviewExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare reviews exchange: %v", err)
	}
	err = rabbitChan.ExchangeDeclare(
		middleware.GamesExchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to bind games exchange: %v", err)
	}
}

func (g *gateway) handleClientData(conn net.Conn) error {
	m := protocol.NewMarshaller(conn)
	unm := protocol.NewUnmarshaller(conn)

	anyMsg, err := unm.ReceiveMessage()
	if err != nil {
		return err
	}
	helloMsg, ok := anyMsg.(*protocol.DataHello)
	if !ok {
		return fmt.Errorf("expected DataHello message, received %T", anyMsg)
	}

	// todo: validate client id
	_ = helloMsg

	err = m.SendMessage(&protocol.DataAccept{})
	if err != nil {
		return err
	}

	gamesRecv, gamesSend := net.Pipe()
	go func() {
		err := g.queueGames(gamesRecv)
		if err != nil {
			fmt.Printf("error while queuing games: %v", err)
		}
	}()
	err = g.receiveData(unm, gamesSend)
	if err != nil {
		return err
	}

	reviewsRecv, reviewsSend := net.Pipe()
	go func() {
		err := g.queueReviews(reviewsRecv)
		if err != nil {
			fmt.Printf("error while queuing games: %v", err)
		}
	}()
	err = g.receiveData(unm, reviewsSend)
	if err != nil {
		return err
	}

	return nil
}

func (g *gateway) receiveData(unm *protocol.Unmarshaller, w io.Writer) error {
	anyMsg, err := unm.ReceiveMessage()
	if err != nil {
		return err
	}
	_, ok := anyMsg.(*protocol.Prepare)
	if !ok {
		return fmt.Errorf("expected Prepare message, received %T", anyMsg)
	}

	for {
		anyMsg, err := unm.ReceiveMessage()
		if err != nil {
			return err
		}

		switch msg := anyMsg.(type) {
		case *protocol.Batch:
			_, err := w.Write(msg.Data)
			if err != nil {
				return err
			}
		case *protocol.Finish:
			return nil
		}
	}
}

func (g *gateway) queueGames(r io.Reader) error {
	csvReader := csv.NewReader(r)

	for {
		record, err := csvReader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		game, err := server.GameFromFullRecord(record)
		if err != nil {
			continue
		}

		fmt.Printf("Game: %#+v\n", game)
	}
}

func (g *gateway) queueReviews(r io.Reader) error {
	csvReader := csv.NewReader(r)

	for {
		record, err := csvReader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		review, err := server.ReviewFromFullRecord(record)
		if err != nil {
			continue
		}

		fmt.Printf("review: %#+v\n", review)
	}
}
