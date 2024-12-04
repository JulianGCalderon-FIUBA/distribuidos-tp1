package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"errors"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type gateway struct {
	config        config
	rabbit        *amqp.Connection
	rabbitCh      *amqp.Channel
	mu            *sync.Mutex
	clients       map[int]chan protocol.Result
	clientCounter uint64
	db            *database.Database
	outputs       []middleware.Output
}

func newGateway(config config) *gateway {
	database_path := "gateway"
	db, err := database.NewDatabase(database_path)

	if err != nil {
		utils.Expect(err, "Could not create database")
	}

	protocol.Register()
	return &gateway{
		config:  config,
		clients: make(map[int]chan protocol.Result),
		mu:      &sync.Mutex{},
		db:      db,
		outputs: []middleware.Output{},
	}
}

func (g *gateway) start(ctx context.Context) error {
	conn, ch, err := middleware.Dial(g.config.RabbitIP)
	if err != nil {
		return err
	}
	closer := utils.SpawnCloser(ctx, conn)
	defer func() {
		closeErr := closer.Close()
		err = errors.Join(err, closeErr)
	}()
	g.rabbit = conn
	g.rabbitCh = ch

	err = g.declareTopology()
	if err != nil {
		return err
	}

	g.clientCounter, err = g.loadClientCounter()

	// clean all system resources if gateway has fallen
	err = g.notifyFallenNode(int(g.clientCounter), middleware.CleanAll)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		log.Infof("Starting requests endpoint")
		err := g.startRequestEndpoint(ctx)
		if err != nil {
			log.Errorf("Failed to run request endpoint: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := g.startDataEndpoint(ctx)
		if err != nil {
			log.Errorf("Failed to run data endpoint: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := g.startResultsEndpoint(ctx)
		if err != nil {
			log.Errorf("Failed to run results handler: %v", err)
		}
	}()
	wg.Wait()

	return nil
}

func (g *gateway) declareTopology() error {
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

	g.outputs = outputs

	rawCh, err := g.rabbit.Channel()
	if err != nil {
		return err
	}
	return topology.Declare(rawCh)
}

// sends a message through the pipeline to notify to clean resources for a single or all clients
// - if cleanAction == CleanAll -> the nodes will clean resources for all clients with id lower than clientID
// - if cleanAction == CleanId -> the nodes will clean resources for the particular clientID
func (g *gateway) notifyFallenNode(clientID int, cleanAction int) error {
	log.Infof("Sending CleanAll for id %v", clientID)
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

	for _, output := range g.outputs {
		for _, k := range output.Keys {
			err := ch.Send([]byte{}, output.Exchange, k)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
