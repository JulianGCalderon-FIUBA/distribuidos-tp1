package filter

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/node"

	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// This interface contains business logic
type Handler[T any] interface {
	// Given a record T, returns which routing key it
	// corresponds to it, when sending it to the exchange
	Filter(record T) []string
}

type Config struct {
	RabbitIP string
	// Name of the queue to read from
	Queue string
	// Exchange to declare, and queues binded to them
	Exchange node.ExchangeConfig
}

// Filter structure, abstracting away queue system details
// Receives an input queue, a name, and a set of output queues,
// and builds the following topology.
//
// :              |--> Q1(K1)
// : [Q] --> [X]--|--> Q2(K2)
// :              |--> Q3(K2)
// :              |--> Q4(K3)
//
// Dispatches every row from Q to exchange X with routing key
// according to Filter function from Handler.
type Filter struct {
	node *node.Node
}

type handler[T any] struct {
	exchange   string
	partitions map[string]middleware.Batch[T]
	statistics map[string]int
	handler    Handler[T]
}

func (h *handler[T]) Apply(ch *middleware.Channel, data []byte) error {
	batch, err := middleware.Deserialize[middleware.Batch[T]](data)
	if err != nil {
		return err
	}

	for _, record := range batch.Data {
		keys := h.handler.Filter(record)
		for _, key := range keys {
			entry := h.partitions[key]
			entry.Data = append(entry.Data, record)
			h.partitions[key] = entry
		}
	}

	for key, partition := range h.partitions {
		partition.BatchID = batch.BatchID
		partition.ClientID = batch.ClientID
		partition.EOF = batch.EOF

		h.statistics[key] += len(partition.Data)

		err := ch.Send(partition, h.exchange, key)
		if err != nil {
			return err
		}

		partition.Data = partition.Data[:0]
		h.partitions[key] = partition
	}

	if batch.EOF {
		log.Info("Received EOF from client")
		for rk, stats := range h.statistics {
			log.Infof("Sent %v records to key %v", stats, rk)
		}
	}

	return nil
}

func NewFilter[T any](cfg Config, h Handler[T]) (*Filter, error) {
	nodeCfg := node.Config{
		RabbitIP:  cfg.RabbitIP,
		Input:     cfg.Queue,
		Exchanges: []node.ExchangeConfig{cfg.Exchange},
	}

	partitions := make(map[string]middleware.Batch[T])
	for key := range cfg.Exchange.QueuesByKey {
		partitions[key] = middleware.Batch[T]{}
	}

	nodeH := &handler[T]{
		exchange:   cfg.Exchange.Name,
		partitions: partitions,
		statistics: map[string]int{},
		handler:    h,
	}

	node, err := node.NewNode(nodeCfg, nodeH)
	if err != nil {
		return nil, err
	}

	return &Filter{
		node: node,
	}, nil
}

func (f *Filter) Run(ctx context.Context) error {
	return f.node.Run(ctx)
}
