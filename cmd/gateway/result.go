package main

import (
	"context"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"errors"
)

const MAX_RESULTS = 5

type resultsHandler struct {
	ch        chan protocol.Result
	results   map[int]bool
	sequencer *middleware.Sequencer
}

func (h *resultsHandler) handle(ch *middleware.Channel, data []byte) error {
	result, err := middleware.Deserialize[protocol.Result](data)
	if err != nil {
		return err
	}
	if h.results[result.Number()] {
		return nil
	}

	log.Infof("Sending Q%v results", result.Number())

	h.ch <- result
	h.results[result.Number()] = true

	if len(h.results) == MAX_RESULTS {
		close(h.ch)
		return nil
	}

	return nil
}

func (h *resultsHandler) handleQ4(ch *middleware.Channel, data []byte) error {
	batch, err := middleware.Deserialize[middleware.Batch[middleware.GameStat]](data)
	if err != nil {
		return err
	}

	if h.sequencer.Seen(batch.BatchID) {
		return nil
	}

	h.sequencer.Mark(batch.BatchID, batch.EOF)

	if len(batch.Data) > 0 {
		log.Infof("Sending Q4 results")
		r := protocol.Q4Result{Games: batch.Data}
		h.ch <- r
	}

	if h.sequencer.EOF() {
		r := protocol.Q4Finish{}
		h.ch <- r
		h.results[r.Number()] = true
	}

	if len(h.results) == MAX_RESULTS {
		close(h.ch)
		return nil
	}

	return nil
}

func (h *resultsHandler) Free() error {
	return nil
}

func (g *gateway) startResultsEndpoint(ctx context.Context) error {
	newResultsHandler := func(clientID int) (*resultsHandler, error) {
		g.mu.Lock()
		chanResults, exists := g.clients[clientID]
		g.mu.Unlock()
		if !exists {
			return nil, errors.New("client does not exist")
		}

		return &resultsHandler{
			ch:        chanResults,
			results:   make(map[int]bool),
			sequencer: middleware.NewSequencer(),
		}, nil
	}

	topology := middleware.Topology{
		Queues: []middleware.QueueConfig{{Name: middleware.Results}, {Name: middleware.ResultsQ4}},
	}
	err := topology.Declare(g.rabbitCh)
	if err != nil {
		return err
	}

	cfg := middleware.Config[*resultsHandler]{
		Builder: newResultsHandler,
		Endpoints: map[string]middleware.HandlerFunc[*resultsHandler]{
			middleware.Results:   (*resultsHandler).handle,
			middleware.ResultsQ4: (*resultsHandler).handleQ4,
		},
	}

	node, err := middleware.NewNode(cfg, g.rabbit)
	if err != nil {
		return err
	}

	return node.Run(ctx)
}
