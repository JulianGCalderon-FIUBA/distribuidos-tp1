package utils

import (
	"context"
	"io"
)

type Closer struct {
	ch chan error
}

// Spawns a closer for the given resource. The closer funtion will be called
// - When the context finalizes
// - When the `Close` method is called
// Guarantees that the close method is called only once.
func SpawnCloser[Resource io.Closer](ctx context.Context, resource Resource) Closer {
	ch := make(chan error)
	go func() {
		select {
		case <-ctx.Done():
			err := resource.Close()
			<-ch
			ch <- err
		case <-ch:
			ch <- resource.Close()
		}
	}()

	return Closer{ch}
}

// Notifies the Closer to close it's associated resource, if it hasn't closed it yet.
// Returns the error, if any, of executing the closer function.
func (c Closer) Close() error {
	c.ch <- nil
	return <-c.ch
}
