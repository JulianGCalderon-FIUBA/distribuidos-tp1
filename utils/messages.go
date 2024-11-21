package utils

// Should it be here? Can't add it to protocol bc circular dependence with middleware

type MsgType uint8

const (
	Ack       MsgType = 'A'
	KeepAlive MsgType = 'K'
)

// Sent between restarter node and normal nodes
type MsgHeader struct {
	Ty MsgType
}
