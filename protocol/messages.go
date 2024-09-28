package protocol

type MessageTag uint8

const (
	RequestHelloTag MessageTag = iota
	AcceptRequestTag
	DataHelloTag
	DataAcceptTag
	GameBatchTag
	ReviewBatchTag
	FinishTag
)

// Connection Handler Messages

// Sent by the client to initiate a request
type RequestHello struct {
	GameSize   uint64
	ReviewSize uint64
}

// Sent by the connection handler to accept a client's request
type AcceptRequest struct {
	ClientID uint64
}

// Data Handler Messages

// Sent by the client to present itself to the data handler
type DataHello struct {
	ClientID uint64
}

// Sent by the data handler to accept a client
type DataAccept struct{}

// Sent by the client to the data handler
type GameBatch struct {
	Games [][]string
}

// Sent by the client to the data handler
type ReviewBatch struct {
	Reviews [][]string
}

// Sent by the client to indicate that it has finished sending data
type Finish struct{}