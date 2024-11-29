package middleware

// Each client should be handled completely independent. Therefore,
// our middleware can hide this detail to the business layer.
// As some handlers require to keep state, we can't use the same instance for
// each client. Therefore, we need a HandlerBuilder function that creates
// a new handler when a new client comes.
type HandlerBuilder[T Handler] func(clientID int) T

// Some nodes need to listen from multiple queues. To allow this, we define
// HandlerFunc, which represents the handling of a message of a particular
// queue. If the user needs to handle multiple queues, it must define
// multiple HandlerFuncs
type HandlerFunc[T Handler] func(h T, ch *Channel, data []byte) error

type Output struct {
	Exchange string
	Keys     []string
}

type Handler interface {
	Free() error
}
