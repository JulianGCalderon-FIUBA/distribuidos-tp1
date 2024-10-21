package middleware

import amqp "github.com/rabbitmq/amqp091-go"

type ExchangeConfig struct {
	Name string
	Type string
}

func (c ExchangeConfig) Declare(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		c.Name,
		c.Type,
		false,
		false,
		false,
		false,
		nil,
	)
}

type QueueConfig struct {
	Name     string
	Bindings map[string][]string
}

func (c QueueConfig) Declare(ch *amqp.Channel) error {
	name, err := ch.QueueDeclare(
		c.Name,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)

	for exchange, keys := range c.Bindings {
		for _, key := range keys {
			err = ch.QueueBind(name.Name, key, exchange, false, nil)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

type Topology struct {
	Exchanges []ExchangeConfig
	Queues    []QueueConfig
}

func (c Topology) Declare(ch *amqp.Channel) error {
	for _, exchange := range c.Exchanges {
		err := exchange.Declare(ch)
		if err != nil {
			return err
		}
	}

	for _, queue := range c.Queues {
		err := queue.Declare(ch)
		if err != nil {
			return err
		}
	}

	return nil
}
