package middleware

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type FilterConfig struct {
	RabbitIP string
	// Name of the queue to read from
	Queue string
	// Name of the exchange to declare
	Exchange string
	// Queues binded to each key
	QueuesByKey map[string][]string
}

type FilterFunc[T any] func(record T) []string

type filterHandler[T any] struct {
	input      string
	output     string
	clientID   int
	filter     FilterFunc[T]
	partitions map[string]Batch[T]
	stats      map[string]int
	sequencer  *Sequencer
}

func (h *filterHandler[T]) handle(ch *Channel, data []byte) error {
	batch, err := Deserialize[Batch[T]](data)
	if err != nil {
		return err
	}

	h.sequencer.Mark(batch.BatchID, batch.EOF)

	for _, record := range batch.Data {
		keys := h.filter(record)
		for _, key := range keys {
			entry := h.partitions[key]
			entry.Data = append(entry.Data, record)
			h.partitions[key] = entry
		}
	}

	for key, partition := range h.partitions {
		partition.BatchID = batch.BatchID
		partition.EOF = batch.EOF

		h.stats[key] += len(partition.Data)

		err := ch.Send(partition, h.output, key)
		if err != nil {
			return err
		}

		partition.Data = partition.Data[:0]
		h.partitions[key] = partition
	}

	if h.sequencer.EOF() {
		log.Infof("Received EOF from client %v", h.clientID)
		for rk, stats := range h.stats {
			log.Infof("Sent %v records to key %v", stats, rk)
		}
		ch.Finish()
	}

	return nil
}

func NewFilter[T any](config FilterConfig, f FilterFunc[T]) (*Node[*filterHandler[T]], error) {
	conn, ch, err := Dial(config.RabbitIP)
	if err != nil {
		return nil, err
	}

	exchangeConfig := ExchangeConfig{
		Name: config.Exchange,
		Type: amqp.ExchangeDirect,
	}

	allKeys := make([]string, 0)
	queueConfigs := make([]QueueConfig, 0)
	for queue, keys := range transpose(config.QueuesByKey) {
		queueConfig := QueueConfig{
			Name: queue,
			Bindings: map[string][]string{
				config.Exchange: keys,
			},
		}
		queueConfigs = append(queueConfigs, queueConfig)
		allKeys = append(allKeys, keys...)
	}
	queueConfigs = append(queueConfigs, QueueConfig{Name: config.Queue})

	outputConfig := Output{
		Exchange: config.Exchange,
		Keys:     allKeys,
	}

	err = Topology{
		Exchanges: []ExchangeConfig{exchangeConfig},
		Queues:    queueConfigs,
	}.Declare(ch)
	if err != nil {
		return nil, err
	}

	nConfig := Config[*filterHandler[T]]{
		Builder: func(clientID int) (*filterHandler[T], error) {
			partitions := make(map[string]Batch[T])
			for key := range config.QueuesByKey {
				partitions[key] = Batch[T]{}
			}

			return &filterHandler[T]{
				input:      config.Queue,
				output:     config.Exchange,
				clientID:   clientID,
				filter:     f,
				partitions: partitions,
				stats:      make(map[string]int),
				sequencer:  NewSequencer(),
			}, nil
		},
		Endpoints: map[string]HandlerFunc[*filterHandler[T]]{
			config.Queue: (*filterHandler[T]).handle,
		},
		OutputConfig: outputConfig,
	}

	return NewNode(nConfig, conn)
}

func transpose(queuesByKey map[string][]string) (keysByQueue map[string][]string) {
	keysByQueue = make(map[string][]string)

	for k, qs := range queuesByKey {
		for _, q := range qs {
			keysByQueue[q] = append(keysByQueue[q], k)
		}
	}

	return keysByQueue
}

func (h *filterHandler[T]) Free() error {
	return nil
}
