package middleware

type MessageTag int

const (
	RequestHelloTag MessageTag = iota
	AcceptRequestTag
	DataHelloTag
	DataAcceptTag
	GameBatchTag
	ReviewBatchTag
)

// Connection Handler Messages

// Sent by the client to initiate a request
type RequestHello struct {
	// todo: data size
}

// Sent by the connection handler to accept a client's request
type AcceptRequest struct {
	ClientID int
}

// Data Handler Messages

// Sent by the client to present itself to the data handler
type DataHello struct {
	ClientID int
}

// Sent by the data handler to accept a client
type DataAccept struct{}

// Sent by the client to the data handler
type GameBatch struct {
	Games []Game
}

// Sent by the client to the data handler
type ReviewBatch struct {
	Reviews []Review
}

// Sent by the client to indicate that it has finished sending data
type Finish struct{}

// Generic Structures

type Game struct {
	Id   int
	Name string
	// todo: add fields
}
type Review struct {
	AppID   int
	AppName string
	Text    string
	Score   int
}
