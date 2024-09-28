package middleware_test

import (
	"bytes"
	"distribuidos/tp1/middleware"
	"reflect"
	"testing"
)

func TestMarshaller(t *testing.T) {
	messages := []middleware.Message{
		&middleware.RequestHello{
			GameSize:   1,
			ReviewSize: 2,
		},
		&middleware.RequestHello{
			GameSize:   3,
			ReviewSize: 4,
		},
		&middleware.AcceptRequest{
			ClientID: 5,
		},
		&middleware.AcceptRequest{
			ClientID: 6,
		},
		&middleware.DataHello{
			ClientID: 7,
		},
		&middleware.DataAccept{},
		// &middleware.GameBatch{},
		// &middleware.ReviewBatch{},
		&middleware.Finish{},
	}

	var b bytes.Buffer
	marshaller := middleware.NewMarshaller(&b)
	unmarshaller := middleware.NewUnmarshaller(&b)

	for _, message := range messages {
		err := marshaller.SendMessage(message)
		if err != nil {
			t.Fatalf("failed to send message %v", err)
		}

		received_message, err := unmarshaller.ReceiveMessage()
		if err != nil {
			t.Fatalf("failed to receive message %v", err)
		}

		if !reflect.DeepEqual(message, received_message) {
			t.Fatalf("expected %v, but received %v", message, received_message)

		}
	}
}
