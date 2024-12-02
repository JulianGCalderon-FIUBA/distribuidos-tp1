package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (g *gateway) notifyFallenNode(outputs []middleware.Output, clientID int, cleanAction int) error {
	rawCh, err := g.rabbit.Channel()
	if err != nil {
		return err
	}
	err = rawCh.Confirm(false)
	if err != nil {
		return err
	}

	ch := middleware.Channel{
		Ch:          rawCh,
		ClientID:    clientID,
		FinishFlag:  false,
		CleanAction: cleanAction,
	}

	for _, output := range outputs {
		for _, k := range output.Keys {
			err := ch.Send([]byte{}, output.Exchange, k)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *gateway) startDataEndpoint(ctx context.Context) (err error) {
	address := fmt.Sprintf(":%v", g.config.DataEndpointPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	closer := utils.SpawnCloser(ctx, listener)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	topology := middleware.Topology{
		Exchanges: []middleware.ExchangeConfig{
			{Name: middleware.ExchangeGames, Type: amqp.ExchangeFanout},
			{Name: middleware.ExchangeReviews, Type: amqp.ExchangeFanout},
		},
		Queues: []middleware.QueueConfig{
			{Name: middleware.GamesQ1,
				Bindings: map[string][]string{middleware.ExchangeGames: {""}}},
			{Name: middleware.GamesGenre,
				Bindings: map[string][]string{middleware.ExchangeGames: {""}}},
			{Name: middleware.ReviewsScore,
				Bindings: map[string][]string{middleware.ExchangeReviews: {""}}},
		},
	}

	outputs := []middleware.Output{
		{
			Exchange: middleware.ExchangeGames,
			Keys:     []string{""},
		},
		{
			Exchange: middleware.ExchangeReviews,
			Keys:     []string{""},
		},
	}

	rawCh, err := g.rabbit.Channel()
	if err != nil {
		return err
	}
	err = topology.Declare(rawCh)
	if err != nil {
		return err
	}

	err = g.notifyFallenNode(outputs, int(g.clientCounter), middleware.CleanAll)

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	log.Infof("Starting accepting data connections")

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := g.handleClientData(ctx, conn)
			if err != nil {
				log.Errorf("Error while handling client: %v", err)
			}
		}()
	}
}

func (g *gateway) handleClientData(ctx context.Context, rawConn net.Conn) (err error) {
	conn := protocol.NewConn(rawConn)
	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()

	var hello protocol.DataHello
	err = conn.Recv(&hello)
	if err != nil {
		return err
	}

	log.Infof("Client data hello with id: %v", hello.ClientID)
	g.mu.Lock()
	if _, ok := g.clients[int(hello.ClientID)]; !ok {
		return fmt.Errorf("Client ID received is unknown")
	}
	g.mu.Unlock()

	err = conn.Send(&protocol.DataAccept{})
	if err != nil {
		return err
	}

	rawCh, err := g.rabbit.Channel()
	if err != nil {
		return err
	}
	err = rawCh.Confirm(false)
	if err != nil {
		return err
	}
	ch := middleware.Channel{
		Ch:        rawCh,
		ClientID:  int(hello.ClientID),
		CleanAction: middleware.NotClean,
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	gamesRecv, gamesSend := net.Pipe()
	go func() {
		defer wg.Done()
		err := g.queueGames(gamesRecv, ch)
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
		defer wg.Done()
		err := g.queueReviews(reviewsRecv, ch)
		if err != nil {
			log.Errorf("Error while queuing reviews: %v", err)
		}
	}()
	err = g.receiveData(conn, reviewsSend)
	if err != nil {
		return err
	}
	reviewsSend.Close()

	wg.Wait()

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

// todo: refactor queue functions to avoid repeating code as they are almost
// identical, except in the data type

func (g *gateway) queueGames(r io.Reader, ch middleware.Channel) error {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1

	var sentGames int
	batch := middleware.Batch[middleware.Game]{
		Data:    []middleware.Game{},
		BatchID: 0,
		EOF:     false,
	}

	_, err := csvReader.Read()
	if err != nil {
		return err
	}

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
			err = ch.Send(batch, middleware.ExchangeGames, "")
			if err != nil {
				return err
			}
			batch.Data = batch.Data[:0]
			batch.BatchID += 1
		}
	}

	batch.EOF = true
	err = ch.Send(batch, middleware.ExchangeGames, "")
	if err != nil {
		return err
	}

	log.Infof("Finished sending %v games", sentGames)

	return nil
}

func (g *gateway) queueReviews(r io.Reader, ch middleware.Channel) error {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1

	var sentReviews int
	batch := middleware.Batch[middleware.Review]{
		Data:    []middleware.Review{},
		BatchID: 0,
		EOF:     false,
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
			err = ch.Send(batch, middleware.ExchangeReviews, "")
			if err != nil {
				return err
			}
			batch.Data = batch.Data[:0]
			batch.BatchID += 1
		}
	}

	batch.EOF = true
	err := ch.Send(batch, middleware.ExchangeReviews, "")
	if err != nil {
		return err
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
