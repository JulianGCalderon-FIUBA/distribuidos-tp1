package middleware

import (
	"fmt"
)

func (m *Middleware) ReceiveBatch(queueName string) ([]byte, error) {
	msgs, err := m.ch.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("could not consume messages: %w", err)
	}

	var batch []byte
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			batch = append(batch, d.Body...)

			forever <- true
			break
		}
	}()

	<-forever
	return batch, nil
}
