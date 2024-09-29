package main

import (
	"bytes"
	"distribuidos/tp1/protocol"
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"strconv"
)

const ReviewExchange string = "reviews"
const GamesExchange string = "games"

func (g *gateway) startDataHandler() {
	address := fmt.Sprintf(":%v", g.config.dataEndpointPort)
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
		ReviewExchange,
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
		GamesExchange,
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

	anyMsg, err = unm.ReceiveMessage()
	if err != nil {
		return err
	}
	_, ok = anyMsg.(*protocol.Prepare)
	if !ok {
		return fmt.Errorf("expected Prepare message, received %T", anyMsg)
	}

gameLoop:
	for {

		msg, err := unm.ReceiveMessage()
		if err != nil {
			return err
		}

		switch msg := msg.(type) {
		case *protocol.Batch:
		case *protocol.Finish:
			break gameLoop
		default:
			return fmt.Errorf("expected Batch or Finish message, received %T", msg)
		}
	}

	anyMsg, err = unm.ReceiveMessage()
	if err != nil {
		return err
	}
	_, ok = anyMsg.(*protocol.Prepare)
	if !ok {
		return fmt.Errorf("expected Prepare message, received %T", anyMsg)
	}

reviewLoop:
	for {
		msg, err := unm.ReceiveMessage()
		if err != nil {
			return err
		}

		switch msg := msg.(type) {
		case *protocol.Batch:
		case *protocol.Finish:
			break reviewLoop
		default:
			return fmt.Errorf("expected Batch or Finish message, received %T", msg)
		}
	}

	return nil
}

type Game struct {
	AppID                  int
	Name                   string
	ReleaseDate            string
	Windows                bool
	Mac                    bool
	Linux                  bool
	AveragePlaytimeForever int
	Genres                 string
}

type Review struct {
	AppID   int
	AppName string
	Text    string
	Score   int
}

func parseGame(line []byte) (Game, error) {
	game := Game{}

	buf := bytes.NewReader(line)
	csv := csv.NewReader(buf)
	record, err := csv.Read()
	if err != nil {
		return game, err
	}

	if len(record) < 37 {
		return game, fmt.Errorf("expected record of length 4, but got %v", len(record))
	}

	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return game, err
	}
	game.AppID = appId

	game.Name = record[1]
	game.ReleaseDate = record[2]
	game.Windows = record[17] == "true"
	game.Mac = record[18] == "true"
	game.Linux = record[19] == "true"

	averagePlaytimeForever, err := strconv.Atoi(record[29])
	if err != nil {
		return game, err
	}
	game.AveragePlaytimeForever = averagePlaytimeForever

	game.Genres = record[36]

	return game, nil
}

func parseReview(line []byte) (Review, error) {
	review := Review{}

	buf := bytes.NewReader(line)
	csv := csv.NewReader(buf)
	record, err := csv.Read()
	if err != nil {
		return review, err
	}

	if len(record) < 4 {
		return review, fmt.Errorf("expected record of length 4, but got %v", len(record))
	}

	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return review, err
	}
	review.AppID = appId

	review.AppName = record[1]
	review.Text = record[2]

	score, err := strconv.Atoi(record[3])
	if err != nil {
		return review, err
	}
	review.Score = score

	return review, nil
}
