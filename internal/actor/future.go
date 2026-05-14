package actor

import (
	"context"
	"time"
)

type futureResult struct {
	value any
	err   error
}

type future struct {
	ch chan futureResult
}

func newFuture() *future {
	return &future{ch: make(chan futureResult, 1)}
}

func (f *future) reply(msg any) bool {
	select {
	case f.ch <- futureResult{value: msg}:
		return true
	default:
		return false
	}
}

func (f *future) replyError(err error) bool {
	select {
	case f.ch <- futureResult{err: err}:
		return true
	default:
		return false
	}
}

func (f *future) wait(timeout time.Duration) (any, error) {
	if timeout < 0 {
		r := <-f.ch
		return r.value, r.err
	}
	if timeout == 0 {
		select {
		case r := <-f.ch:
			return r.value, r.err
		default:
			return nil, &AskTimeoutError{Timeout: 0}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case r := <-f.ch:
		return r.value, r.err
	case <-ctx.Done():
		return nil, &AskTimeoutError{Timeout: timeout}
	}
}
