package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
)

type resultsHandler struct {
	ch      chan protocol.Results
	results int
}

func (h *resultsHandler) handle(ch *middleware.Channel, data []byte) error {
	result, err := middleware.Deserialize[protocol.Results](data)
	if err != nil {
		return err
	}

	log.Infof("Received results")

	switch r := result.(type) {
	case protocol.Q1Results:
		h.results += 1
	case protocol.Q2Results:
		h.results += 1
	case protocol.Q3Results:
		h.results += 1
	case protocol.Q4Results:
		if r.EOF {
			h.results += 1
		}
	case protocol.Q5Results:
		h.results += 1
	}
	h.ch <- result

	if h.results == MAX_RESULTS {
		log.Infof("Received all results")
		close(h.ch)
		return nil
	}

	return nil
}

func (g *gateway) startResultsEndpoint(ctx context.Context) error {
	newResultsHandler := func(clientID int) resultsHandler {
		g.mu.Lock()
		chanResults := g.clients[clientID]
		g.mu.Unlock()

		return resultsHandler{
			ch: chanResults,
		}
	}

	topology := middleware.Topology{
		Queues: []middleware.QueueConfig{{Name: middleware.Results}},
	}
	err := topology.Declare(g.rabbitCh)
	if err != nil {
		return err
	}

	cfg := middleware.Config[resultsHandler]{
		Builder: newResultsHandler,
		Endpoints: map[string]middleware.HandlerFunc[resultsHandler]{
			middleware.Results: (*resultsHandler).handle,
		},
	}

	node, err := middleware.NewNode(cfg, g.rabbit)
	if err != nil {
		return err
	}

	return node.Run(ctx)
}
