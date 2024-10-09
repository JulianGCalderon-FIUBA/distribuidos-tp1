package joiner

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
	// Called with each message of type T
	Aggregate(ch *middleware.Channel, data T) error
	// Called when EOF is reached.
	Conclude(ch *middleware.Channel) error
}

type Config struct {
	RabbitIP string
	// Name of the queue to read from
	Input string
	// Name of the queue to send result to
	Output string
	// Number of partitions to join
	Partitions int
}

// Joiner structure, abstracting away queue system details
// Receives an input queue, and process each partial result
// received until `PartitionsNumber` is met
type Joiner struct {
	node *node.Node
}

type handler[T any] struct {
	missing int
	handler Handler[T]
}

func (h *handler[T]) Apply(ch *middleware.Channel, data []byte) error {
	msg, err := middleware.Deserialize[T](data)
	if err != nil {
		return err
	}

	log.Infof("Received partial result")

	err = h.handler.Aggregate(ch, msg)
	if err != nil {
		return err
	}

	h.missing -= 1
	if h.missing == 0 {
		log.Infof("Received all results")
		return h.handler.Conclude(ch)
	}
	return nil
}

func NewJoiner[T any](cfg Config, h Handler[T]) (*Joiner, error) {
	nodeCfg := node.Config{
		RabbitIP: cfg.RabbitIP,
		Exchanges: []node.ExchangeConfig{{
			Name: "",
			Type: amqp.ExchangeDirect,
			QueuesByKey: map[string][]string{
				cfg.Output: {cfg.Output},
			},
		}},
		Input: cfg.Input,
	}

	nodeH := &handler[T]{
		handler: h,
		missing: cfg.Partitions,
	}

	node, err := node.NewNode(nodeCfg, nodeH)
	if err != nil {
		return nil, err
	}

	return &Joiner{
		node: node,
	}, nil
}

func (f *Joiner) Run(ctx context.Context) error {
	return f.node.Run(ctx)
}
