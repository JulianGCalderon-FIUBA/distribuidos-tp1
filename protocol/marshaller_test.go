package protocol_test

import (
	"bytes"
	"distribuidos/tp1/protocol"
	"reflect"
	"testing"
)

func TestMarshaller(t *testing.T) {
	messages := []protocol.Message{
		&protocol.RequestHello{
			GameSize:   1,
			ReviewSize: 2,
		},
		&protocol.RequestHello{
			GameSize:   3,
			ReviewSize: 4,
		},
		&protocol.AcceptRequest{
			ClientID: 5,
		},
		&protocol.AcceptRequest{
			ClientID: 6,
		},
		&protocol.DataHello{
			ClientID: 7,
		},
		&protocol.DataAccept{},
		&protocol.GameBatch{Games: [][]byte{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {10, 11, 12, 13, 14, 15}}},
		&protocol.ReviewBatch{Reviews: [][]byte{{0, 0, 23, 80}, {40, 32, 99, 100}}},
		&protocol.Finish{},
	}

	var b bytes.Buffer
	marshaller := protocol.NewMarshaller(&b)
	unmarshaller := protocol.NewUnmarshaller(&b)

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
