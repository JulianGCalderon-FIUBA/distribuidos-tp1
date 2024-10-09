package aggregator

import (
	"context"
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/server/middleware/node"

	logging "github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
)

var log = logging.MustGetLogger("log")

// This interface contains business logic
type Handler[T any] interface {
	// Called with each batch of type T
	Aggregate(ch *middleware.Channel, batch middleware.Batch[T]) error
	// Called when EOF is reached.
	Conclude(ch *middleware.Channel) error
}

type Config struct {
	RabbitIP string
	// Queue to read from
	Queue string
	// Output queue to declare
	Output string
}

type Aggregator struct {
	node *node.Node
}

type handler[T any] struct {
	missingBatchIDs map[int]struct{}
	latestBatchID   int
	receivedEof     bool
	handler         Handler[T]
}

func (h *handler[T]) Apply(ch *middleware.Channel, data []byte) error {
	batch, err := middleware.Deserialize[middleware.Batch[T]](data)
	if err != nil {
		return err
	}
	delete(h.missingBatchIDs, batch.BatchID)
	for i := h.latestBatchID + 1; i < batch.BatchID; i++ {
		h.missingBatchIDs[i] = struct{}{}
	}
	h.latestBatchID = max(batch.BatchID, h.latestBatchID)
	if batch.EOF {
		h.receivedEof = true
	}

	err = h.handler.Aggregate(ch, batch)
	if err != nil {
		return err
	}

	if h.receivedEof && len(h.missingBatchIDs) == 0 {
		log.Info("Received EOF from client")
		return h.handler.Conclude(ch)
	}
	return nil
}

func NewAggregator[T any](cfg Config, h Handler[T]) (*Aggregator, error) {

	nodeCfg := node.Config{
		RabbitIP: cfg.RabbitIP,
		Exchanges: []node.ExchangeConfig{{
			Name: "",
			Type: amqp.ExchangeDirect,
			QueuesByKey: map[string][]string{
				cfg.Output: {cfg.Output},
			},
		}},
		Queue: cfg.Queue,
	}

	nodeH := &handler[T]{
		missingBatchIDs: make(map[int]struct{}),
		latestBatchID:   0,
		receivedEof:     false,
		handler:         h,
	}

	node, err := node.NewNode(nodeCfg, nodeH)
	if err != nil {
		return nil, err
	}

	return &Aggregator{
		node: node,
	}, nil
}

func (a *Aggregator) Run(ctx context.Context) error {
	return a.node.Run(ctx)
}
