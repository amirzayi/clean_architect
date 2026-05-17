package queue

import "context"

type Payload struct {
	Data any
}

type Driver interface {
	EnQueue(context.Context, Payload) error
	DeQueue(context.Context) (<-chan Payload, <-chan error)
}
