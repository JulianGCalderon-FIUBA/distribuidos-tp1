package main

import (
	"context"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

func (g *gateway) startDataHandler(ctx context.Context) (err error) {
	address := fmt.Sprintf(":%v", g.config.DataEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to bind socket: %v", err)
		return
	}

	err = g.m.Init(middleware.DataHandlerexchanges, middleware.DataHandlerQueues)
	defer g.m.Close()
	if err != nil {
		log.Fatalf("Failed to initialize middleware")
		return
	}

	closer := utils.SpawnCloser(ctx, listener)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			return err
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

	var sentGames int
	batch := middleware.Batch[middleware.Game]{
		Data: []middleware.Game{},
		// todo: receive client ID
		ClientID: 1,
		BatchID:  1,
		EOF:      false,
	}

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
			// ignoring known errors to avoid spam
			if err != emptyGameNameError && err != emptyGameGenresError {
				log.Errorf("Failed to parse game: %v", err)
			}
			continue
		}

		batch.Data = append(batch.Data, game)
		sentGames += 1

		if len(batch.Data) == g.config.BatchSize {
			err = g.m.Send(batch, middleware.GamesExchange, "")
			if err != nil {
				log.Errorf("Failed to send games batch: %v", err)
			}
			batch.Data = batch.Data[:0]
			batch.BatchID += 1
		}
	}

	batch.EOF = true
	err := g.m.Send(batch, middleware.GamesExchange, "")
	if err != nil {
		log.Errorf("Failed to send games batch: %v", err)
	}

	log.Infof("Finished sending %v games", sentGames)

	return nil
}

func (g *gateway) queueReviews(r io.Reader) error {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1

	var sentReviews int
	batch := middleware.Batch[middleware.Review]{
		Data: []middleware.Review{},
		// todo: receive client ID
		ClientID: 1,
		BatchID:  1,
		EOF:      false,
	}

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
			// ignoring known errors to avoid spam
			if err != emptyReviewTextError {
				log.Errorf("Failed to parse review: %v", err)
			}
			continue
		}

		batch.Data = append(batch.Data, review)
		sentReviews += 1

		if len(batch.Data) == g.config.BatchSize {
			err = g.m.Send(batch, middleware.ReviewExchange, "")
			if err != nil {
				log.Errorf("Failed to send reviews batch: %v", err)
			}
			batch.Data = batch.Data[:0]
			batch.BatchID += 1
		}
	}

	batch.EOF = true
	err := g.m.Send(batch, middleware.ReviewExchange, "")
	if err != nil {
		log.Errorf("Failed to send reviews batch: %v", err)
	}

	log.Infof("Finished sending %v reviews", sentReviews)

	return nil
}

var emptyGameNameError = errors.New("game name should not be empty")
var emptyGameGenresError = errors.New("game genres should not be empty")
var emptyReviewTextError = errors.New("review text should not be empty")

func gameFromFullRecord(record []string) (game middleware.Game, err error) {
	if len(record) < 37 {
		err = fmt.Errorf("expected 37 fields, got %v", len(record))
		return
	}
	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return
	}
	var releaseDate time.Time
	releaseDate, err = time.Parse("Jan 2, 2006", record[2])
	if err != nil {
		releaseDate, err = time.Parse("Jan 2006", record[2])
		if err != nil {
			return
		}
	}
	averagePlaytimeForever, err := strconv.Atoi(record[29])
	if err != nil {
		return
	}

	game.AppID = uint64(appId)
	game.Name = record[1]
	if game.Name == "" {
		err = emptyGameNameError
		return
	}
	game.ReleaseYear = uint16(releaseDate.Year())
	game.Windows = record[17] == "True"
	game.Mac = record[18] == "True"
	game.Linux = record[19] == "True"
	game.AveragePlaytimeForever = uint64(averagePlaytimeForever)
	if record[36] == "" {
		err = emptyGameGenresError
		return
	}
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
	if review.Text == "" {
		err = emptyReviewTextError
		return
	}
	review.Score = middleware.Score(score)

	return
}
