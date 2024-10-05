package main

import (
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

func (g *gateway) startDataHandler() {
	address := fmt.Sprintf(":%v", g.config.DataEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to bind socket: %v", err)
	}

	err = g.m.Init(middleware.DataHandlerexchanges, middleware.DataHandlerQueues)
	if err != nil {
		log.Fatalf("Failed to initialize middleware")
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()
			err := g.handleClientData(conn)
			if err != nil {
				log.Errorf("Error while handling client: %v", err)
			}
		}(conn)
	}
}

func (g *gateway) handleClientData(netConn net.Conn) error {
	conn := protocol.NewConn(netConn)

	var hello protocol.DataHello
	err := conn.Recv(&hello)
	if err != nil {
		return err
	}

	// todo: validate client id
	log.Infof("Client data hello with id: %v", hello.ClientID)

	err = conn.Send(&protocol.DataAccept{})
	if err != nil {
		return err
	}

	gamesRecv, gamesSend := net.Pipe()
	go func() {
		err := g.queueGames(gamesRecv)
		if err != nil {
			log.Errorf("Error while queuing games: %v", err)
		}
	}()
	err = g.receiveData(conn, gamesSend)
	if err != nil {
		return err
	}
	gamesSend.Close()

	reviewsRecv, reviewsSend := net.Pipe()
	go func() {
		err := g.queueReviews(reviewsRecv)
		if err != nil {
			log.Errorf("Error while queuing reviews: %v", err)
		}
	}()
	err = g.receiveData(conn, reviewsSend)
	if err != nil {
		return err
	}
	reviewsSend.Close()

	return nil
}

func (g *gateway) receiveData(unm *protocol.Conn, w io.Writer) error {
	for {
		var anyMsg any
		err := unm.Recv(&anyMsg)
		if err != nil {
			return err
		}

		switch msg := anyMsg.(type) {
		case protocol.Batch:
			_, err := w.Write(msg.Data)
			if err != nil {
				return err
			}
		case protocol.Finish:
			return nil
		}
	}
}

func (g *gateway) queueGames(r io.Reader) error {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1
	var batch middleware.Batch[middleware.Game]

	var sentGames int

	_, _ = csvReader.Read()

	for {
		record, err := csvReader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			log.Errorf("Failed to parse row: %v", err)
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		game, err := gameFromFullRecord(record)
		if err != nil {
			log.Errorf("Failed to parse game: %v", err)
			continue
		}

		batch = append(batch, game)
		sentGames += 1

		if len(batch) == g.config.BatchSize {
			err = g.m.Publish(batch, middleware.GamesExchange, "")
			if err != nil {
				log.Errorf("Failed to send games batch: %v", err)
			}
			batch = batch[:0]
		}
	}

	if len(batch) != 0 {
		err := g.m.Publish(batch, middleware.GamesExchange, "")
		if err != nil {
			log.Errorf("Failed to send games batch: %v", err)
		}
	}

	log.Infof("Finished sending %v games", sentGames)

	return nil
}

func (g *gateway) queueReviews(r io.Reader) error {
	csvReader := csv.NewReader(r)
	var batch middleware.Batch[middleware.Review]

	var sentReviews int

	_, _ = csvReader.Read()

	for {
		record, err := csvReader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			log.Errorf("Failed to parse row: %v", err)
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		review, err := reviewFromFullRecord(record)
		if err != nil {
			log.Errorf("Failed to parse review: %v", err)
			continue
		}

		batch = append(batch, review)
		sentReviews += 1

		if len(batch) == g.config.BatchSize {
			err = g.m.Publish(batch, middleware.ReviewExchange, "")
			if err != nil {
				log.Errorf("Failed to send reviews batch: %v", err)
			}
			batch = batch[:0]
		}
	}

	if len(batch) != 0 {
		err := g.m.Publish(batch, middleware.ReviewExchange, "")
		if err != nil {
			log.Errorf("Failed to send reviews batch: %v", err)
		}
	}

	log.Infof("Finished sending %v reviews", sentReviews)

	return nil
}

func gameFromFullRecord(record []string) (game middleware.Game, err error) {
	if len(record) < 37 {
		err = fmt.Errorf("expected 37 fields, got %v", len(record))
		return
	}
	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return
	}
	releaseDate, err := time.Parse("Jan 2, 2006", record[2])
	if err != nil {
		return
	}
	averagePlaytimeForever, err := strconv.Atoi(record[29])
	if err != nil {
		return
	}

	game.AppID = uint64(appId)
	game.Name = record[1]
	game.ReleaseYear = uint16(releaseDate.Year())
	game.Windows = record[17] == "true"
	game.Mac = record[18] == "true"
	game.Linux = record[19] == "true"
	game.AveragePlaytimeForever = uint64(averagePlaytimeForever)
	game.Genres = strings.Split(record[36], ",")

	return
}

func reviewFromFullRecord(record []string) (review middleware.Review, err error) {
	if len(record) < 4 {
		err = fmt.Errorf("expected 4 fields, got %v", len(record))
		return
	}
	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return
	}
	score, err := strconv.Atoi(record[3])
	if err != nil {
		return
	}

	review.AppID = uint64(appId)
	review.Text = record[2]
	review.Score = middleware.Score(score)

	return
}
