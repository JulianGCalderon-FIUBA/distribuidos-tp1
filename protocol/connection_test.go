package protocol_test

import (
	"bytes"
	"distribuidos/tp1/protocol"
	"reflect"
	"testing"
)

type BufferCloser struct {
	bytes.Buffer
}

func (b *BufferCloser) Close() error {
	return nil
}

func TestConnAny(t *testing.T) {
	messages := []any{
		protocol.RequestHello{
			GameSize:   1,
			ReviewSize: 2,
		},
		protocol.RequestHello{
			GameSize:   3,
			ReviewSize: 4,
		},
		protocol.AcceptRequest{
			ClientID: 5,
		},
		protocol.AcceptRequest{
			ClientID: 6,
		},
		protocol.DataHello{
			ClientID: 7,
		},
		protocol.DataAccept{},
		protocol.Batch{Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		protocol.Finish{},
	}

	var b BufferCloser
	protocol.Register()
	conn := protocol.NewConn(&b)

	for _, msg := range messages {
		err := conn.SendAny(msg)
		if err != nil {
			t.Fatalf("failed to send message %v", err)
		}

		var recv_msg any
		err = conn.Recv(&recv_msg)
		if err != nil {
			t.Fatalf("failed to receive message %v", err)
		}

		if !reflect.DeepEqual(msg, recv_msg) {
			t.Fatalf("expected %v, but received %v", msg, recv_msg)
		}
	}
}

func TestConn(t *testing.T) {
	msg := protocol.RequestHello{
		GameSize:   1,
		ReviewSize: 2,
	}

	var b BufferCloser
	protocol.Register()
	conn := protocol.NewConn(&b)

	err := conn.Send(msg)
	if err != nil {
		t.Fatalf("failed to send message %v", err)
	}

	var recv_msg protocol.RequestHello
	err = conn.Recv(&recv_msg)
	if err != nil {
		t.Fatalf("failed to receive message %v", err)
	}

	if !reflect.DeepEqual(msg, recv_msg) {
		t.Fatalf("expected %v, but received %v", msg, recv_msg)
	}
}
