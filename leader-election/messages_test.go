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

	buf, err := e.Encode(nil)
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

	buf, err := c.Encode(nil)
	if err != nil {
		t.Fatalf("Failed to encode coordinator msg: %v", err)
	}

	recv_c := leaderelection.CoordinatorMsg{}
	err = recv_c.Decode(buf)
	if err != nil {
		t.Fatalf("Failed to decode coordinator msg: %v", err)
	}

	if !reflect.DeepEqual(c, recv_c) {
		t.Fatalf("Expected %v, but received %v", c, recv_c)
	}
}

func TestSerializeElectionWithHeader(t *testing.T) {
	e := leaderelection.ElectionMsg{
		Ids: []uint64{1, 2, 3},
	}
	h := leaderelection.MsgHeader{
		Ty: e.Type(),
		Id: 1,
	}

	buf, err := e.EncodeWithHeader(1)
	if err != nil {
		t.Fatalf("Failed to encode election msg: %v", err)
	}

	recv_e := leaderelection.ElectionMsg{}
	recv_h, err := recv_e.DecodeWithHeader(buf)
	if err != nil {
		t.Fatalf("Failed to decode election msg: %v", err)
	}

	if !reflect.DeepEqual(e, recv_e) {
		t.Fatalf("Expected %v, but received %v", e, recv_e)
	}
	if !reflect.DeepEqual(h, recv_h) {
		t.Fatalf("Expected %v, but received %v", h, recv_h)
	}
}

func TestSerializeCoordinatorWithHeader(t *testing.T) {
	c := leaderelection.CoordinatorMsg{
		Leader: 3,
		Ids:    []uint64{1, 2, 3},
	}
	h := leaderelection.MsgHeader{
		Ty: c.Type(),
		Id: 1,
	}

	buf, err := c.EncodeWithHeader(1)
	if err != nil {
		t.Fatalf("Failed to encode coordinator msg: %v", err)
	}

	recv_c := leaderelection.CoordinatorMsg{}
	recv_h, err := recv_c.DecodeWithHeader(buf)
	if err != nil {
		t.Fatalf("Failed to decode coordinator msg: %v", err)
	}

	if !reflect.DeepEqual(c, recv_c) {
		t.Fatalf("Expected %v, but received %v", c, recv_c)
	}
	if !reflect.DeepEqual(h, recv_h) {
		t.Fatalf("Expected %v, but received %v", h, recv_h)
	}
}

func TestSerializeAckWithHeader(t *testing.T) {
	a := leaderelection.AckMsg{}
	h := leaderelection.MsgHeader{
		Ty: a.Type(),
		Id: 1,
	}

	buf, err := a.EncodeWithHeader(1)
	if err != nil {
		t.Fatalf("Failed to encode ack msg: %v", err)
	}

	recv_a := leaderelection.AckMsg{}
	recv_h, err := recv_a.DecodeWithHeader(buf)
	if err != nil {
		t.Fatalf("Failed to decode ack msg: %v", err)
	}

	if !reflect.DeepEqual(a, recv_a) {
		t.Fatalf("Expected %v, but received %v", a, recv_a)
	}
	if !reflect.DeepEqual(h, recv_h) {
		t.Fatalf("Expected %v, but received %v", h, recv_h)
	}
}
