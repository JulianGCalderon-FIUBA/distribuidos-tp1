package restarter_test

import (
	"distribuidos/tp1/restarter-protocol"
	"reflect"
	"testing"
)

func TestSerializeElection(t *testing.T) {
	e := restarter.Election{
		Ids: []uint64{1, 2, 3},
	}

	buf, err := e.Encode(nil)
	if err != nil {
		t.Fatalf("Failed to encode election msg: %v", err)
	}
	recv_e, err := restarter.DecodeElection(buf)
	if err != nil {
		t.Fatalf("Failed to decode election msg: %v", err)
	}

	if !reflect.DeepEqual(e, recv_e) {
		t.Fatalf("Expected %v, but received %v", e, recv_e)
	}
}

func TestSerializeCoordinator(t *testing.T) {
	c := restarter.Coordinator{
		Leader: 3,
		Ids:    []uint64{1, 2, 3},
	}

	buf, err := c.Encode(nil)
	if err != nil {
		t.Fatalf("Failed to encode coordinator msg: %v", err)
	}
	recv_c, err := restarter.DecodeCoordinator(buf)
	if err != nil {
		t.Fatalf("Failed to decode coordinator msg: %v", err)
	}

	if !reflect.DeepEqual(c, recv_c) {
		t.Fatalf("Expected %v, but received %v", c, recv_c)
	}
}

func TestSerializePacket(t *testing.T) {
	packetList := []restarter.Packet{
		{
			Id: 1,
			Msg: restarter.Election{
				Ids: []uint64{1, 2, 3},
			},
		},
		{
			Id: 1,
			Msg: restarter.Coordinator{
				Leader: 3,
				Ids:    []uint64{1, 2, 3},
			},
		},
		{
			Id:  1,
			Msg: restarter.Ack{},
		}}

	for _, p := range packetList {

		buf, err := p.Encode()
		if err != nil {
			t.Fatalf("Failed to encode packet: %v", err)
		}

		recv_p, err := restarter.Decode(buf)
		if err != nil {
			t.Fatalf("Failed to encode packet: %v", err)
		}

		if !reflect.DeepEqual(p, recv_p) {
			t.Fatalf("Expected %v, but received %v", p, recv_p)
		}
	}
}
