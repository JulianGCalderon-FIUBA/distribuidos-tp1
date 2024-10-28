package protocol

import (
	"encoding/gob"
	"io"
)

type Conn struct {
	conn io.ReadWriteCloser
	enco *gob.Encoder
	deco *gob.Decoder
}

func Register() {
	gob.Register(Batch{})
	gob.Register(Finish{})
	gob.Register(Q1Result{})
	gob.Register(Q2Result{})
	gob.Register(Q3Result{})
	gob.Register(Q4Result{})
	gob.Register(Q5Result{})
}

func NewConn(conn io.ReadWriteCloser) *Conn {
	return &Conn{
		conn: conn,
		enco: gob.NewEncoder(conn),
		deco: gob.NewDecoder(conn),
	}
}

// Send as interface, can be decoded as interface
//
// Used when receiver expects different type of messages
func (c *Conn) SendAny(msg any) error {
	return c.enco.Encode(&msg)
}

// Send as concrete, can be decoded as concrete
//
// Used when receiver expects only this specific type
func (c *Conn) Send(msg any) error {
	return c.enco.Encode(msg)
}

// Must receive pointer to concrete or interface
func (c *Conn) Recv(msg any) error {
	return c.deco.Decode(msg)
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
