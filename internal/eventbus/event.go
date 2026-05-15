package eventbus

import "context"

type Event interface {
	Name() string
}

type Handler interface {
	Handle(ctx context.Context, event Event) error
}

type HandlerFunc func(ctx context.Context, event Event) error

func (f HandlerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

type Bus interface {
	Publish(ctx context.Context, event Event)
	Subscribe(eventName string, handler Handler)
}
