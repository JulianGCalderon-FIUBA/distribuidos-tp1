package leaderelection_test

import (
	leaderelection "distribuidos/tp1/leader-election"
	"reflect"
	"testing"
)

func TestSerializeElection(t *testing.T) {
	e := leaderelection.ElectionMsg{
		Ids: []uint64{1, 2, 3},
	}

	buf, err := e.Encode()
	if err != nil {
		t.Fatalf("Failed to encode election msg: %v", err)
	}

	recv_e := leaderelection.ElectionMsg{}
	err = recv_e.Decode(buf)
	if err != nil {
		t.Fatalf("Failed to decode election msg: %v", err)
	}

	if !reflect.DeepEqual(e, recv_e) {
		t.Fatalf("Expected %v, but received %v", e, recv_e)
	}
}

func TestSerializeCoordinator(t *testing.T) {
	c := leaderelection.CoordinatorMsg{
		Leader: 3,
		Ids:    []uint64{1, 2, 3},
	}

	buf, err := c.Encode()
	if err != nil {
		t.Fatalf("Failed to encode election msg: %v", err)
	}

	recv_c := leaderelection.CoordinatorMsg{}
	err = recv_c.Decode(buf)
	if err != nil {
		t.Fatalf("Failed to decode election msg: %v", err)
	}

	if !reflect.DeepEqual(c, recv_c) {
		t.Fatalf("Expected %v, but received %v", c, recv_c)
	}
}
