package middleware

import (
	logging "github.com/op/go-logging"
	amqp "github.com/rabbitmq/amqp091-go"
)

var log = logging.MustGetLogger("log")

type Channel struct {
	Ch          *amqp.Channel
	ClientID    int
	FinishFlag  bool
	CleanAction int
}

func (c *Channel) Send(msg any, exchange, key string) error {
	buf, err := Serialize(msg)
	if err != nil {
		log.Panicf("Failed to serialize result %v", err)
	}

	err = c.Ch.Publish(exchange, key, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "",
		Headers: amqp.Table{
			"clientID":    c.ClientID,
			"cleanAction": c.CleanAction,
		},
		Body: buf,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Channel) Finish() {
	c.FinishFlag = true
}

func (c *Channel) SendAny(msg any, exchange, key string) error {
	return c.Send(&msg, exchange, key)
}
