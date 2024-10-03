package main

import (
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func (g *gateway) startDataHandler() {
	address := fmt.Sprintf(":%v", g.config.DataEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to bind socket: %v", err)
	}

	g.m.Init()

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
			fmt.Println("Finished receiving data")
			return nil
		}
	}
}

func (g *gateway) queueGames(r io.Reader) error {
	csvReader := csv.NewReader(r)
	batch := middleware.BatchGame{}

	for {
		record, err := csvReader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			fmt.Println("Parse error")
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
			continue
		}

		batch.Data = append(batch.Data, game)
		if len(batch.Data) == g.config.BatchSize {
			err = g.m.SendBatchGame(batch, "")
			if err != nil {
				fmt.Println("Could not send batch")
			}
			batch = middleware.BatchGame{}
		}
	}

	if len(batch.Data) != 0 {
		err := g.m.SendBatchGame(batch, "")
		if err != nil {
			fmt.Println("Could not send batch")
		}
	}

	return nil
}

func (g *gateway) queueReviews(r io.Reader) error {
	csvReader := csv.NewReader(r)
	batch := middleware.BatchReview{}

	for {
		record, err := csvReader.Read()
		if errors.Is(err, &csv.ParseError{}) {
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
			continue
		}

		// fmt.Printf("review: %#+v\n", review)

		batch.Data = append(batch.Data, review)
		if len(batch.Data) == g.config.BatchSize {
			err = g.m.SendBatchReview(batch)
			if err != nil {
				fmt.Println("Could not send batch")
			}
			batch = middleware.BatchReview{}
		}
	}

	if len(batch.Data) != 0 {
		err := g.m.SendBatchReview(batch)
		if err != nil {
			fmt.Println("Could not send batch")
		}
	}
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
	game.ReleaseDate = middleware.Date{
		Day:   uint8(releaseDate.Day()),
		Month: uint8(releaseDate.Month()),
		Year:  uint16(releaseDate.Year()),
	}
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
