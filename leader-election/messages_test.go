package leaderelection_test

import (
	leaderelection "distribuidos/tp1/leader-election"
	"reflect"
	"testing"
)

func TestSerializeElection(t *testing.T) {
	e := leaderelection.Election{
		Ids: []uint64{1, 2, 3},
	}

	buf, err := e.Encode(nil)
	if err != nil {
		t.Fatalf("Failed to encode election msg: %v", err)
	}
	recv_e, err := leaderelection.DecodeElection(buf)
	if err != nil {
		t.Fatalf("Failed to decode election msg: %v", err)
	}

	if !reflect.DeepEqual(e, recv_e) {
		t.Fatalf("Expected %v, but received %v", e, recv_e)
	}
}

func TestSerializeCoordinator(t *testing.T) {
	c := leaderelection.Coordinator{
		Leader: 3,
		Ids:    []uint64{1, 2, 3},
	}

	buf, err := c.Encode(nil)
	if err != nil {
		t.Fatalf("Failed to encode coordinator msg: %v", err)
	}
	recv_c, err := leaderelection.DecodeCoordinator(buf)
	if err != nil {
		t.Fatalf("Failed to decode coordinator msg: %v", err)
	}

	if !reflect.DeepEqual(c, recv_c) {
		t.Fatalf("Expected %v, but received %v", c, recv_c)
	}
}

func TestSerializePacket(t *testing.T) {
	packetList := []leaderelection.Packet{
		{
			Id: 1,
			Msg: leaderelection.Election{
				Ids: []uint64{1, 2, 3},
			},
		},
		{
			Id: 1,
			Msg: leaderelection.Coordinator{
				Leader: 3,
				Ids:    []uint64{1, 2, 3},
			},
		},
		{
			Id:  1,
			Msg: leaderelection.Ack{},
		}}

	for _, p := range packetList {

		buf, err := p.Encode()
		if err != nil {
			t.Fatalf("Failed to encode packet: %v", err)
		}

		recv_p, err := leaderelection.Decode(buf)
		if err != nil {
			t.Fatalf("Failed to encode packet: %v", err)
		}

		if !reflect.DeepEqual(p, recv_p) {
			t.Fatalf("Expected %v, but received %v", p, recv_p)
		}
	}
}
